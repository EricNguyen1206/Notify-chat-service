package response

const (
	AuthLoginSuccess = 1 // Login Success
	AuthLoginFailed  = 2 // Login Failed

	ErrCodeSuccess      = 4001 // Success
	ErrCodeParamInvalid = 4003 // Email invalid
)

// message
var msg = map[int]string{
	// Auth
	AuthLoginSuccess: "login success",
	AuthLoginFailed:  "login failed",

	// User

	ErrCodeSuccess:      "success",
	ErrCodeParamInvalid: "email is invalid",
}
