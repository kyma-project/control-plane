package dbmodel

import "time"

type CLSInstanceDTO struct {
	ID        string
	Name      string
	Region    string
	CreatedAt time.Time
}
