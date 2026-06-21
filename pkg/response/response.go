package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Meta holds pagination metadata returned with list responses.
type Meta struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
	Total  int `json:"total"`
}

// envelope is the standard JSON response shape used across all endpoints.
type envelope struct {
	Success bool   `json:"success"`
	Data    any    `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
	Meta    *Meta  `json:"meta,omitempty"`
}

// OK sends a 200 OK response with optional data payload.
func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, envelope{Success: true, Data: data})
}

// Created sends a 201 Created response.
func Created(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, envelope{Success: true, Data: data})
}

// NoContent sends a 204 No Content response (no body).
func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

// OKWithMeta sends a 200 OK response with pagination metadata.
func OKWithMeta(c *gin.Context, data any, meta Meta) {
	c.JSON(http.StatusOK, envelope{Success: true, Data: data, Meta: &meta})
}

// BadRequest sends a 400 Bad Request response.
func BadRequest(c *gin.Context, msg string) {
	c.JSON(http.StatusBadRequest, envelope{Success: false, Error: msg})
}

// Unauthorized sends a 401 Unauthorized response.
func Unauthorized(c *gin.Context, msg string) {
	c.JSON(http.StatusUnauthorized, envelope{Success: false, Error: msg})
}

// Forbidden sends a 403 Forbidden response.
func Forbidden(c *gin.Context, msg string) {
	c.JSON(http.StatusForbidden, envelope{Success: false, Error: msg})
}

// NotFound sends a 404 Not Found response.
func NotFound(c *gin.Context, msg string) {
	c.JSON(http.StatusNotFound, envelope{Success: false, Error: msg})
}

// Conflict sends a 409 Conflict response.
func Conflict(c *gin.Context, msg string) {
	c.JSON(http.StatusConflict, envelope{Success: false, Error: msg})
}

// UnprocessableEntity sends a 422 Unprocessable Entity response.
func UnprocessableEntity(c *gin.Context, msg string) {
	c.JSON(http.StatusUnprocessableEntity, envelope{Success: false, Error: msg})
}

// InternalServerError sends a 500 Internal Server Error response.
func InternalServerError(c *gin.Context, msg string) {
	c.JSON(http.StatusInternalServerError, envelope{Success: false, Error: msg})
}
