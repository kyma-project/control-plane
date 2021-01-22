package system_info

type Information struct {
	skrShoots []Shoot
}

type Shoot struct {
	name string
}

func GetShootInfo() (Information, error) {
	return Information{}, nil
}
