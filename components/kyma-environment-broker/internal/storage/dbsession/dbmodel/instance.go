package dbmodel

// InstanceFilter holds the filters when queryíing Instances
type InstanceFilter struct {
	PageSize         int
	Page             int
	GlobalAccountIDs []string
	SubAccountIDs    []string
	InstanceIDs      []string
	RuntimeIDs       []string
	Regions          []string
	Plans            []string
	Domains          []string
}
