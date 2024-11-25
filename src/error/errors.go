package error

type ResponseError struct {
	StatusCode int    `json:"status_code"`
	Message    string `json:"message"`
	Reason     string `json:"reason"`
	Details    string `json:"details"`
}

func RespErr(statusCode int, message string, reason string, details string) *ResponseError {
	return &ResponseError{
		StatusCode: statusCode,
		Message:    message,
		Reason:     reason,
		Details:    details,
	}
}
