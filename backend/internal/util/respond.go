package util

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ErrorBody is the standard API error envelope (BRD §9).
type ErrorBody struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail holds a machine-readable code and human message.
type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// JSON writes a success payload wrapped in { "data": ... }.
func JSON(c *gin.Context, status int, data any) {
	c.JSON(status, gin.H{"data": data})
}

// Error writes the standard error envelope.
func Error(c *gin.Context, status int, code, message string) {
	c.JSON(status, ErrorBody{
		Error: ErrorDetail{Code: code, Message: message},
	})
}

// AbortError writes an error envelope and aborts the middleware chain.
func AbortError(c *gin.Context, status int, code, message string) {
	Error(c, status, code, message)
	c.Abort()
}

// BadRequest is a convenience for 400 VALIDATION_ERROR.
func BadRequest(c *gin.Context, message string) {
	Error(c, http.StatusBadRequest, "VALIDATION_ERROR", message)
}
