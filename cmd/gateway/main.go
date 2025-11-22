package main

import (
	"fmt"
	filehandle "kates3/gateway/handlers/file"
	fileservice "kates3/gateway/service/file"
	"log"
	"log/slog"
	"os"

	"github.com/gin-gonic/gin"
)

const (
	storageDir = "files"
)

func main() {

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	fileService := fileservice.New(storageDir)

	uploadHandler := filehandle.NewUploader(logger, fileService)
	downloadHandler := filehandle.NewDownloader(logger, fileService)

	router := gin.Default()

	// Middleware
	router.Use(gin.Recovery())
	api := router.Group("/api/v1")
	{
		api.POST("/upload", uploadHandler.Upload)
		api.GET("/download/:id", downloadHandler.Download)
		api.GET("/health", func(c *gin.Context) {
			c.JSON(200, gin.H{"status": "ok"})
		})
	}

	// Запуск сервера
	port := ":8080"
	addr := fmt.Sprintf("http://localhost%s", port)
	logger.Debug("Starting server", slog.String("addr", addr))

	if err := router.Run(port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
