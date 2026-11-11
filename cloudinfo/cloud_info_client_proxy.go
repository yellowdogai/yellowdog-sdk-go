package cloudinfo

import (
	"github.com/yellowdogai/yellowdog-sdk-go/internal"
)

// cloudInfoClientProxy makes every request the CloudInfo client sends. Each method backs the
// CloudInfoClient method of the same name with its first letter lowercased.
type cloudInfoClientProxy struct {
	http *internal.HTTPClient
}

func newCloudInfoClientProxy(httpClient *internal.HTTPClient) *cloudInfoClientProxy {
	return &cloudInfoClientProxy{http: httpClient}
}

// The service path is camelCase, unlike every other service.
const cloudInfoServicePath = "/cloudInfo"

func (p *cloudInfoClientProxy) getRegions(regionSearch *RegionSearch) *internal.SearchClient[Region] {
	return internal.NewSearchClient[Region](p.http, cloudInfoServicePath+"/regions", regionSearch)
}

func (p *cloudInfoClientProxy) getSubRegions(subRegionSearch *SubRegionSearch) *internal.SearchClient[SubRegion] {
	return internal.NewSearchClient[SubRegion](p.http, cloudInfoServicePath+"/subRegions", subRegionSearch)
}

func (p *cloudInfoClientProxy) getInstanceTypes(instanceTypeSearch *InstanceTypeSearch) *internal.SearchClient[InstanceType] {
	return internal.NewSearchClient[InstanceType](p.http, cloudInfoServicePath+"/instanceTypes", instanceTypeSearch)
}

func (p *cloudInfoClientProxy) getInstanceTypePrices(instanceTypePriceSearch *InstanceTypePriceSearch) *internal.SearchClient[InstanceTypePrice] {
	return internal.NewSearchClient[InstanceTypePrice](p.http, cloudInfoServicePath+"/instanceTypePrices", instanceTypePriceSearch)
}
