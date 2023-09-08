package networking

const (
	DefaultNodesCIDR    = "10.250.0.0/22"
	DefaultPodsCIDR     = "10.96.0.0/13"
	DefaultServicesCIDR = "10.104.0.0/13"
)

var GardenerSeedCIDRs = []string{"10.243.128.0/17", "10.242.0.0/16", "10.243.0.0/17", "10.64.0.0/11", "10.254.0.0/16", "10.243.0.0/16"}
