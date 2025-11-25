package filehandle

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

type downloader struct {
	logger  *slog.Logger
	service FileService
}

func NewDownloader(
	logger *slog.Logger,
	service FileService,
) downloader {
	return downloader{
		logger:  logger.With("component", "downloader"),
		service: service,
	}
}

func (d *downloader) Download(
	c *gin.Context,
) {
	fileID := c.Param("id")
	if fileID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "File ID is required",
		})
		return
	}

	// Получаем reader для файла
	fileReader, fileSize, err := d.service.Download(c.Request.Context(), fileID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": fmt.Sprintf("File not found: %v", err),
		})
		return
	}
	defer fileReader.Close()

	// Определяем имя файла для скачивания
	fileName := fileID

	// Устанавливаем заголовки для скачивания
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fileName))
	c.Header("Content-Length", fmt.Sprintf("%d", fileSize))
	c.Header("Cache-Control", "no-cache")

	// Стримим файл клиенту
	_, err = io.Copy(c.Writer, fileReader)
	if err != nil {
		// Логируем ошибку, но не отправляем клиенту (соединение может быть разорвано)
		d.logger.Debug("Error streaming file ", "error", err)
		return
	}
}
