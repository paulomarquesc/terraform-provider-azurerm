package netapp

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/netapp/mgmt/2021-10-01/netapp"
	"github.com/hashicorp/go-azure-helpers/resourcemanager/location"
	"github.com/hashicorp/terraform-provider-azurerm/helpers/azure"
	"github.com/hashicorp/terraform-provider-azurerm/helpers/validate"
	"github.com/hashicorp/terraform-provider-azurerm/internal/clients"
	"github.com/hashicorp/terraform-provider-azurerm/internal/services/netapp/parse"
	netAppValidate "github.com/hashicorp/terraform-provider-azurerm/internal/services/netapp/validate"
	"github.com/hashicorp/terraform-provider-azurerm/internal/tags"
	"github.com/hashicorp/terraform-provider-azurerm/internal/tf/pluginsdk"
	"github.com/hashicorp/terraform-provider-azurerm/internal/tf/suppress"
	"github.com/hashicorp/terraform-provider-azurerm/internal/tf/validation"
	"github.com/hashicorp/terraform-provider-azurerm/utils"
)

type VolumeObjects interface {
	netapp.VolumeProperties | netapp.VolumeGroupVolumeProperties
}

type VolumeIDs interface {
	parse.VolumeId | parse.VolumeGroupId
}

type ExportPolicyObjects interface {
	netapp.VolumePropertiesExportPolicy | netapp.VolumePatchPropertiesExportPolicy
}

func netAppVolumeCommonSchema() map[string]*pluginsdk.Schema {
	return map[string]*pluginsdk.Schema{
		"name": {
			Type:         pluginsdk.TypeString,
			Required:     true,
			ForceNew:     true,
			ValidateFunc: netAppValidate.VolumeName,
		},

		"volume_path": {
			Type:         pluginsdk.TypeString,
			Required:     true,
			ForceNew:     true,
			ValidateFunc: netAppValidate.VolumePath,
		},

		"service_level": {
			Type:     pluginsdk.TypeString,
			Required: true,
			ForceNew: true,
			ValidateFunc: validation.StringInSlice([]string{
				string(netapp.ServiceLevelPremium),
				string(netapp.ServiceLevelStandard),
				string(netapp.ServiceLevelUltra),
			}, false),
		},

		"subnet_id": {
			Type:         pluginsdk.TypeString,
			Required:     true,
			ForceNew:     true,
			ValidateFunc: azure.ValidateResourceID,
		},

		"create_from_snapshot_resource_id": {
			Type:         pluginsdk.TypeString,
			Optional:     true,
			Computed:     true,
			ForceNew:     true,
			ValidateFunc: netAppValidate.SnapshotID,
		},

		"network_features": {
			Type:     pluginsdk.TypeString,
			Optional: true,
			Computed: true,
			ForceNew: true,
			ValidateFunc: validation.StringInSlice([]string{
				string(netapp.NetworkFeaturesBasic),
				string(netapp.NetworkFeaturesStandard),
			}, false),
		},

		"protocols": {
			Type:     pluginsdk.TypeSet,
			ForceNew: true,
			Optional: true,
			Computed: true,
			MaxItems: 2,
			Elem: &pluginsdk.Schema{
				Type: pluginsdk.TypeString,
				ValidateFunc: validation.StringInSlice([]string{
					"NFSv3",
					"NFSv4.1",
					"CIFS",
				}, false),
			},
		},

		"security_style": {
			Type:     pluginsdk.TypeString,
			Optional: true,
			ForceNew: true,
			Computed: true,
			ValidateFunc: validation.StringInSlice([]string{
				"Unix", // Using hardcoded values instead of SDK enum since no matter what case is passed,
				"Ntfs", // ANF changes casing to Pascal case in the backend. Please refer to https://github.com/Azure/azure-sdk-for-go/issues/14684
			}, false),
		},

		"storage_quota_in_gb": {
			Type:         pluginsdk.TypeInt,
			Required:     true,
			ValidateFunc: validation.IntBetween(100, 102400),
		},

		"throughput_in_mibps": {
			Type:     pluginsdk.TypeFloat,
			Optional: true,
			Computed: true,
		},

		"export_policy_rule": {
			Type:     pluginsdk.TypeList,
			Optional: true,
			MaxItems: 5,
			Elem: &pluginsdk.Resource{
				Schema: map[string]*pluginsdk.Schema{
					"rule_index": {
						Type:         pluginsdk.TypeInt,
						Required:     true,
						ValidateFunc: validation.IntBetween(1, 5),
					},

					"allowed_clients": {
						Type:     pluginsdk.TypeSet,
						Required: true,
						Elem: &pluginsdk.Schema{
							Type:         pluginsdk.TypeString,
							ValidateFunc: validate.CIDR,
						},
					},

					"protocols_enabled": {
						Type:     pluginsdk.TypeList,
						Optional: true,
						Computed: true,
						MaxItems: 1,
						MinItems: 1,
						Elem: &pluginsdk.Schema{
							Type: pluginsdk.TypeString,
							ValidateFunc: validation.StringInSlice([]string{
								"NFSv3",
								"NFSv4.1",
								"CIFS",
							}, false),
						},
					},

					"unix_read_only": {
						Type:     pluginsdk.TypeBool,
						Optional: true,
					},

					"unix_read_write": {
						Type:     pluginsdk.TypeBool,
						Optional: true,
					},

					"root_access_enabled": {
						Type:     pluginsdk.TypeBool,
						Optional: true,
					},

					"kerberos5_read_only": {
						Type:     pluginsdk.TypeBool,
						Optional: true,
					},

					"kerberos5_read_write": {
						Type:     pluginsdk.TypeBool,
						Optional: true,
					},

					"kerberos5i_read_only": {
						Type:     pluginsdk.TypeBool,
						Optional: true,
					},

					"kerberos5i_read_write": {
						Type:     pluginsdk.TypeBool,
						Optional: true,
					},

					"kerberos5p_read_only": {
						Type:     pluginsdk.TypeBool,
						Optional: true,
					},

					"kerberos5p_read_write": {
						Type:     pluginsdk.TypeBool,
						Optional: true,
					},
				},
			},
		},

		"tags": tags.Schema(),

		"mount_ip_addresses": {
			Type:     pluginsdk.TypeList,
			Computed: true,
			Elem: &pluginsdk.Schema{
				Type: pluginsdk.TypeString,
			},
		},

		"snapshot_directory_visible": {
			Type:     pluginsdk.TypeBool,
			Optional: true,
			Computed: true,
		},

		"data_protection_replication": {
			Type:     pluginsdk.TypeList,
			Optional: true,
			MaxItems: 1,
			ForceNew: true,
			Elem: &pluginsdk.Resource{
				Schema: map[string]*pluginsdk.Schema{
					"endpoint_type": {
						Type:     pluginsdk.TypeString,
						Optional: true,
						Default:  "dst",
						ValidateFunc: validation.StringInSlice([]string{
							"dst",
						}, false),
					},

					"remote_volume_location": azure.SchemaLocation(),

					"remote_volume_resource_id": {
						Type:         pluginsdk.TypeString,
						Required:     true,
						ValidateFunc: azure.ValidateResourceID,
					},

					"replication_frequency": {
						Type:     pluginsdk.TypeString,
						Required: true,
						ValidateFunc: validation.StringInSlice([]string{
							"10minutes",
							"daily",
							"hourly",
						}, false),
					},
				},
			},
		},

		"data_protection_snapshot_policy": {
			Type:     pluginsdk.TypeList,
			Optional: true,
			MaxItems: 1,
			Elem: &pluginsdk.Resource{
				Schema: map[string]*pluginsdk.Schema{
					"snapshot_policy_id": {
						Type:         pluginsdk.TypeString,
						Required:     true,
						ValidateFunc: azure.ValidateResourceID,
					},
				},
			},
		},
	}
}

func netAppVolumeSchema() map[string]*pluginsdk.Schema {
	return mergeSchemas(netAppVolumeCommonSchema(), map[string]*pluginsdk.Schema{
		"resource_group_name": azure.SchemaResourceGroupName(),

		"location": azure.SchemaLocation(),

		"account_name": {
			Type:         pluginsdk.TypeString,
			Required:     true,
			ForceNew:     true,
			ValidateFunc: netAppValidate.AccountName,
		},

		"pool_name": {
			Type:         pluginsdk.TypeString,
			Required:     true,
			ForceNew:     true,
			ValidateFunc: netAppValidate.PoolName,
		},
	})
}

func netAppVolumeGroupVolumeSchema() map[string]*pluginsdk.Schema {
	return mergeSchemas(netAppVolumeCommonSchema(), map[string]*pluginsdk.Schema{
		"capacity_pool_id": {
			Type:             pluginsdk.TypeString,
			Required:         true,
			ForceNew:         true,
			DiffSuppressFunc: suppress.CaseDifference,
			ValidateFunc:     azure.ValidateResourceID,
		},

		"proximity_placement_group_id": {
			Type:             pluginsdk.TypeString,
			Required:         true,
			ForceNew:         true,
			DiffSuppressFunc: suppress.CaseDifference,
			ValidateFunc:     azure.ValidateResourceID,
		},

		"volume_spec_name": {
			Type:     pluginsdk.TypeString,
			Required: true,
			ForceNew: true,
		},
	})
}

func getVolumePropertiesObject[T VolumeObjects, T1 VolumeIDs](ctx context.Context, d *pluginsdk.ResourceData, meta interface{}, id T1) (*T, error) {
	// TODO: (pmarques) retire this
	var volumeType T

	if reflect.TypeOf(any(volumeType).(T)) == reflect.TypeOf(netapp.VolumeProperties{}) {

		volumeProperties, err := getCommonVolumeProperties(ctx, d, any(id).(parse.VolumeId))
		if err != nil {
			return &volumeType, err
		}

		location := azure.NormalizeLocation(d.Get("location").(string))

		// Handling volume creation from snapshot case
		snapshotResourceID := d.Get("create_from_snapshot_resource_id").(string)
		snapshotID := ""
		if snapshotResourceID != "" {
			// Get snapshot ID GUID value
			parsedSnapshotResourceID, err := parse.SnapshotID(snapshotResourceID)
			if err != nil {
				return &volumeType, fmt.Errorf("parsing snapshotResourceID %q: %+v", snapshotResourceID, err)
			}

			snapshotClient := meta.(*clients.Client).NetApp.SnapshotClient
			snapshotResponse, err := snapshotClient.Get(
				ctx,
				parsedSnapshotResourceID.ResourceGroup,
				parsedSnapshotResourceID.NetAppAccountName,
				parsedSnapshotResourceID.CapacityPoolName,
				parsedSnapshotResourceID.VolumeName,
				parsedSnapshotResourceID.Name,
			)
			if err != nil {
				return &volumeType, fmt.Errorf("getting snapshot from NetApp Volume %q (Resource Group %q): %+v", parsedSnapshotResourceID.VolumeName, parsedSnapshotResourceID.ResourceGroup, err)
			}
			snapshotID = *snapshotResponse.SnapshotID

			// Validate if properties that cannot be changed matches (protocols, subnet_id, location, resource group, account_name, pool_name, service_level)
			volumeClient := meta.(*clients.Client).NetApp.VolumeClient
			sourceVolume, err := volumeClient.Get(
				ctx,
				parsedSnapshotResourceID.ResourceGroup,
				parsedSnapshotResourceID.NetAppAccountName,
				parsedSnapshotResourceID.CapacityPoolName,
				parsedSnapshotResourceID.VolumeName,
			)
			if err != nil {
				return &volumeType, fmt.Errorf("getting source NetApp Volume (snapshot's parent resource) %q (Resource Group %q): %+v", parsedSnapshotResourceID.VolumeName, parsedSnapshotResourceID.ResourceGroup, err)
			}

			parsedVolumeID, err := parse.VolumeID(*sourceVolume.ID)
			if err != nil {
				return &volumeType, fmt.Errorf("parsing Source Volume ID: %s", err)
			}
			propertyMismatch := []string{}
			if !ValidateSlicesEquality(*sourceVolume.ProtocolTypes, *volumeProperties.ProtocolTypes, false) {
				propertyMismatch = append(propertyMismatch, "protocols")
			}
			if !strings.EqualFold(*sourceVolume.SubnetID, *volumeProperties.SubnetID) {
				propertyMismatch = append(propertyMismatch, "subnet_id")
			}
			if !strings.EqualFold(*sourceVolume.Location, location) {
				propertyMismatch = append(propertyMismatch, "location")
			}
			if !strings.EqualFold(string(sourceVolume.ServiceLevel), string(volumeProperties.ServiceLevel)) {
				propertyMismatch = append(propertyMismatch, "service_level")
			}
			if !strings.EqualFold(parsedVolumeID.ResourceGroup, any(id).(parse.VolumeId).ResourceGroup) {
				propertyMismatch = append(propertyMismatch, "resource_group_name")
			}
			if !strings.EqualFold(parsedVolumeID.NetAppAccountName, any(id).(parse.VolumeId).NetAppAccountName) {
				propertyMismatch = append(propertyMismatch, "account_name")
			}
			if !strings.EqualFold(parsedVolumeID.CapacityPoolName, any(id).(parse.VolumeId).CapacityPoolName) {
				propertyMismatch = append(propertyMismatch, "pool_name")
			}
			if len(propertyMismatch) > 0 {
				return &volumeType, fmt.Errorf("following NetApp Volume properties on new Volume from Snapshot does not match Snapshot's source %s: %s", id, strings.Join(propertyMismatch, ", "))
			}

			volumeProperties.SnapshotID = utils.String(snapshotID)
		}

		return any(volumeProperties).(*T), nil

	} else if reflect.TypeOf(any(volumeType).(T)) == reflect.TypeOf(netapp.VolumeGroupVolumeProperties{}) {

		volumeProperties, err := getCommonVolumeProperties(ctx, d, any(id).(parse.VolumeGroupId))
		if err != nil {
			return &volumeType, err
		}

		proximityPlacementGroupID := d.Get("proximity_placement_group_id").(string)
		volumeProperties.ProximityPlacementGroup = utils.String(proximityPlacementGroupID)

		return any(volumeProperties).(*T), nil
	}

	return &volumeType, fmt.Errorf("invalid type provided for generics function")
}

func getCommonVolumeProperties[T VolumeIDs](ctx context.Context, d *pluginsdk.ResourceData, id T) (*netapp.VolumeProperties, error) {
	// TODO: (pmarques) retire this

	// Handling common volume properties

	volumePath := d.Get("volume_path").(string)
	serviceLevel := d.Get("service_level").(string)
	subnetID := d.Get("subnet_id").(string)

	networkFeatures := d.Get("network_features").(string)
	if networkFeatures == "" {
		networkFeatures = string(netapp.NetworkFeaturesBasic)
	}

	protocols := d.Get("protocols").(*pluginsdk.Set).List()
	if len(protocols) == 0 {
		protocols = append(protocols, "NFSv3")
	}

	// Handling security style property
	securityStyle := d.Get("security_style").(string)
	if strings.EqualFold(securityStyle, "unix") && len(protocols) == 1 && strings.EqualFold(protocols[0].(string), "cifs") {
		return &netapp.VolumeProperties{}, fmt.Errorf("unix security style cannot be used in a CIFS enabled volume for %s", id)

	}
	if strings.EqualFold(securityStyle, "ntfs") && len(protocols) == 1 && (strings.EqualFold(protocols[0].(string), "nfsv3") || strings.EqualFold(protocols[0].(string), "nfsv4.1")) {
		return &netapp.VolumeProperties{}, fmt.Errorf("ntfs security style cannot be used in a NFSv3/NFSv4.1 enabled volume for %s", id)
	}

	storageQuotaInGB := int64(d.Get("storage_quota_in_gb").(int) * 1073741824)

	exportPolicyRuleRaw := d.Get("export_policy_rule").([]interface{})
	exportPolicyRule := expandNetAppVolumeExportPolicyRule[netapp.VolumePropertiesExportPolicy](exportPolicyRuleRaw)

	dataProtectionReplicationRaw := d.Get("data_protection_replication").([]interface{})
	dataProtectionSnapshotPolicyRaw := d.Get("data_protection_snapshot_policy").([]interface{})

	dataProtectionReplication := expandNetAppVolumeDataProtectionReplication(dataProtectionReplicationRaw)
	dataProtectionSnapshotPolicy := expandNetAppVolumeDataProtectionSnapshotPolicy(dataProtectionSnapshotPolicyRaw)

	volumeType := ""
	if dataProtectionReplication != nil && dataProtectionReplication.Replication != nil && strings.ToLower(string(dataProtectionReplication.Replication.EndpointType)) == "dst" {
		volumeType = "DataProtection"
	}

	// Validating that snapshot policies are not being created in a data protection volume
	if dataProtectionSnapshotPolicy.Snapshot != nil && volumeType != "" {
		return &netapp.VolumeProperties{}, fmt.Errorf("snapshot policy cannot be enabled on a data protection volume for %s", id)
	}

	snapshotDirectoryVisible := d.Get("snapshot_directory_visible").(bool)

	volumeProperties := &netapp.VolumeProperties{
		CreationToken:   utils.String(volumePath),
		ServiceLevel:    netapp.ServiceLevel(serviceLevel),
		SubnetID:        utils.String(subnetID),
		NetworkFeatures: netapp.NetworkFeatures(networkFeatures),
		ProtocolTypes:   utils.ExpandStringSlice(protocols),
		SecurityStyle:   netapp.SecurityStyle(securityStyle),
		UsageThreshold:  utils.Int64(storageQuotaInGB),
		ExportPolicy:    exportPolicyRule,
		VolumeType:      utils.String(volumeType),
		DataProtection: &netapp.VolumePropertiesDataProtection{
			Replication: dataProtectionReplication.Replication,
			Snapshot:    dataProtectionSnapshotPolicy.Snapshot,
		},
		SnapshotDirectoryVisible: utils.Bool(snapshotDirectoryVisible),
	}

	if throughputMibps, ok := d.GetOk("throughput_in_mibps"); ok {
		volumeProperties.ThroughputMibps = utils.Float(throughputMibps.(float64))
	}

	return volumeProperties, nil
}

func setVolumePropertiesResourceData[T VolumeObjects](d *pluginsdk.ResourceData, properties T) error {
	// TODO: (pmarques) retire this
	var volumeType T
	var volObjectValue = reflect.ValueOf(&properties).Elem()

	// Working on Common Properties
	d.Set("volume_path", volObjectValue.FieldByName("CreationToken").Interface().(*string))
	d.Set("service_level", volObjectValue.FieldByName("ServiceLevel").Interface().(netapp.ServiceLevel))
	d.Set("subnet_id", volObjectValue.FieldByName("SubnetID").Interface().(*string))
	d.Set("network_features", volObjectValue.FieldByName("NetworkFeatures").Interface().(netapp.NetworkFeatures))
	d.Set("protocols", volObjectValue.FieldByName("ProtocolTypes").Interface().(*[]string))
	d.Set("security_style", volObjectValue.FieldByName("SecurityStyle").Interface().(netapp.SecurityStyle))
	d.Set("snapshot_directory_visible", volObjectValue.FieldByName("SnapshotDirectoryVisible").Interface().(*bool))
	d.Set("throughput_in_mibps", volObjectValue.FieldByName("ThroughputMibps").Interface().(*float64))

	if *(volObjectValue.FieldByName("UsageThreshold").Interface().(*int64)) > 0 {
		usageThreshold := *(volObjectValue.FieldByName("UsageThreshold").Interface().(*int64)) / 1073741824
		d.Set("storage_quota_in_gb", usageThreshold)
	}

	if err := d.Set("export_policy_rule", flattenNetAppVolumeExportPolicyRule(volObjectValue.FieldByName("ExportPolicy").Interface().(*netapp.VolumePropertiesExportPolicy))); err != nil {
		return fmt.Errorf("setting `export_policy_rule`: %+v", err)
	}

	if err := d.Set("mount_ip_addresses", flattenNetAppVolumeMountIPAddresses(volObjectValue.FieldByName("MountTargets").Interface().(*[]netapp.MountTargetProperties))); err != nil {
		return fmt.Errorf("setting `mount_ip_addresses`: %+v", err)
	}

	if err := d.Set("data_protection_replication", flattenNetAppVolumeDataProtectionReplication(volObjectValue.FieldByName("DataProtection").Interface().(*netapp.VolumePropertiesDataProtection))); err != nil {
		return fmt.Errorf("setting `data_protection_replication`: %+v", err)
	}

	if err := d.Set("data_protection_snapshot_policy", flattenNetAppVolumeDataProtectionSnapshotPolicy(volObjectValue.FieldByName("DataProtection").Interface().(*netapp.VolumePropertiesDataProtection))); err != nil {
		return fmt.Errorf("setting `data_protection_snapshot_policy`: %+v", err)
	}

	// Working on specific properties based on object type
	if reflect.TypeOf(any(volumeType).(T)) == reflect.TypeOf(netapp.VolumeProperties{}) {

		return nil

	} else if reflect.TypeOf(any(volumeType).(T)) == reflect.TypeOf(netapp.VolumeGroupVolumeProperties{}) {

		d.Set("proximity_placement_group_id", volObjectValue.FieldByName("ProximityPlacementGroup").Interface().(*string))

		return nil

	}

	return fmt.Errorf("invalid type provided for generics function")
}

func expandNetAppVolumeExportPolicyRule[T ExportPolicyObjects](input []interface{}) *T {
	var ExportPolicyType T

	results := make([]netapp.ExportPolicyRule, 0)
	for _, item := range input {
		if item != nil {
			v := item.(map[string]interface{})
			ruleIndex := int32(v["rule_index"].(int))
			allowedClients := strings.Join(*utils.ExpandStringSlice(v["allowed_clients"].(*pluginsdk.Set).List()), ",")

			cifsEnabled := false
			nfsv3Enabled := false
			nfsv41Enabled := false

			if vpe := v["protocols_enabled"]; vpe != nil {
				protocolsEnabled := vpe.([]interface{})
				if len(protocolsEnabled) != 0 {
					for _, protocol := range protocolsEnabled {
						if protocol != nil {
							switch strings.ToLower(protocol.(string)) {
							case "cifs":
								cifsEnabled = true
							case "nfsv3":
								nfsv3Enabled = true
							case "nfsv4.1":
								nfsv41Enabled = true
							}
						}
					}
				}
			}

			unixReadOnly := v["unix_read_only"].(bool)
			unixReadWrite := v["unix_read_write"].(bool)
			rootAccessEnabled := v["root_access_enabled"].(bool)
			kerberos5ReadOnly := v["kerberos5_read_only"].(bool)
			kerberos5ReadWrite := v["kerberos5_read_write"].(bool)
			kerberos5iReadOnly := v["kerberos5i_read_only"].(bool)
			kerberos5iReadWrite := v["kerberos5i_read_write"].(bool)
			kerberos5pReadOnly := v["kerberos5p_read_only"].(bool)
			kerberos5pReadWrite := v["kerberos5p_read_write"].(bool)

			result := netapp.ExportPolicyRule{
				AllowedClients:      utils.String(allowedClients),
				Cifs:                utils.Bool(cifsEnabled),
				Nfsv3:               utils.Bool(nfsv3Enabled),
				Nfsv41:              utils.Bool(nfsv41Enabled),
				RuleIndex:           utils.Int32(ruleIndex),
				UnixReadOnly:        utils.Bool(unixReadOnly),
				UnixReadWrite:       utils.Bool(unixReadWrite),
				HasRootAccess:       utils.Bool(rootAccessEnabled),
				Kerberos5ReadOnly:   utils.Bool(kerberos5ReadOnly),
				Kerberos5ReadWrite:  utils.Bool(kerberos5ReadWrite),
				Kerberos5iReadOnly:  utils.Bool(kerberos5iReadOnly),
				Kerberos5iReadWrite: utils.Bool(kerberos5iReadWrite),
				Kerberos5pReadOnly:  utils.Bool(kerberos5pReadOnly),
				Kerberos5pReadWrite: utils.Bool(kerberos5pReadWrite),
			}

			results = append(results, result)
		}
	}

	if reflect.TypeOf(any(ExportPolicyType).(T)) == reflect.TypeOf(netapp.VolumePropertiesExportPolicy{}) {

		return any(&netapp.VolumePropertiesExportPolicy{
			Rules: &results,
		}).(*T)

	} else if reflect.TypeOf(any(ExportPolicyType).(T)) == reflect.TypeOf(netapp.VolumePatchPropertiesExportPolicy{}) {

		return any(&netapp.VolumePatchPropertiesExportPolicy{
			Rules: &results,
		}).(*T)

	}

	return any(&netapp.VolumePatchPropertiesExportPolicy{
		Rules: &[]netapp.ExportPolicyRule{},
	}).(*T)
}

func expandNetAppVolumeDataProtectionReplication(input []interface{}) *netapp.VolumePropertiesDataProtection {
	if len(input) == 0 || input[0] == nil {
		return &netapp.VolumePropertiesDataProtection{}
	}

	replicationObject := netapp.ReplicationObject{}

	replicationRaw := input[0].(map[string]interface{})

	if v, ok := replicationRaw["endpoint_type"]; ok {
		replicationObject.EndpointType = netapp.EndpointType(v.(string))
	}
	if v, ok := replicationRaw["remote_volume_location"]; ok {
		replicationObject.RemoteVolumeRegion = utils.String(v.(string))
	}
	if v, ok := replicationRaw["remote_volume_resource_id"]; ok {
		replicationObject.RemoteVolumeResourceID = utils.String(v.(string))
	}
	if v, ok := replicationRaw["replication_frequency"]; ok {
		replicationObject.ReplicationSchedule = netapp.ReplicationSchedule(translateTFSchedule(v.(string)))
	}

	return &netapp.VolumePropertiesDataProtection{
		Replication: &replicationObject,
	}
}

func expandNetAppVolumeDataProtectionSnapshotPolicy(input []interface{}) *netapp.VolumePropertiesDataProtection {
	if len(input) == 0 || input[0] == nil {
		return &netapp.VolumePropertiesDataProtection{}
	}

	snapshotObject := netapp.VolumeSnapshotProperties{}

	snapshotRaw := input[0].(map[string]interface{})

	if v, ok := snapshotRaw["snapshot_policy_id"]; ok {
		snapshotObject.SnapshotPolicyID = utils.String(v.(string))
	}

	return &netapp.VolumePropertiesDataProtection{
		Snapshot: &snapshotObject,
	}
}

func expandNetAppVolumeDataProtectionSnapshotPolicyPatch(input []interface{}) *netapp.VolumePatchPropertiesDataProtection {
	if len(input) == 0 || input[0] == nil {
		return &netapp.VolumePatchPropertiesDataProtection{}
	}

	snapshotObject := netapp.VolumeSnapshotProperties{}

	snapshotRaw := input[0].(map[string]interface{})

	if v, ok := snapshotRaw["snapshot_policy_id"]; ok {
		snapshotObject.SnapshotPolicyID = utils.String(v.(string))
	}

	return &netapp.VolumePatchPropertiesDataProtection{
		Snapshot: &snapshotObject,
	}
}

func flattenNetAppVolumeExportPolicyRule(input *netapp.VolumePropertiesExportPolicy) []interface{} {
	results := make([]interface{}, 0)
	if input == nil || input.Rules == nil {
		return results
	}

	for _, item := range *input.Rules {
		ruleIndex := int32(0)
		if v := item.RuleIndex; v != nil {
			ruleIndex = *v
		}
		allowedClients := []string{}
		if v := item.AllowedClients; v != nil {
			allowedClients = strings.Split(*v, ",")
		}

		protocolsEnabled := []string{}
		if v := item.Cifs; v != nil {
			if *v {
				protocolsEnabled = append(protocolsEnabled, "CIFS")
			}
		}
		if v := item.Nfsv3; v != nil {
			if *v {
				protocolsEnabled = append(protocolsEnabled, "NFSv3")
			}
		}
		if v := item.Nfsv41; v != nil {
			if *v {
				protocolsEnabled = append(protocolsEnabled, "NFSv4.1")
			}
		}
		unixReadOnly := false
		if v := item.UnixReadOnly; v != nil {
			unixReadOnly = *v
		}
		unixReadWrite := false
		if v := item.UnixReadWrite; v != nil {
			unixReadWrite = *v
		}
		rootAccessEnabled := false
		if v := item.HasRootAccess; v != nil {
			rootAccessEnabled = *v
		}

		result := map[string]interface{}{
			"rule_index":          ruleIndex,
			"allowed_clients":     utils.FlattenStringSlice(&allowedClients),
			"unix_read_only":      unixReadOnly,
			"unix_read_write":     unixReadWrite,
			"root_access_enabled": rootAccessEnabled,
			"protocols_enabled":   utils.FlattenStringSlice(&protocolsEnabled),
		}
		results = append(results, result)
	}

	return results
}

func flattenNetAppVolumeMountIPAddresses(input *[]netapp.MountTargetProperties) []interface{} {
	results := make([]interface{}, 0)
	if input == nil {
		return results
	}

	for _, item := range *input {
		if item.IPAddress != nil {
			results = append(results, item.IPAddress)
		}
	}

	return results
}

func flattenNetAppVolumeDataProtectionReplication(input *netapp.VolumePropertiesDataProtection) []interface{} {
	if input == nil || input.Replication == nil {
		return []interface{}{}
	}

	if strings.ToLower(string(input.Replication.EndpointType)) == "" || strings.ToLower(string(input.Replication.EndpointType)) != "dst" {
		return []interface{}{}
	}

	return []interface{}{
		map[string]interface{}{
			"endpoint_type":             strings.ToLower(string(input.Replication.EndpointType)),
			"remote_volume_location":    location.NormalizeNilable(input.Replication.RemoteVolumeRegion),
			"remote_volume_resource_id": input.Replication.RemoteVolumeResourceID,
			"replication_frequency":     translateSDKSchedule(strings.ToLower(string(input.Replication.ReplicationSchedule))),
		},
	}
}

func flattenNetAppVolumeDataProtectionSnapshotPolicy(input *netapp.VolumePropertiesDataProtection) []interface{} {
	if input == nil || input.Snapshot == nil {
		return []interface{}{}
	}

	return []interface{}{
		map[string]interface{}{
			"snapshot_policy_id": input.Snapshot.SnapshotPolicyID,
		},
	}
}

func translateTFSchedule(scheduleName string) string {
	if strings.EqualFold(scheduleName, "10minutes") {
		return "_10minutely"
	}

	return scheduleName
}

func translateSDKSchedule(scheduleName string) string {
	if strings.EqualFold(scheduleName, "_10minutely") {
		return "10minutes"
	}

	return scheduleName
}

func mergeSchemas(schemas ...map[string]*pluginsdk.Schema) map[string]*pluginsdk.Schema {

	result := map[string]*pluginsdk.Schema{}

	for _, schema := range schemas {
		for k, v := range schema {
			if result[k] == nil {
				result[k] = v
			}
		}
	}

	return result
}
