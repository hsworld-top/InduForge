package api

const (
	ErrorCodeTokenRequired                 = 10001
	ErrorCodeTokenInvalid                  = 10002
	ErrorCodeTokenExpired                  = 10003
	ErrorCodeUserUnavailable               = 10004
	ErrorCodeInvalidCredentials            = 10007
	ErrorCodeTenantRequired                = 10008
	ErrorCodeInvalidCaptcha                = 10010
	ErrorCodePermissionDenied              = 11001
	ErrorCodeInvalidRequest                = 20001
	ErrorCodeNotFound                      = 21001
	ErrorCodeAlreadyExists                 = 21002
	ErrorCodeTenantNotFound                = 22001
	ErrorCodeTenantCodeExists              = 22002
	ErrorCodeUserNotFound                  = 23001
	ErrorCodeUsernameExists                = 23002
	ErrorCodeProjectNotFound               = 24001
	ErrorCodeProjectCodeExists             = 24002
	ErrorCodeSceneNotCommitted             = 26001
	ErrorCodeSceneContractInvalid          = 26002
	ErrorCodeSceneContractViolation        = 26003
	ErrorCodeSceneCommandNotDeclared       = 26004
	ErrorCodeSceneCommandNotRegistered     = 26005
	ErrorCodeSceneCommandTimeout           = 26006
	ErrorCodeSceneDatapointNotDeclared     = 26007
	ErrorCodeSceneDatapointReadFailed      = 26008
	ErrorCodeSceneDatapointSubscribeFailed = 26009
	ErrorCodeSceneDatapointWriteForbidden  = 26010
	ErrorCodeSceneDatapointWriteFailed     = 26011
	ErrorCodeInternal                      = 30001
)
