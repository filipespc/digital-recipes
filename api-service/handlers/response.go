package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// StandardResponse represents the standard API response format
type StandardResponse struct {
	Data       interface{} `json:"data"`
	Pagination *Pagination `json:"pagination,omitempty"`
	Meta       *Meta       `json:"meta,omitempty"`
	Error      *string     `json:"error,omitempty"`
}

// Pagination contains pagination metadata
type Pagination struct {
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

// Meta contains additional response metadata
type Meta struct {
	RequestID string `json:"request_id,omitempty"`
	Timestamp string `json:"timestamp,omitempty"`
}

// SuccessResponse sends a standardized success response
func SuccessResponse(c *gin.Context, data interface{}) {
	response := StandardResponse{
		Data: data,
	}
	c.JSON(http.StatusOK, response)
}

// SuccessResponseWithPagination sends a standardized success response with pagination
func SuccessResponseWithPagination(c *gin.Context, data interface{}, pagination *Pagination) {
	response := StandardResponse{
		Data:       data,
		Pagination: pagination,
	}
	c.JSON(http.StatusOK, response)
}

// ErrorResponse sends a standardized error response
func ErrorResponse(c *gin.Context, statusCode int, message string) {
	response := StandardResponse{
		Error: &message,
	}
	c.JSON(statusCode, response)
}

// BadRequestError sends a 400 error response
func BadRequestError(c *gin.Context, message string) {
	ErrorResponse(c, http.StatusBadRequest, message)
}

// NotFoundError sends a 404 error response
func NotFoundError(c *gin.Context, message string) {
	ErrorResponse(c, http.StatusNotFound, message)
}

// InternalServerError sends a 500 error response
func InternalServerError(c *gin.Context, message string) {
	ErrorResponse(c, http.StatusInternalServerError, message)
}