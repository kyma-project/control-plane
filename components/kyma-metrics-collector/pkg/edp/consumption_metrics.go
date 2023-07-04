package edp

type ConsumptionMetrics struct {
	RuntimeId    string     `json:"runtime_id" validate:"required"`
	SubAccountId string     `json:"sub_account_id" validate:"required"`
	ShootName    string     `json:"shoot_name" validate:"required"`
	Timestamp    string     `json:"timestamp" validate:"required"`
	Compute      Compute    `json:"compute" validate:"required"`
	Networking   Networking `json:"networking" validate:"required"`
}
type Networking struct {
	ProvisionedVnets int `json:"provisioned_vnets" validate:"numeric"`
	ProvisionedIPs   int `json:"provisioned_ips" validate:"numeric"`
}

type VMType struct {
	Name  string `json:"name" validate:"required"`
	Count int    `json:"count" validate:"numeric"`
}

type Compute struct {
	VMTypes            []VMType           `json:"vm_types" validate:"required"`
	ProvisionedCpus    int                `json:"provisioned_cpus" validate:"numeric"`
	ProvisionedRAMGb   float64            `json:"provisioned_ram_gb" validate:"numeric"`
	ProvisionedVolumes ProvisionedVolumes `json:"provisioned_volumes" validate:"required"`
}

type ProvisionedVolumes struct {
	SizeGbTotal   int64 `json:"size_gb_total" validate:"numeric"`
	Count         int   `json:"count" validate:"numeric"`
	SizeGbRounded int64 `json:"size_gb_rounded" validate:"numeric"`
}
