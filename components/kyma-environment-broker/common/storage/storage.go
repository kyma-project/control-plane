package storage

import "database/sql"

func StringToSQLNullString(input string) sql.NullString {
	result := sql.NullString{}

	if input != "" {
		result.String = input
		result.Valid = true
	}

	return result
}

func SQLNullStringToString(input sql.NullString) string {
	if input.Valid {
		return input.String
	}

	return ""
}
