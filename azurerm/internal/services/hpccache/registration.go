package hpccache

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

type Registration struct{}

// Name is the name of this Service
func (r Registration) Name() string {
	return "HPC Cache"
}

// WebsiteCategories returns a list of categories which can be used for the sidebar
func (r Registration) WebsiteCategories() []string {
	return []string{
		"Storage",
	}
}

// SupportedDataSources returns the supported Data Sources supported by this Service
func (r Registration) SupportedDataSources() map[string]*schema.Resource {
	return map[string]*schema.Resource{}
}

// SupportedResources returns the supported Resources supported by this Service
func (r Registration) SupportedResources() map[string]*schema.Resource {
	return map[string]*schema.Resource{
		"azurerm_hpc_cache":               resourceHPCCache(),
		"azurerm_hpc_cache_access_policy": resourceHPCCacheAccessPolicy(),
		"azurerm_hpc_cache_blob_target":   resourceHPCCacheBlobTarget(),
		"azurerm_hpc_cache_nfs_target":    resourceHPCCacheNFSTarget(),
	}
}
