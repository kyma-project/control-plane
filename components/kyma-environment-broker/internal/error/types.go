package error

type ErrorReason string

const (
	ErrorDBNotFound      ErrorReason = "ERR_DB_NOT_FOUND"
	ErrorDBInternal      ErrorReason = "ERR_DB_INTERNAL"
	ErrorDBAlreadyExists ErrorReason = "ERR_DB_ALREADY_EXISTS"
	ErrorDBConflict      ErrorReason = "ERR_DB_CONFLICT"
	ErrorDBUnknown       ErrorReason = "ERR_DB_UNKNOWN"

	ErrorKEBInternal ErrorReason = "ERR_KEB_INTERNAL"
	KEBTimeOut       ErrorReason = "ERR_KEB_TIMEOUT"

	ErrorOther ErrorReason = "ERR_UNKNOWN"
)
