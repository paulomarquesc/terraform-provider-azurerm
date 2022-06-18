package netapp

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/netapp/mgmt/2021-10-01/netapp"
	"github.com/hashicorp/terraform-provider-azurerm/helpers/azure"
	"github.com/hashicorp/terraform-provider-azurerm/helpers/tf"
	"github.com/hashicorp/terraform-provider-azurerm/internal/clients"
	"github.com/hashicorp/terraform-provider-azurerm/internal/services/netapp/parse"
	netAppValidate "github.com/hashicorp/terraform-provider-azurerm/internal/services/netapp/validate"
	"github.com/hashicorp/terraform-provider-azurerm/internal/tags"
	"github.com/hashicorp/terraform-provider-azurerm/internal/tf/pluginsdk"
	"github.com/hashicorp/terraform-provider-azurerm/internal/tf/validation"
	"github.com/hashicorp/terraform-provider-azurerm/internal/timeouts"
	"github.com/hashicorp/terraform-provider-azurerm/utils"
)

func resourceNetAppVolumeGroup() *pluginsdk.Resource {
	return &pluginsdk.Resource{
		Create: resourceNetAppVolumeGroupCreate,
		Read:   resourceNetAppVolumeGroupRead,
		Delete: resourceNetAppVolumeGroupDelete,

		Timeouts: &pluginsdk.ResourceTimeout{
			Create: pluginsdk.DefaultTimeout(60 * time.Minute),
			Read:   pluginsdk.DefaultTimeout(5 * time.Minute),
			Update: pluginsdk.DefaultTimeout(60 * time.Minute),
			Delete: pluginsdk.DefaultTimeout(60 * time.Minute),
		},
		Importer: pluginsdk.ImporterValidatingResourceId(func(id string) error {
			_, err := parse.VolumeGroupID(id)
			return err
		}),

		Schema: map[string]*pluginsdk.Schema{
			"name": {
				Type:         pluginsdk.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: netAppValidate.VolumeGroupName,
			},

			"resource_group_name": azure.SchemaResourceGroupName(),

			"location": azure.SchemaLocation(),

			"account_name": {
				Type:         pluginsdk.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: netAppValidate.AccountName,
			},

			"group_description": {
				Type:     pluginsdk.TypeString,
				Required: true,
				ForceNew: true,
			},

			"application_type": {
				Type:     pluginsdk.TypeString,
				Required: true,
				ForceNew: true,
				ValidateFunc: validation.StringInSlice([]string{
					string(netapp.ApplicationTypeSAPHANA),
				}, false),
			},

			"application_identifier": {
				Type:         pluginsdk.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringLenBetween(1, 3),
			},

			"deployment_spec_id": {
				Type:         pluginsdk.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.IsUUID,
			},

			"volume": {
				Type:     pluginsdk.TypeList,
				Required: true,
				ForceNew: true,
				MinItems: 1,
				MaxItems: 5,
				Elem: &pluginsdk.Resource{
					Schema: netAppVolumeGroupVolumeSchema(),
				},
			},

			// Can't use tags.Schema since there is no patch available
			"tags": {
				Type:     pluginsdk.TypeMap,
				Optional: true,
				ForceNew: true,
				Elem: &pluginsdk.Schema{
					Type: pluginsdk.TypeString,
				},
			},
		},
	}
}

func resourceNetAppVolumeGroupCreate(d *pluginsdk.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).NetApp.VolumeGroupClient
	subscriptionId := meta.(*clients.Client).Account.SubscriptionId
	ctx, cancel := timeouts.ForCreateUpdate(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id := parse.NewVolumeGroupID(subscriptionId, d.Get("resource_group_name").(string), d.Get("account_name").(string), d.Get("name").(string))
	if d.IsNewResource() {
		existing, err := client.Get(ctx, id.ResourceGroup, id.NetAppAccountName, id.Name)
		if err != nil {
			if !utils.ResponseWasNotFound(existing.Response) {
				return fmt.Errorf("checking for presence of existing %s: %+v", id, err)
			}
		}
		if !utils.ResponseWasNotFound(existing.Response) {
			return tf.ImportAsExistsError("azurerm_netapp_volume_group", id.ID())
		}
	}

	location := azure.NormalizeLocation(d.Get("location").(string))
	groupDescription := d.Get("group_description").(string)
	applicationType := d.Get("application_type").(string)
	applicationIdentifier := d.Get("application_identifier").(string)
	deploymentSpecId := d.Get("deployment_spec_id").(string)

	volumeList := make([]netapp.VolumeGroupVolumeProperties, 0)
	for _, item := range d.Get("volume").([]interface{}) {
		// volume, err := getVolumePropertiesObject[netapp.VolumeGroupVolumeProperties](ctx, &item, meta, id)
		// if err != nil {
		// 	return err
		// }

		if item != nil {
			v := item.(map[string]interface{})

			name := v["name"].(string)
			volumePath := v["volume_path"].(string)
			serviceLevel := v["service_level"].(string)
			subnetID := v["subnet_id"].(string)
			capacityPoolID := v["capacity_pool_id"].(string)

			networkFeatures := v["network_features"].(string)
			if networkFeatures == "" {
				networkFeatures = string(netapp.NetworkFeaturesBasic)
			}

			protocols := v["protocols"].(*pluginsdk.Set).List()
			if len(protocols) == 0 {
				protocols = append(protocols, "NFSv3")
			}

			// Handling security style property
			securityStyle := v["security_style"].(string)
			if strings.EqualFold(securityStyle, "unix") && len(protocols) == 1 && strings.EqualFold(protocols[0].(string), "cifs") {
				return fmt.Errorf("unix security style cannot be used in a CIFS enabled volume for %s", id)

			}
			if strings.EqualFold(securityStyle, "ntfs") && len(protocols) == 1 && (strings.EqualFold(protocols[0].(string), "nfsv3") || strings.EqualFold(protocols[0].(string), "nfsv4.1")) {
				return fmt.Errorf("ntfs security style cannot be used in a NFSv3/NFSv4.1 enabled volume for %s", id)
			}

			storageQuotaInGB := int64(v["storage_quota_in_gb"].(int) * 1073741824)

			exportPolicyRuleRaw := v["export_policy_rule"].([]interface{})
			exportPolicyRule := expandNetAppVolumeExportPolicyRule[netapp.VolumePropertiesExportPolicy](exportPolicyRuleRaw)

			dataProtectionReplicationRaw := v["data_protection_replication"].([]interface{})
			dataProtectionSnapshotPolicyRaw := v["data_protection_snapshot_policy"].([]interface{})

			dataProtectionReplication := expandNetAppVolumeDataProtectionReplication(dataProtectionReplicationRaw)
			dataProtectionSnapshotPolicy := expandNetAppVolumeDataProtectionSnapshotPolicy(dataProtectionSnapshotPolicyRaw)

			volumeType := ""
			if dataProtectionReplication != nil && dataProtectionReplication.Replication != nil && strings.ToLower(string(dataProtectionReplication.Replication.EndpointType)) == "dst" {
				volumeType = "DataProtection"
			}

			// Validating that snapshot policies are not being created in a data protection volume
			if dataProtectionSnapshotPolicy.Snapshot != nil && volumeType != "" {
				return fmt.Errorf("snapshot policy cannot be enabled on a data protection volume for %s", id)
			}

			snapshotDirectoryVisible := v["snapshot_directory_visible"].(bool)

			volumeProperties := &netapp.VolumeGroupVolumeProperties{
				Name: utils.String(name),
				VolumeProperties: &netapp.VolumeProperties{

					CapacityPoolResourceID: utils.String(capacityPoolID),
					CreationToken:          utils.String(volumePath),
					ServiceLevel:           netapp.ServiceLevel(serviceLevel),
					SubnetID:               utils.String(subnetID),
					NetworkFeatures:        netapp.NetworkFeatures(networkFeatures),
					ProtocolTypes:          utils.ExpandStringSlice(protocols),
					SecurityStyle:          netapp.SecurityStyle(securityStyle),
					UsageThreshold:         utils.Int64(storageQuotaInGB),
					ExportPolicy:           exportPolicyRule,
					VolumeType:             utils.String(volumeType),
					DataProtection: &netapp.VolumePropertiesDataProtection{
						Replication: dataProtectionReplication.Replication,
						Snapshot:    dataProtectionSnapshotPolicy.Snapshot,
					},
					SnapshotDirectoryVisible: utils.Bool(snapshotDirectoryVisible),
				},
				Tags: tags.Expand(v["tags"].(map[string]interface{})),
			}

			if throughputMibps, ok := v["throughput_in_mibps"]; ok {
				volumeProperties.VolumeProperties.ThroughputMibps = utils.Float(throughputMibps.(float64))
			}

			proximityPlacementGroupID := v["proximity_placement_group_id"].(string)
			volumeProperties.VolumeProperties.ProximityPlacementGroup = utils.String(proximityPlacementGroupID)

			volumeSpecName := v["volume_spec_name"].(string)
			volumeProperties.VolumeProperties.VolumeSpecName = utils.String(volumeSpecName)

			volumeList = append(volumeList, *volumeProperties)
		}
	}

	parameters := netapp.VolumeGroupDetails{
		Location: utils.String(location),
		VolumeGroupProperties: &netapp.VolumeGroupProperties{
			GroupMetaData: &netapp.VolumeGroupMetaData{
				GroupDescription:      utils.String(groupDescription),
				ApplicationType:       netapp.ApplicationType(applicationType),
				ApplicationIdentifier: utils.String(applicationIdentifier),
				DeploymentSpecID:      utils.String(deploymentSpecId),
			},
			Volumes: &volumeList,
		},
		Tags: tags.Expand(d.Get("tags").(map[string]interface{})),
	}

	future, err := client.Create(ctx, parameters, id.ResourceGroup, id.NetAppAccountName, id.Name)
	if err != nil {
		return fmt.Errorf("creating %s: %+v", id, err)
	}
	if err = future.WaitForCompletionRef(ctx, client.Client); err != nil {
		return fmt.Errorf("waiting for the creation of %s: %+v", id, err)
	}

	// // Waiting for volume be completely provisioned
	// if err := waitForVolumeCreateOrUpdate(ctx, client, id); err != nil {
	// 	return err
	// }

	d.SetId(id.ID())

	return resourceNetAppVolumeGroupRead(d, meta)
}

func resourceNetAppVolumeGroupRead(d *pluginsdk.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).NetApp.VolumeGroupClient
	ctx, cancel := timeouts.ForRead(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id, err := parse.VolumeGroupID(d.Id())
	if err != nil {
		return err
	}

	resp, err := client.Get(ctx, id.ResourceGroup, id.NetAppAccountName, id.Name)
	if err != nil {
		if utils.ResponseWasNotFound(resp.Response) {
			log.Printf("[INFO] %s was not found - removing from state", *id)
			d.SetId("")
			return nil
		}
		return fmt.Errorf("reading %s: %+v", *id, err)
	}

	d.Set("name", id.Name)
	d.Set("resource_group_name", id.ResourceGroup)
	d.Set("account_name", id.NetAppAccountName)
	if location := resp.Location; location != nil {
		d.Set("location", azure.NormalizeLocation(*location))
	}

	if volGroupProps := resp.VolumeGroupProperties; volGroupProps != nil {
		d.Set("group_description", volGroupProps.GroupMetaData.GroupDescription)
		d.Set("application_type", volGroupProps.GroupMetaData.ApplicationType)
		d.Set("application_identifier", volGroupProps.GroupMetaData.ApplicationIdentifier)

		if err := d.Set("volume", flattenNetAppVolumeGroupVolumes(volGroupProps.Volumes)); err != nil {
			return fmt.Errorf("setting `volume`: %+v", err)
		}
	}

	return tags.FlattenAndSet(d, resp.Tags)
}

func resourceNetAppVolumeGroupDelete(d *pluginsdk.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).NetApp.VolumeGroupClient
	ctx, cancel := timeouts.ForDelete(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id, err := parse.VolumeID(d.Id())
	if err != nil {
		return err
	}

	// Deleting volume and waiting for it fo fully complete the operation
	future, err := client.Delete(ctx, id.ResourceGroup, id.NetAppAccountName, id.Name)
	if err != nil {
		return fmt.Errorf("deleting %s: %+v", *id, err)
	}

	log.Printf("[DEBUG] Waiting for %s to be deleted", *id)
	if err := future.WaitForCompletionRef(ctx, client.Client); err != nil {
		return fmt.Errorf("waiting for deletion of %q: %+v", id, err)
	}

	// TODO (pmarques): Check if this is needed.
	// if err := waitForVolumeDeletion(ctx, client, *id); err != nil {
	// 	return fmt.Errorf("waiting for deletion of %s: %+v", *id, err)
	// }

	return nil
}

func flattenNetAppVolumeGroupVolumes(input *[]netapp.VolumeGroupVolumeProperties) []interface{} {
	results := make([]interface{}, 0)
	if input == nil {
		return results
	}

	for _, item := range *input {
		volumeResourceData := pluginsdk.ResourceData{}
		if err := setVolumePropertiesResourceData(&volumeResourceData, item); err != nil {
			// If this fails in any item, invalidate the whole set
			return make([]interface{}, 0)
		}

		results = append(results, volumeResourceData)
	}

	return results
}
