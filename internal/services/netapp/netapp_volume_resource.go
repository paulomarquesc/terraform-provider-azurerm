package netapp

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/netapp/mgmt/2021-10-01/netapp"
	"github.com/hashicorp/terraform-provider-azurerm/helpers/azure"
	"github.com/hashicorp/terraform-provider-azurerm/helpers/tf"
	"github.com/hashicorp/terraform-provider-azurerm/internal/clients"
	"github.com/hashicorp/terraform-provider-azurerm/internal/services/netapp/parse"
	"github.com/hashicorp/terraform-provider-azurerm/internal/tags"
	"github.com/hashicorp/terraform-provider-azurerm/internal/tf/pluginsdk"
	"github.com/hashicorp/terraform-provider-azurerm/internal/timeouts"
	"github.com/hashicorp/terraform-provider-azurerm/utils"
)

func resourceNetAppVolume() *pluginsdk.Resource {
	return &pluginsdk.Resource{
		Create: resourceNetAppVolumeCreate,
		Read:   resourceNetAppVolumeRead,
		Update: resourceNetAppVolumeUpdate,
		Delete: resourceNetAppVolumeDelete,

		Timeouts: &pluginsdk.ResourceTimeout{
			Create: pluginsdk.DefaultTimeout(60 * time.Minute),
			Read:   pluginsdk.DefaultTimeout(5 * time.Minute),
			Update: pluginsdk.DefaultTimeout(60 * time.Minute),
			Delete: pluginsdk.DefaultTimeout(60 * time.Minute),
		},
		Importer: pluginsdk.ImporterValidatingResourceId(func(id string) error {
			_, err := parse.VolumeID(id)
			return err
		}),

		Schema: netAppVolumeSchema(),
	}
}

func resourceNetAppVolumeCreate(d *pluginsdk.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).NetApp.VolumeClient
	subscriptionId := meta.(*clients.Client).Account.SubscriptionId
	ctx, cancel := timeouts.ForCreateUpdate(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id := parse.NewVolumeID(subscriptionId, d.Get("resource_group_name").(string), d.Get("account_name").(string), d.Get("pool_name").(string), d.Get("name").(string))
	if d.IsNewResource() {
		existing, err := client.Get(ctx, id.ResourceGroup, id.NetAppAccountName, id.CapacityPoolName, id.Name)
		if err != nil {
			if !utils.ResponseWasNotFound(existing.Response) {
				return fmt.Errorf("checking for presence of existing %s: %+v", id, err)
			}
		}
		if !utils.ResponseWasNotFound(existing.Response) {
			return tf.ImportAsExistsError("azurerm_netapp_volume", id.ID())
		}
	}

	location := azure.NormalizeLocation(d.Get("location").(string))

	volumeProperties, err := getVolumePropertiesObject[netapp.VolumeProperties](ctx, d, meta, id)
	if err != nil {
		return err
	}

	parameters := netapp.Volume{
		Location:         utils.String(location),
		VolumeProperties: volumeProperties,
		Tags:             tags.Expand(d.Get("tags").(map[string]interface{})),
	}

	if throughputMibps, ok := d.GetOk("throughput_in_mibps"); ok {
		parameters.VolumeProperties.ThroughputMibps = utils.Float(throughputMibps.(float64))
	}

	future, err := client.CreateOrUpdate(ctx, parameters, id.ResourceGroup, id.NetAppAccountName, id.CapacityPoolName, id.Name)
	if err != nil {
		return fmt.Errorf("creating %s: %+v", id, err)
	}
	if err = future.WaitForCompletionRef(ctx, client.Client); err != nil {
		return fmt.Errorf("waiting for the creation of %s: %+v", id, err)
	}

	// Waiting for volume be completely provisioned
	if err := waitForVolumeCreateOrUpdate(ctx, client, id); err != nil {
		return err
	}

	// If this is a data replication secondary volume, authorize replication on primary volume
	if *volumeProperties.VolumeType == string("DataProtection") {
		replVolID, err := parse.VolumeID(*volumeProperties.DataProtection.Replication.RemoteVolumeResourceID)
		if err != nil {
			return err
		}

		future, err := client.AuthorizeReplication(
			ctx,
			replVolID.ResourceGroup,
			replVolID.NetAppAccountName,
			replVolID.CapacityPoolName,
			replVolID.Name,
			netapp.AuthorizeRequest{
				RemoteVolumeResourceID: utils.String(id.ID()),
			},
		)
		if err != nil {
			return fmt.Errorf("cannot authorize volume replication: %v", err)
		}

		if err = future.WaitForCompletionRef(ctx, client.Client); err != nil {
			return fmt.Errorf("cannot get authorize volume replication future response: %v", err)
		}

		// Wait for volume replication authorization to complete
		log.Printf("[DEBUG] Waiting for replication authorization on NetApp Volume Provisioning Service %q (Resource Group %q) to complete", id.Name, id.ResourceGroup)
		if err := waitForReplAuthorization(ctx, client, id); err != nil {
			return err
		}
	}

	d.SetId(id.ID())

	return resourceNetAppVolumeRead(d, meta)
}

func resourceNetAppVolumeUpdate(d *pluginsdk.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).NetApp.VolumeClient
	ctx, cancel := timeouts.ForCreateUpdate(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id, err := parse.VolumeID(d.Id())
	if err != nil {
		return err
	}

	shouldUpdate := false
	update := netapp.VolumePatch{
		VolumePatchProperties: &netapp.VolumePatchProperties{},
	}

	if d.HasChange("storage_quota_in_gb") {
		shouldUpdate = true
		storageQuotaInBytes := int64(d.Get("storage_quota_in_gb").(int) * 1073741824)
		update.VolumePatchProperties.UsageThreshold = utils.Int64(storageQuotaInBytes)
	}

	if d.HasChange("export_policy_rule") {
		shouldUpdate = true
		exportPolicyRuleRaw := d.Get("export_policy_rule").([]interface{})
		exportPolicyRule := expandNetAppVolumeExportPolicyRule[netapp.VolumePatchPropertiesExportPolicy](exportPolicyRuleRaw)
		update.VolumePatchProperties.ExportPolicy = exportPolicyRule
	}

	if d.HasChange("data_protection_snapshot_policy") {
		// Validating that snapshot policies are not being created in a data protection volume
		dataProtectionReplicationRaw := d.Get("data_protection_replication").([]interface{})
		dataProtectionReplication := expandNetAppVolumeDataProtectionReplication(dataProtectionReplicationRaw)

		if dataProtectionReplication != nil && dataProtectionReplication.Replication != nil && strings.ToLower(string(dataProtectionReplication.Replication.EndpointType)) == "dst" {
			return fmt.Errorf("snapshot policy cannot be enabled on a data protection volume, NetApp Volume %q (Resource Group %q)", id.Name, id.ResourceGroup)
		}

		shouldUpdate = true
		dataProtectionSnapshotPolicyRaw := d.Get("data_protection_snapshot_policy").([]interface{})
		dataProtectionSnapshotPolicy := expandNetAppVolumeDataProtectionSnapshotPolicyPatch(dataProtectionSnapshotPolicyRaw)
		update.VolumePatchProperties.DataProtection = dataProtectionSnapshotPolicy
	}

	if d.HasChange("throughput_in_mibps") {
		shouldUpdate = true
		throughputMibps := d.Get("throughput_in_mibps")
		update.VolumePatchProperties.ThroughputMibps = utils.Float(throughputMibps.(float64))
	}

	if d.HasChange("tags") {
		shouldUpdate = true
		tagsRaw := d.Get("tags").(map[string]interface{})
		update.Tags = tags.Expand(tagsRaw)
	}

	if shouldUpdate {
		future, err := client.Update(ctx, update, id.ResourceGroup, id.NetAppAccountName, id.CapacityPoolName, id.Name)
		if err != nil {
			return fmt.Errorf("updating Volume %q: %+v", id.Name, err)
		}
		if err = future.WaitForCompletionRef(ctx, client.Client); err != nil {
			return fmt.Errorf("waiting for the update of %s: %+v", id, err)
		}

		// Wait for volume to complete update
		if err := waitForVolumeCreateOrUpdate(ctx, client, *id); err != nil {
			return err
		}
	}

	return resourceNetAppVolumeRead(d, meta)
}

func resourceNetAppVolumeRead(d *pluginsdk.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).NetApp.VolumeClient
	ctx, cancel := timeouts.ForRead(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id, err := parse.VolumeID(d.Id())
	if err != nil {
		return err
	}

	resp, err := client.Get(ctx, id.ResourceGroup, id.NetAppAccountName, id.CapacityPoolName, id.Name)
	if err != nil {
		if utils.ResponseWasNotFound(resp.Response) {
			log.Printf("[INFO] %s was not found - removing from state", *id)
			d.SetId("")
			return nil
		}
		return fmt.Errorf("reading %s: %+v", *id, err)
	}

	// Setting general volume properties
	d.Set("name", id.Name)
	d.Set("resource_group_name", id.ResourceGroup)
	d.Set("account_name", id.NetAppAccountName)
	d.Set("pool_name", id.CapacityPoolName)
	if location := resp.Location; location != nil {
		d.Set("location", azure.NormalizeLocation(*location))
	}

	// Setting Volume properties
	if props := resp.VolumeProperties; props != nil {
		if err := setVolumePropertiesResourceData(d, *props); err != nil {
			return err
		}
	}

	return tags.FlattenAndSet(d, resp.Tags)
}

func resourceNetAppVolumeDelete(d *pluginsdk.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).NetApp.VolumeClient
	ctx, cancel := timeouts.ForDelete(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id, err := parse.VolumeID(d.Id())
	if err != nil {
		return err
	}

	// Removing replication if present
	dataProtectionReplicationRaw := d.Get("data_protection_replication").([]interface{})
	dataProtectionReplication := expandNetAppVolumeDataProtectionReplication(dataProtectionReplicationRaw)

	if replicaVolumeId := id; dataProtectionReplication != nil && dataProtectionReplication.Replication != nil {
		if dataProtectionReplication.Replication.RemoteVolumeResourceID == nil {
			return fmt.Errorf("remote volume id was nil")
		}

		if strings.ToLower(string(dataProtectionReplication.Replication.EndpointType)) != "dst" {
			// This is the case where primary volume started the deletion, in this case, to be consistent we will remove replication from secondary
			replicaVolumeId, err = parse.VolumeID(*dataProtectionReplication.Replication.RemoteVolumeResourceID)
			if err != nil {
				return err
			}
		}

		// Checking replication status before deletion, it need to be broken before proceeding with deletion
		if res, err := client.ReplicationStatusMethod(ctx, replicaVolumeId.ResourceGroup, replicaVolumeId.NetAppAccountName, replicaVolumeId.CapacityPoolName, replicaVolumeId.Name); err == nil {
			// Wait for replication state = "mirrored"
			if strings.ToLower(string(res.MirrorState)) == "uninitialized" {
				if err := waitForReplMirrorState(ctx, client, *replicaVolumeId, "mirrored"); err != nil {
					return fmt.Errorf("waiting for replica %s to become 'mirrored': %+v", *replicaVolumeId, err)
				}
			}

			// Breaking replication
			_, err = client.BreakReplication(ctx,
				replicaVolumeId.ResourceGroup,
				replicaVolumeId.NetAppAccountName,
				replicaVolumeId.CapacityPoolName,
				replicaVolumeId.Name,
				&netapp.BreakReplicationRequest{
					ForceBreakReplication: utils.Bool(true),
				})

			if err != nil {
				return fmt.Errorf("breaking replication for %s: %+v", *replicaVolumeId, err)
			}

			// Waiting for replication be in broken state
			log.Printf("[DEBUG] Waiting for the replication of %s to be in broken state", *replicaVolumeId)
			if err := waitForReplMirrorState(ctx, client, *replicaVolumeId, "broken"); err != nil {
				return fmt.Errorf("waiting for the breaking of replication for %s: %+v", *replicaVolumeId, err)
			}
		}

		// Deleting replication and waiting for it to fully complete the operation
		future, err := client.DeleteReplication(ctx, replicaVolumeId.ResourceGroup, replicaVolumeId.NetAppAccountName, replicaVolumeId.CapacityPoolName, replicaVolumeId.Name)
		if err != nil {
			return fmt.Errorf("deleting replicate %s: %+v", *replicaVolumeId, err)
		}

		log.Printf("[DEBUG] Waiting for the replica of %s to be deleted", replicaVolumeId)
		if err := future.WaitForCompletionRef(ctx, client.Client); err != nil {
			return fmt.Errorf("waiting for the replica %s to be deleted: %+v", *replicaVolumeId, err)
		}
		if err := waitForReplicationDeletion(ctx, client, *replicaVolumeId); err != nil {
			return fmt.Errorf("waiting for the replica %s to be deleted: %+v", *replicaVolumeId, err)
		}
	}

	// Deleting volume and waiting for it fo fully complete the operation
	future, err := client.Delete(ctx, id.ResourceGroup, id.NetAppAccountName, id.CapacityPoolName, id.Name, utils.Bool(true))
	if err != nil {
		return fmt.Errorf("deleting %s: %+v", *id, err)
	}

	log.Printf("[DEBUG] Waiting for %s to be deleted", *id)
	if err := future.WaitForCompletionRef(ctx, client.Client); err != nil {
		return fmt.Errorf("waiting for deletion of %q: %+v", id, err)
	}
	if err := waitForVolumeDeletion(ctx, client, *id); err != nil {
		return fmt.Errorf("waiting for deletion of %s: %+v", *id, err)
	}

	return nil
}

func waitForVolumeCreateOrUpdate(ctx context.Context, client *netapp.VolumesClient, id parse.VolumeId) error {
	deadline, ok := ctx.Deadline()
	if !ok {
		return fmt.Errorf("context had no deadline")
	}
	stateConf := &pluginsdk.StateChangeConf{
		ContinuousTargetOccurence: 5,
		Delay:                     10 * time.Second,
		MinTimeout:                10 * time.Second,
		Pending:                   []string{"204", "404"},
		Target:                    []string{"200", "202"},
		Refresh:                   netappVolumeStateRefreshFunc(ctx, client, id),
		Timeout:                   time.Until(deadline),
	}

	if _, err := stateConf.WaitForStateContext(ctx); err != nil {
		return fmt.Errorf("waiting for %s to finish creating: %+v", id, err)
	}

	return nil
}

func waitForReplAuthorization(ctx context.Context, client *netapp.VolumesClient, id parse.VolumeId) error {
	deadline, ok := ctx.Deadline()
	if !ok {
		return fmt.Errorf("context had no deadline")
	}
	stateConf := &pluginsdk.StateChangeConf{
		ContinuousTargetOccurence: 5,
		Delay:                     10 * time.Second,
		MinTimeout:                10 * time.Second,
		Pending:                   []string{"204", "404", "400"}, // TODO: Remove 400 when bug is fixed on RP side, where replicationStatus returns 400 at some point during authorization process
		Target:                    []string{"200", "202"},
		Refresh:                   netappVolumeReplicationStateRefreshFunc(ctx, client, id),
		Timeout:                   time.Until(deadline),
	}

	if _, err := stateConf.WaitForStateContext(ctx); err != nil {
		return fmt.Errorf("waiting for replication authorization NetApp Volume Provisioning Service %q (Resource Group %q) to complete: %+v", id.Name, id.ResourceGroup, err)
	}

	return nil
}

func waitForReplMirrorState(ctx context.Context, client *netapp.VolumesClient, id parse.VolumeId, desiredState string) error {
	deadline, ok := ctx.Deadline()
	if !ok {
		return fmt.Errorf("context had no deadline")
	}
	stateConf := &pluginsdk.StateChangeConf{
		ContinuousTargetOccurence: 5,
		Delay:                     10 * time.Second,
		MinTimeout:                10 * time.Second,
		Pending:                   []string{"200"}, // 200 means mirror state is still Mirrored
		Target:                    []string{"204"}, // 204 means mirror state is <> than Mirrored
		Refresh:                   netappVolumeReplicationMirrorStateRefreshFunc(ctx, client, id, desiredState),
		Timeout:                   time.Until(deadline),
	}

	if _, err := stateConf.WaitForStateContext(ctx); err != nil {
		return fmt.Errorf("waiting for %s to be in the state %q: %+v", id, desiredState, err)
	}

	return nil
}

func waitForReplicationDeletion(ctx context.Context, client *netapp.VolumesClient, id parse.VolumeId) error {
	deadline, ok := ctx.Deadline()
	if !ok {
		return fmt.Errorf("context had no deadline")
	}

	stateConf := &pluginsdk.StateChangeConf{
		ContinuousTargetOccurence: 5,
		Delay:                     10 * time.Second,
		MinTimeout:                10 * time.Second,
		Pending:                   []string{"200", "202", "400"}, // TODO: Remove 400 when bug is fixed on RP side, where replicationStatus returns 400 while it is in "Deleting" state
		Target:                    []string{"404"},
		Refresh:                   netappVolumeReplicationStateRefreshFunc(ctx, client, id),
		Timeout:                   time.Until(deadline),
	}

	if _, err := stateConf.WaitForStateContext(ctx); err != nil {
		return fmt.Errorf("waiting for Replication of %s to be deleted: %+v", id, err)
	}

	return nil
}

func waitForVolumeDeletion(ctx context.Context, client *netapp.VolumesClient, id parse.VolumeId) error {
	deadline, ok := ctx.Deadline()
	if !ok {
		return fmt.Errorf("context had no deadline")
	}
	stateConf := &pluginsdk.StateChangeConf{
		ContinuousTargetOccurence: 5,
		Delay:                     10 * time.Second,
		MinTimeout:                10 * time.Second,
		Pending:                   []string{"200", "202"},
		Target:                    []string{"204", "404"},
		Refresh:                   netappVolumeStateRefreshFunc(ctx, client, id),
		Timeout:                   time.Until(deadline),
	}

	if _, err := stateConf.WaitForStateContext(ctx); err != nil {
		return fmt.Errorf("waiting for %s to be deleted: %+v", id, err)
	}

	return nil
}

func netappVolumeStateRefreshFunc(ctx context.Context, client *netapp.VolumesClient, id parse.VolumeId) pluginsdk.StateRefreshFunc {
	return func() (interface{}, string, error) {
		res, err := client.Get(ctx, id.ResourceGroup, id.NetAppAccountName, id.CapacityPoolName, id.Name)
		if err != nil {
			if !utils.ResponseWasNotFound(res.Response) {
				return nil, "", fmt.Errorf("retrieving NetApp Volume %q (Resource Group %q): %s", id.Name, id.ResourceGroup, err)
			}
		}

		return res, strconv.Itoa(res.StatusCode), nil
	}
}

func netappVolumeReplicationMirrorStateRefreshFunc(ctx context.Context, client *netapp.VolumesClient, id parse.VolumeId, desiredState string) pluginsdk.StateRefreshFunc {
	validStates := []string{"mirrored", "broken", "uninitialized"}

	return func() (interface{}, string, error) {
		// Possible Mirror States to be used as desiredStates:
		// mirrored, broken or uninitialized
		if !utils.SliceContainsValue(validStates, strings.ToLower(desiredState)) {
			return nil, "", fmt.Errorf("Invalid desired mirror state was passed to check mirror replication state (%s), possible values: (%+v)", desiredState, netapp.PossibleMirrorStateValues())
		}

		res, err := client.ReplicationStatusMethod(ctx, id.ResourceGroup, id.NetAppAccountName, id.CapacityPoolName, id.Name)
		if err != nil {
			if !utils.ResponseWasNotFound(res.Response) {
				return nil, "", fmt.Errorf("retrieving replication status information from NetApp Volume %q (Resource Group %q): %s", id.Name, id.ResourceGroup, err)
			}
		}

		// TODO: fix this refresh function to use strings instead of fake status codes
		// Setting 200 as default response
		response := 200
		if strings.EqualFold(string(res.MirrorState), desiredState) {
			// return 204 if state matches desired state
			response = 204
		}

		return res, strconv.Itoa(response), nil
	}
}

func netappVolumeReplicationStateRefreshFunc(ctx context.Context, client *netapp.VolumesClient, id parse.VolumeId) pluginsdk.StateRefreshFunc {
	return func() (interface{}, string, error) {
		res, err := client.ReplicationStatusMethod(ctx, id.ResourceGroup, id.NetAppAccountName, id.CapacityPoolName, id.Name)
		if err != nil {
			if res.StatusCode == 400 && (strings.Contains(strings.ToLower(err.Error()), "deleting") || strings.Contains(strings.ToLower(err.Error()), "volume replication missing or deleted")) {
				// This error can be ignored until a bug is fixed on RP side that it is returning 400 while the replication is in "Deleting" process
				// TODO: remove this workaround when above bug is fixed
			} else if !utils.ResponseWasNotFound(res.Response) {
				return nil, "", fmt.Errorf("retrieving replication status from NetApp Volume %q (Resource Group %q): %s", id.Name, id.ResourceGroup, err)
			}
		}

		return res, strconv.Itoa(res.StatusCode), nil
	}
}
