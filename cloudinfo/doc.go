// Package cloudinfo provides access to the YellowDog Cloud Information API.
//
// It exposes read-only metadata about cloud providers: available regions,
// supported instance types, and on-demand pricing. This information is
// useful for building dynamic compute source configurations.
//
// Use [CloudInfoClient] (available as client.CloudInfo) to query cloud metadata:
//
//	regions, _ := client.CloudInfo.GetRegions(&cloudinfo.RegionSearch{
//	    Providers: []model.CloudProvider{model.CloudProviderAws},
//	}).ListAll(ctx)
//	types, _ := client.CloudInfo.GetInstanceTypes(&cloudinfo.InstanceTypeSearch{
//	    Providers: []model.CloudProvider{model.CloudProviderAws},
//	    Region:    optional.Of("eu-west-2"),
//	}).ListAll(ctx)
package cloudinfo
