package filehandle

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

type FileService interface {
	Upload(
		ctx context.Context,
		file io.Reader,
		size int64,
	) (string, error)

	Download(
		ctx context.Context,
		fileID string,
	) (io.ReadCloser, int64, error)
}

type uploader struct {
	logger  *slog.Logger
	service FileService
}

func NewUploader(
	logger *slog.Logger,
	service FileService,
) uploader {
	return uploader{
		logger:  logger.With("component", "uploader"),
		service: service,
	}
}

// Upload обрабатывает бинарную загрузку файла
// Пример использования в Postman:
// - Method: POST
// - URL: http://localhost:8080/api/v1/upload
// - Body: binary -> выбираем файл
// - Headers: Content-Type: application/octet-stream
func (h *uploader) Upload(
	c *gin.Context,
) {
	// Получаем Content-Length для проверки размера
	contentLength := c.Request.ContentLength
	if contentLength < 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Content-Length header is required",
		})
		return
	}

	// Проверяем максимальный размер файла (10 GiB)
	maxSize := int64(10 * 1024 * 1024 * 1024) // 10 GiB
	if contentLength > maxSize {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{
			"error": fmt.Sprintf("File too large. Maximum size is %d bytes", maxSize),
		})
		return
	}

	// Получаем оригинальное имя файла из заголовка (опционально)
	fileName := c.GetHeader("X-File-Name")
	if fileName == "" {
		fileName = "unknown"
	}

	// Используем тело запроса как reader
	fileReader := c.Request.Body
	defer fileReader.Close()

	// Загружаем файл через сервис
	fileID, err := h.service.Upload(c.Request.Context(), fileReader, contentLength)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to upload file: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"fileId":   fileID,
		"fileName": fileName,
		"size":     contentLength,
		"status":   "uploaded",
	})
}
