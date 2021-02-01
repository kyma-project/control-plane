package dbmodel

import "time"

type CLSTenantDTO struct {
	ID        string
	Name      string
	Region    string
	CreatedAt time.Time
}
