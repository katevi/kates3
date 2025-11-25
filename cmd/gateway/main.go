package main

import (
	"fmt"
	filehandle "kates3/internal/gateway/handlers/file"
	fileservice "kates3/internal/gateway/service/file"
	"kates3/internal/gateway/storage"
	"kates3/internal/gateway/storage/registry"
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

	storageRegistry := registry.NewSimpleRegistry()

	// Регистрируем 6 mock storage серверов
	for i := 1; i <= 6; i++ {
		server := &registry.StorageServer{
			ID:      fmt.Sprintf("storage%d", i),
			Address: fmt.Sprintf("localhost:900%d", i),
			Weight:  1,
		}
		storageRegistry.RegisterServer(server)
	}

	storageClient := storage.NewMockClient()

	fileService := fileservice.NewFileService(logger, storageRegistry, storageClient)

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
