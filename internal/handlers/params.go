package handlers

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

func paramUint(c *gin.Context, name string) (uint, bool) {
	id, err := strconv.ParseUint(c.Param(name), 10, 64)
	if err != nil {
		return 0, false
	}
	return uint(id), true
}
