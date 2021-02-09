package dbmodel

import "time"

type CLSInstanceDTO struct {
	ID              string
	GlobalAccountID string
	CreatedAt       time.Time
}
