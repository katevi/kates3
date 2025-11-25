package chunk

import (
	"fmt"
	"io"
)

type Chunk struct {
	ID   string
	Data *io.PipeReader
	Size int64
}

type AccurateManager struct {
	chunkCount int
	bufferSize int
}

func NewAccurateManager(chunkCount int) *AccurateManager {
	return &AccurateManager{
		chunkCount: chunkCount,
		bufferSize: 64 * 1024, // 64KB
	}
}

func (m *AccurateManager) Split(file io.Reader, totalSize int64, fileID string) ([]Chunk, error) {
	chunkSize := totalSize / int64(m.chunkCount)
	chunks := make([]Chunk, m.chunkCount)

	for i := 0; i < m.chunkCount; i++ {
		// Вычисляем размер этого чанка
		start := int64(i) * chunkSize
		end := start + chunkSize
		if i == m.chunkCount-1 {
			end = totalSize // последний чанк получает остаток
		}
		currentChunkSize := end - start

		// Создаем ограниченный reader для этого чанка
		limitedReader := &io.LimitedReader{R: file, N: currentChunkSize}

		reader, writer := io.Pipe()

		chunks[i] = Chunk{
			ID:   fmt.Sprintf("%s-chunk-%d", fileID, i),
			Data: reader,
			Size: currentChunkSize,
		}

		// Запускаем копирование данных в пайп
		go m.pumpData(limitedReader, writer, currentChunkSize)
	}

	return chunks, nil
}

func (m *AccurateManager) pumpData(reader io.Reader, writer *io.PipeWriter, chunkSize int64) {
	defer writer.Close()

	buffer := make([]byte, m.bufferSize)
	var totalWritten int64

	for {
		n, err := reader.Read(buffer)
		if n > 0 {
			if _, writeErr := writer.Write(buffer[:n]); writeErr != nil {
				return // пайп закрыт
			}
			totalWritten += int64(n)
		}

		if err != nil {
			if err != io.EOF {
				writer.CloseWithError(err)
			}
			break
		}

		// Защита от переполнения
		if totalWritten >= chunkSize {
			break
		}
	}
}

func (m *AccurateManager) Join(chunkReaders []io.ReadCloser, writer io.Writer) error {
	for _, reader := range chunkReaders {
		if _, err := io.Copy(writer, reader); err != nil {
			reader.Close()
			return err
		}
		reader.Close()
	}
	return nil
}
