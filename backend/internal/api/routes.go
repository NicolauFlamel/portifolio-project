package api

import (
    "github.com/gin-gonic/gin"
    swaggerFiles "github.com/swaggo/files"
    ginSwagger "github.com/swaggo/gin-swagger"
)

func RegisterRoutes(r *gin.Engine) {
    h := NewHandler()

    // Swagger UI
    r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

    // Health
    r.GET("/health", h.Health)

    // Documents
    r.GET("/api/docs", h.GetAllDocs)
    r.GET("/api/docs/:id", h.GetDoc)
    r.POST("/api/docs", h.CreateDoc)
}