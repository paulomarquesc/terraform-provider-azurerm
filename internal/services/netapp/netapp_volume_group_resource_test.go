package netapp_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-provider-azurerm/internal/acceptance"
	"github.com/hashicorp/terraform-provider-azurerm/internal/acceptance/check"
	"github.com/hashicorp/terraform-provider-azurerm/internal/clients"
	"github.com/hashicorp/terraform-provider-azurerm/internal/services/netapp/parse"
	"github.com/hashicorp/terraform-provider-azurerm/internal/tf/pluginsdk"
	"github.com/hashicorp/terraform-provider-azurerm/utils"
)

type NetAppVolumeGroupResource struct{}

func TestAccNetAppVolumeGroup_basic(t *testing.T) {
	data := acceptance.BuildTestData(t, "azurerm_netapp_volume", "test")
	r := NetAppVolumeGroupResource{}

	data.ResourceTest(t, r, []acceptance.TestStep{
		{
			Config: r.basic(data),
			Check: acceptance.ComposeTestCheckFunc(
				check.That(data.ResourceName).ExistsInAzure(r),
			),
		},
		data.ImportStep(),
	})
}

func (t NetAppVolumeGroupResource) Exists(ctx context.Context, clients *clients.Client, state *pluginsdk.InstanceState) (*bool, error) {
	id, err := parse.VolumeGroupID(state.ID)
	if err != nil {
		return nil, err
	}

	resp, err := clients.NetApp.VolumeGroupClient.Get(ctx, id.ResourceGroup, id.NetAppAccountName, id.Name)
	if err != nil {
		return nil, fmt.Errorf("reading Netapp Volume (%s): %+v", id.String(), err)
	}

	return utils.Bool(resp.ID != nil), nil
}

func (NetAppVolumeGroupResource) basic(data acceptance.TestData) string {
	template := NetAppVolumeGroupResource{}.templatePPG(data)
	return fmt.Sprintf(`
%[1]s

resource "azurerm_netapp_volume_group" "test" {
  name                   = "acctest-NetAppVolumeGroup-%[2]d"
  location               = azurerm_resource_group.test.location
  resource_group_name    = azurerm_resource_group.test.name
  account_name           = azurerm_netapp_account.test.name
  group_description      = "Test volume group"
  application_type       = "SAP-HANA"
  application_identifier = "TST"
  deployment_spec_id     = "20542149-bfca-5618-1879-9863dc6767f1"
  
  volume {
    name                         = "acctest-NetAppVolume-1-%[2]d"
    volume_path                  = "my-unique-file-path-1-%[2]d"
    service_level                = "Standard"
    capacity_pool_id             = azurerm_netapp_pool.test.id
    subnet_id                    = azurerm_subnet.test.id
    proximity_placement_group_id = azurerm_proximity_placement_group.test.id
    volume_spec_name             = "data"
    storage_quota_in_gb          = 1024
    throughput_in_mibps          = 24
    
    export_policy_rule {
      rule_index            = 1
      allowed_clients       = ["0.0.0.0/0"]
      protocols_enabled     = ["NFSv3"]
      unix_read_only        = false
      unix_read_write       = true
      kerberos5_read_only   = false
			kerberos5_read_write  = false
			kerberos5i_read_only  = false
			kerberos5i_read_write = false
			kerberos5p_read_only  = false
			kerberos5p_read_write = false
    }
  
    tags = {
      "SkipASMAzSecPack" = "true"
    }
  }

  volume {
    name                         = "acctest-NetAppVolume-2-%[2]d"
    volume_path                  = "my-unique-file-path-2-%[2]d"
    service_level                = "Standard"
    capacity_pool_id             = azurerm_netapp_pool.test.id
    subnet_id                    = azurerm_subnet.test.id
    proximity_placement_group_id = azurerm_proximity_placement_group.test.id
    volume_spec_name             = "log"
    storage_quota_in_gb          = 1024
    throughput_in_mibps          = 24
    
    export_policy_rule {
      rule_index            = 1
      allowed_clients       = ["0.0.0.0/0"]
      protocols_enabled     = ["NFSv3"]
      unix_read_only        = false
      unix_read_write       = true
      kerberos5_read_only   = false
			kerberos5_read_write  = false
			kerberos5i_read_only  = false
			kerberos5i_read_write = false
			kerberos5p_read_only  = false
			kerberos5p_read_write = false
    }
  
    tags = {
      "SkipASMAzSecPack" = "true"
    }
  }

  volume {
    name                         = "acctest-NetAppVolume-3-%[2]d"
    volume_path                  = "my-unique-file-path-3-%[2]d"
    service_level                = "Standard"
    capacity_pool_id             = azurerm_netapp_pool.test.id
    subnet_id                    = azurerm_subnet.test.id
    proximity_placement_group_id = azurerm_proximity_placement_group.test.id
    volume_spec_name             = "shared"
    storage_quota_in_gb          = 1024
    throughput_in_mibps          = 24
    
    export_policy_rule {
      rule_index            = 1
      allowed_clients       = ["0.0.0.0/0"]
      protocols_enabled     = ["NFSv3"]
      unix_read_only        = false
      unix_read_write       = true
      kerberos5_read_only   = false
			kerberos5_read_write  = false
			kerberos5i_read_only  = false
			kerberos5i_read_write = false
			kerberos5p_read_only  = false
			kerberos5p_read_write = false
    }
  
    tags = {
      "SkipASMAzSecPack" = "true"
    }
  }

  volume {
    name                         = "acctest-NetAppVolume-4-%[2]d"
    volume_path                  = "my-unique-file-path-4-%[2]d"
    service_level                = "Standard"
    capacity_pool_id             = azurerm_netapp_pool.test.id
    subnet_id                    = azurerm_subnet.test.id
    proximity_placement_group_id = azurerm_proximity_placement_group.test.id
    volume_spec_name             = "data-backup"
    storage_quota_in_gb          = 1024
    throughput_in_mibps          = 24
    
    export_policy_rule {
      rule_index            = 1
      allowed_clients       = ["0.0.0.0/0"]
      protocols_enabled     = ["NFSv3"]
      unix_read_only        = false
      unix_read_write       = true
      kerberos5_read_only   = false
			kerberos5_read_write  = false
			kerberos5i_read_only  = false
			kerberos5i_read_write = false
			kerberos5p_read_only  = false
			kerberos5p_read_write = false
    }
  
    tags = {
      "SkipASMAzSecPack" = "true"
    }
  }

  volume {
    name                         = "acctest-NetAppVolume-5-%[2]d"
    volume_path                  = "my-unique-file-path-5-%[2]d"
    service_level                = "Standard"
    capacity_pool_id             = azurerm_netapp_pool.test.id
    subnet_id                    = azurerm_subnet.test.id
    proximity_placement_group_id = azurerm_proximity_placement_group.test.id
    volume_spec_name             = "log-backup"
    storage_quota_in_gb          = 1024
    throughput_in_mibps          = 24
    
    export_policy_rule {
      rule_index            = 1
      allowed_clients       = ["0.0.0.0/0"]
      protocols_enabled     = ["NFSv3"]
      unix_read_only        = false
      unix_read_write       = true
      kerberos5_read_only   = false
			kerberos5_read_write  = false
			kerberos5i_read_only  = false
			kerberos5i_read_write = false
			kerberos5p_read_only  = false
			kerberos5p_read_write = false
    }
  
    tags = {
      "SkipASMAzSecPack" = "true"
    }
  }

  tags = {
    "SkipASMAzSecPack" = "true"
  }

  depends_on = [
    azurerm_linux_virtual_machine.test,
    azurerm_proximity_placement_group.test
  ]
}
`, template, data.RandomInteger)
}

func (NetAppVolumeGroupResource) templatePPG(data acceptance.TestData) string {
	return fmt.Sprintf(`
provider "azurerm" {
  alias = "all2"
  features {
    resource_group {
      prevent_deletion_if_contains_resources = false
    }
  }
}

locals {
  admin_username    = "testadmin%[1]d"
  admin_password    = "Password1234!%[1]d"
}

resource "azurerm_resource_group" "test" {
  name     = "acctestRG-netapp-%[1]d"
  location = "%[2]s"

  tags = {
    "SkipASMAzSecPack" = "true"
  }
}

resource "azurerm_network_security_group" "test" {
  name                = "acctest-NSG-%[1]d"
  location            = azurerm_resource_group.test.location
  resource_group_name = azurerm_resource_group.test.name

  tags = {
    environment        = "Production",
    "SkipASMAzSecPack" = "true"
  }
}

resource "azurerm_virtual_network" "test" {
  name                = "acctest-VirtualNetwork-%[1]d"
  location            = azurerm_resource_group.test.location
  resource_group_name = azurerm_resource_group.test.name
  address_space       = ["10.6.0.0/16"]

  tags = {
    "SkipASMAzSecPack" = "true"
  }
}

resource "azurerm_subnet" "test" {
  name                 = "acctest-DelegatedSubnet-%[1]d"
  resource_group_name  = azurerm_resource_group.test.name
  virtual_network_name = azurerm_virtual_network.test.name
  address_prefixes     = ["10.6.2.0/24"]

  delegation {
    name = "testdelegation"

    service_delegation {
      name    = "Microsoft.Netapp/volumes"
      actions = ["Microsoft.Network/networkinterfaces/*", "Microsoft.Network/virtualNetworks/subnets/join/action"]
    }
  }
}

resource "azurerm_subnet" "test1" {
  name                      = "acctest-HostsSubnet-%[1]d"
  resource_group_name       = azurerm_resource_group.test.name
  virtual_network_name      = azurerm_virtual_network.test.name
  address_prefixes          = ["10.6.1.0/24"]
}

resource "azurerm_subnet_network_security_group_association" "public" {
  subnet_id                 = azurerm_subnet.test.id
  network_security_group_id = azurerm_network_security_group.test.id
}

resource "azurerm_proximity_placement_group" "test" {
  name                = "acctest-PPG-%[1]d"
  location            = azurerm_resource_group.test.location
  resource_group_name = azurerm_resource_group.test.name
}

resource "azurerm_availability_set" "test" {
  name                = "acctest-avset-%[1]d"
  location            = azurerm_resource_group.test.location
  resource_group_name = azurerm_resource_group.test.name

  proximity_placement_group_id = azurerm_proximity_placement_group.test.id
}

resource "azurerm_network_interface" "test" {
  name                = "acctest-nic-%[1]d"
  resource_group_name = azurerm_resource_group.test.name
  location            = azurerm_resource_group.test.location

  ip_configuration {
    name                          = "internal"
    subnet_id                     = azurerm_subnet.test1.id
    private_ip_address_allocation = "Dynamic"
  }
}

resource "azurerm_linux_virtual_machine" "test" {
  name                            = "acctest-vm-%[1]d"
  resource_group_name             = azurerm_resource_group.test.name
  location                        = azurerm_resource_group.test.location
  size                            = "Standard_M8ms"
  admin_username                  = local.admin_username
  admin_password                  = local.admin_password
  disable_password_authentication = false
  proximity_placement_group_id    = azurerm_proximity_placement_group.test.id
  availability_set_id             = azurerm_availability_set.test.id
  network_interface_ids = [
    azurerm_network_interface.test.id
  ]

  source_image_reference {
    publisher = "Canonical"
    offer     = "UbuntuServer"
    sku       = "18.04-LTS"
    version   = "latest"
  }

  os_disk {
    storage_account_type = "Standard_LRS"
    caching              = "ReadWrite"
  }

  tags = {
      "Owner" = "pmarques"
  }
}

resource "azurerm_netapp_account" "test" {
  name                = "acctest-NetAppAccount-%[1]d"
  location            = azurerm_resource_group.test.location
  resource_group_name = azurerm_resource_group.test.name

  depends_on = [
    azurerm_subnet.test,
    azurerm_subnet.test1
  ]

  tags = {
    "SkipASMAzSecPack" = "true"
  }
}

resource "azurerm_netapp_pool" "test" {
  name                = "acctest-NetAppPool-%[1]d"
  location            = azurerm_resource_group.test.location
  resource_group_name = azurerm_resource_group.test.name
  account_name        = azurerm_netapp_account.test.name
  service_level       = "Standard"
  size_in_tb          = 8
  qos_type            = "Manual"

  tags = {
    "SkipASMAzSecPack" = "true"
  }
}
`, data.RandomInteger, "westus")
}
