package handler

import (
	"encoding/json"
	"log"
	"net/http"
)

type ResponseError struct {
	StatusCode int    `json:"status_code"`
	Message    string `json:"message"`
	Reason     string `json:"reason"`
	Details    string `json:"details"`
}

// Err creates a new ResponseError with the given status code, message and error details
//
// Parameters:
//   - code: HTTP status code for the error response
//   - message: Human-readable message describing the error
//   - err: The underlying error that occurred
//
// Returns:
//   - ResponseError containing the formatted error details
func Err(code int, message string, err error) ResponseError {
	var details string
	if err != nil {
		details = err.Error()
	}
	return ResponseError{
		StatusCode: code,
		Message:    message,
		Reason:     http.StatusText(code),
		Details:    details,
	}
}

func SendErr(w http.ResponseWriter, e ResponseError) {
	w.WriteHeader(e.StatusCode)
	_ = json.NewEncoder(w).Encode(e)
	log.Printf("Error sent to client: %v\n", e)
}
