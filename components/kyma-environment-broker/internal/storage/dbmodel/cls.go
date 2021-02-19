package dbmodel

import (
	"database/sql"
	"time"
)

type CLSInstanceDTO struct {
	Version                int
	ID                     string
	GlobalAccountID        string
	Region                 string
	RemovedBySKRInstanceID sql.NullString
	CreatedAt              time.Time
	SKRInstanceID          string
}

type CLSInstanceReferenceDTO struct {
	ID            string
	CLSInstanceID string
	SKRInstanceID string
}
