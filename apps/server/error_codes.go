package main

// ValidationError represents a validation error with a code for frontend localization
type ValidationError struct {
	Code    string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

// Error codes for frontend localization matching
// These codes are sent in the API response and matched on the frontend
// to display the appropriate localized error message
const (
	// Authentication errors
	ErrCodeUnauthorized       = "ERR_UNAUTHORIZED"
	ErrCodeInvalidCredentials = "ERR_INVALID_CREDENTIALS"
	ErrCodeSessionExpired     = "ERR_SESSION_EXPIRED"
	ErrCodeForbidden          = "ERR_FORBIDDEN"

	// Validation errors
	ErrCodeInvalidInput      = "ERR_INVALID_INPUT"
	ErrCodeMissingField      = "ERR_MISSING_FIELD"
	ErrCodeInvalidMAC        = "ERR_INVALID_MAC"
	ErrCodeInvalidIP         = "ERR_INVALID_IP"
	ErrCodeInvalidBroadcast  = "ERR_INVALID_BROADCAST"
	ErrCodeInvalidInterface  = "ERR_INVALID_INTERFACE"
	ErrCodeInvalidName       = "ERR_INVALID_NAME"
	ErrCodeNameTooLong       = "ERR_NAME_TOO_LONG"
	ErrCodeDescriptionTooLong = "ERR_DESCRIPTION_TOO_LONG"

	// Resource errors
	ErrCodeNotFound         = "ERR_NOT_FOUND"
	ErrCodeAlreadyExists    = "ERR_ALREADY_EXISTS"
	ErrCodeCannotDelete     = "ERR_CANNOT_DELETE"
	ErrCodeCannotUpdate     = "ERR_CANNOT_UPDATE"

	// Operation errors
	ErrCodeReadOnlyMode     = "ERR_READONLY_MODE"
	ErrCodeRateLimited      = "ERR_RATE_LIMITED"
	ErrCodeOperationFailed  = "ERR_OPERATION_FAILED"
	ErrCodeNetworkError     = "ERR_NETWORK_ERROR"
	ErrCodeDatabaseError    = "ERR_DATABASE_ERROR"

	// Wake-on-LAN errors
	ErrCodeWakeFailed       = "ERR_WAKE_FAILED"
	ErrCodePingFailed       = "ERR_PING_FAILED"
	ErrCodeInterfaceNotFound = "ERR_INTERFACE_NOT_FOUND"

	// User management errors
	ErrCodeUserNotFound                = "ERR_USER_NOT_FOUND"
	ErrCodeUserExists                  = "ERR_USER_EXISTS"
	ErrCodeCannotRemoveSelf            = "ERR_CANNOT_REMOVE_SELF"
	ErrCodeCannotRemoveOwnSuperuser    = "ERR_CANNOT_REMOVE_OWN_SUPERUSER"
	ErrCodeCannotRemoveLastSuperuser   = "ERR_CANNOT_REMOVE_LAST_SUPERUSER"
	ErrCodeCannotDeleteSelf            = "ERR_CANNOT_DELETE_SELF"
	ErrCodeCannotDeleteLastSuperuser   = "ERR_CANNOT_DELETE_LAST_SUPERUSER"
	ErrCodeCannotChangeSuperuserPassword = "ERR_CANNOT_CHANGE_SUPERUSER_PASSWORD"
	ErrCodePasswordTooShort            = "ERR_PASSWORD_TOO_SHORT"
	ErrCodeUsernameTooShort            = "ERR_USERNAME_TOO_SHORT"

	// Host errors
	ErrCodeHostNotFound     = "ERR_HOST_NOT_FOUND"
	ErrCodeHostExists       = "ERR_HOST_EXISTS"
)
