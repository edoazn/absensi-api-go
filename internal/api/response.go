package api

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type Envelope struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Message string      `json:"message,omitempty"`
	Errors  interface{} `json:"errors,omitempty"`
}

func OK(c *gin.Context, data interface{}, message string) {
	c.JSON(http.StatusOK, Envelope{Success: true, Data: data, Message: message})
}

func Created(c *gin.Context, data interface{}, message string) {
	c.JSON(http.StatusCreated, Envelope{Success: true, Data: data, Message: message})
}

func Fail(c *gin.Context, status int, message string) {
	c.AbortWithStatusJSON(status, Envelope{Success: false, Message: message})
}

func FailWithErrors(c *gin.Context, status int, message string, errs interface{}) {
	c.AbortWithStatusJSON(status, Envelope{Success: false, Message: message, Errors: errs})
}

func BindJSON(c *gin.Context, dst interface{}) bool {
	if err := c.ShouldBindJSON(dst); err != nil {
		FailWithErrors(c, http.StatusUnprocessableEntity, "Data yang dikirim tidak valid", validationErrors(err))
		return false
	}
	return true
}

func validationErrors(err error) map[string]string {
	out := map[string]string{}
	var ve validator.ValidationErrors
	if errors.As(err, &ve) {
		for _, f := range ve {
			out[f.Field()] = msgForTag(f.Tag())
		}
		return out
	}
	out["body"] = "Format JSON tidak valid"
	return out
}

func msgForTag(tag string) string {
	switch tag {
	case "required":
		return "wajib diisi"
	case "email":
		return "format email tidak valid"
	default:
		return "tidak valid"
	}
}
