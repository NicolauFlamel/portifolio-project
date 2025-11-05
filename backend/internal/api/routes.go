package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/NicolauFlamel/portifolio-project-backend/internal/utils"
)

func RegisterRoutes(r *gin.Engine) {
	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "Go backend running"})
	})

	r.GET("/hash", func(c *gin.Context) {
		data := c.Query("data")
		hash := utils.ComputeSHA256(data)
		c.JSON(http.StatusOK, gin.H{
			"input": data,
			"hash":  hash,
		})
	})
}