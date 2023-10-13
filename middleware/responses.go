package middleware

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"os"
)

type GenericResponse struct {
	Success bool `json:"success"`
}

func RespondOK(c *gin.Context, obj any) {
	type Response struct {
		Data any `json:"data,omitempty"`
	}

	c.JSON(http.StatusOK, Response{Data: obj})
}

func RespondErr(c *gin.Context, error APIError, reason string) {
	type response struct {
		Error  string `json:"error"`
		Reason string `json:"reason"`
	}

	r := response{Error: error.Error()}
	if os.Getenv("ENVIRONMENT") != "production" {
		r.Reason = reason
	}

	if error == APIErrorUnknown {
		c.AbortWithStatusJSON(http.StatusInternalServerError, r)
	} else if error == APIErrorNotFound {
		c.AbortWithStatusJSON(http.StatusNotFound, r)
	} else if error == APIErrorUnauthorizedDevice {
		c.AbortWithStatusJSON(http.StatusUnauthorized, r)
	} else {
		c.AbortWithStatusJSON(http.StatusBadRequest, r)
	}
}
