package fileservice

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type service struct {
	storageDir string
	files      map[string]string // fileID -> filePath
	mu         sync.RWMutex
}

func New(
	storageDir string,
) *service {
	os.MkdirAll(storageDir, 0755)

	return &service{
		storageDir: storageDir,
		files:      make(map[string]string),
	}
}

func (s *service) Upload(
	ctx context.Context,
	file io.Reader,
	size int64,
) (string, error) {
	// Генерируем ID файла
	fileID := generateFileID()
	filePath := filepath.Join(s.storageDir, fileID)

	// Создаем файл
	outFile, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}
	defer outFile.Close()

	// Копируем данные из reader в файл
	bytesWritten, err := io.Copy(outFile, file)
	if err != nil {
		// Удаляем файл если ошибка
		os.Remove(filePath)
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	// Сохраняем в мапе
	s.mu.Lock()
	s.files[fileID] = filePath
	s.mu.Unlock()

	fmt.Printf("Uploaded file %s (%d bytes)\n", fileID, bytesWritten)
	return fileID, nil
}

func (s *service) Download(
	ctx context.Context,
	fileID string,
) (io.ReadCloser, int64, error) {
	s.mu.RLock()
	filePath, exists := s.files[fileID]
	s.mu.RUnlock()

	if !exists {
		// Проверяем есть ли файл на диске
		filePath = filepath.Join(s.storageDir, fileID)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			return nil, 0, fmt.Errorf("file not found")
		}
		// Добавляем в мапу если нашли на диске
		s.mu.Lock()
		s.files[fileID] = filePath
		s.mu.Unlock()
	}

	// Открываем файл
	file, err := os.Open(filePath)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to open file: %w", err)
	}

	// Получаем размер файла
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, 0, fmt.Errorf("failed to get file info: %w", err)
	}

	return file, info.Size(), nil
}

func generateFileID() string {
	hash := sha256.Sum256([]byte(time.Now().String() + fmt.Sprintf("%d", time.Now().UnixNano())))
	return hex.EncodeToString(hash[:16])
}
