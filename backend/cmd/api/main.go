package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"github.com/gov-spending/backend/docs"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/gov-spending/backend/internal/config"
	"github.com/gov-spending/backend/internal/handlers"
	"github.com/gov-spending/backend/internal/middleware"
	"github.com/gov-spending/backend/internal/services"
	"github.com/gov-spending/backend/pkg/fabric"
)

// @title           Government Spending Blockchain API
// @version         1.0
// @description     API for managing government spending documents across Federal, State, and Municipal channels
// @description     Uses Hyperledger Fabric blockchain for immutable, transparent spending records

// @BasePath  /

// @tag.name Health
// @tag.description Health check endpoints

// @tag.name Document Types
// @tag.description Manage document type templates

// @tag.name Documents
// @tag.description Create and query spending documents

// @tag.name Transfers
// @tag.description Cross-channel inter-government transfers

// @tag.name Verification
// @tag.description Verify cross-channel links

func main() {
	configPath := flag.String("config", "", "Path to configuration file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load configuration")
	}

	setupLogging(cfg.Logging)

	log.Info().Msg("Starting Government Spending Blockchain API")
	log.Info().Str("networkPath", cfg.Fabric.NetworkPath).Msg("Fabric network path")

	gatewayManager := fabric.NewGatewayManager(cfg)
	defer gatewayManager.Close()

	fabricService := services.NewFabricService(gatewayManager)

	handler := handlers.NewHandler(fabricService, cfg)

	if swaggerHost := os.Getenv("SWAGGER_HOST"); swaggerHost != "" {
		docs.SwaggerInfo.Host = swaggerHost
	} else {
		docs.SwaggerInfo.Host = ""
	}

	router := setupRouter(cfg, handler)

	srv := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Info().Str("port", cfg.Server.Port).Msg("Server starting")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("Server failed to start")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("Server forced to shutdown")
	}

	log.Info().Msg("Server stopped")
}

func setupLogging(cfg config.LoggingConfig) {
	level, err := zerolog.ParseLevel(cfg.Level)
	if err != nil {
		level = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(level)

	if cfg.Format == "console" {
		log.Logger = log.Output(zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: "15:04:05",
		})
	}

	if level <= zerolog.DebugLevel {
		log.Logger = log.With().Caller().Logger()
	}
}

func setupRouter(cfg *config.Config, h *handlers.Handler) *gin.Engine {
	gin.SetMode(cfg.Server.Mode)

	router := gin.New()

	router.Use(middleware.RequestID())
	router.Use(middleware.Recovery())
	router.Use(middleware.Logger())
	router.Use(middleware.CORS())

	router.GET("/health", h.HealthCheck)
	router.GET("/config", h.ConfigInfo)

	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	api := router.Group("/api")
	{

		api.POST("/transfers/initiate", h.InitiateTransfer)

		api.POST("/anchors/verify", h.VerifyAnchor)

		channel := api.Group("/:channel")
		{

			docTypes := channel.Group("/document-types")
			{
				docTypes.POST("", h.RegisterDocumentType)
				docTypes.GET("", h.ListDocumentTypes)
				docTypes.GET("/:typeId", h.GetDocumentType)
				docTypes.DELETE("/:typeId", h.DeactivateDocumentType)
			}

			docs := channel.Group("/documents")
			{
				docs.POST("", h.CreateDocument)
				docs.GET("", h.QueryDocuments)
				docs.GET("/:docId", h.GetDocument)
				docs.GET("/:docId/history", h.GetDocumentHistory)
				docs.GET("/:docId/linked", h.GetLinkedDocuments)
				docs.POST("/:docId/invalidate", h.InvalidateDocument)
			}

			transfers := channel.Group("/transfers")
			{
				transfers.POST("/acknowledge", h.AcknowledgeTransfer)
			}
		}
	}

	return router
}
