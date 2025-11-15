package main

import (
    "log"

		"github.com/gin-gonic/gin"
    "github.com/NicolauFlamel/portifolio-project-backend/internal/api"
    _ "github.com/NicolauFlamel/portifolio-project-backend/internal/docs"
)

// @title PublicDocs Blockchain API
// @version 1.0
// @description Multi-channel Hyperledger Fabric document API.
// @BasePath /
func main() {
    r := gin.Default()
    api.RegisterRoutes(r)

    log.Println("HTTP server on :8080")
    if err := r.Run(":8080"); err != nil {
        log.Fatal(err)
    }
}