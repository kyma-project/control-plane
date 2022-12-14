package internal

const (
	BTPRegionSwitzerlandAzure = "cf-ch20"
	BTPRegionEuropeAWS        = "cf-eu11"
)

func IsEURestrictedAccess(platformRegion string) bool {
	switch platformRegion {
	case BTPRegionSwitzerlandAzure, BTPRegionEuropeAWS:
		return true
	default:
		return false
	}
}
