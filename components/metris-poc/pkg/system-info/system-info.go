package system_info

type Information struct {
	skrShoots []Shoot
}

type Shoot struct {
	name string
}

func GetSystemInfo() (Information, error) {
	return Information{}, nil
}
