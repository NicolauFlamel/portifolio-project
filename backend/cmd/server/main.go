package main

import (
	"log"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/NicolauFlamel/portifolio-project-backend/internal/api"
)

func main() {
	r := gin.Default()
	api.RegisterRoutes(r)

	fmt.Println("Backend running on :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}