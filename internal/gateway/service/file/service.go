package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"time"

	"kates3/internal/gateway/chunk"
	"kates3/internal/gateway/storage"
	"kates3/internal/gateway/storage/registry"
)

type FileService struct {
	logger          *slog.Logger
	storageRegistry registry.StorageRegistry
	storageClient   storage.Client
	chunkManager    *chunk.AccurateManager
}

type FileMetadata struct {
	ID     string
	Size   int64
	Chunks []ChunkInfo
}

type ChunkInfo struct {
	ID     string
	Server string
	Size   int64
}

func NewFileService(
	logger *slog.Logger,
	registry registry.StorageRegistry,
	client storage.Client,
) *FileService {
	return &FileService{
		logger:          logger,
		storageRegistry: registry,
		storageClient:   client,
		chunkManager:    chunk.NewAccurateManager(6),
	}
}

func (s *FileService) Upload(ctx context.Context, file io.Reader, size int64) (string, error) {
	fileID := generateFileID()
	servers, err := s.storageRegistry.SelectServers(6)
	if err != nil {
		return "", err
	}

	s.logger.Debug("Splitting file to chunks")
	chunks, err := s.chunkManager.Split(file, size, fileID)
	if err != nil {
		return "", err
	}
	s.logger.Debug("File splitted to chunks", "len_chunks", len(chunks))

	if err := s.uploadChunksStreaming(ctx, chunks, servers); err != nil {
		s.cleanupChunks(ctx, chunks, servers)
		return "", err
	}

	s.logger.Debug("Chunks uploaded to servers")

	metadata := &FileMetadata{
		ID:     fileID,
		Size:   size,
		Chunks: make([]ChunkInfo, len(chunks)),
	}

	for i, chunk := range chunks {
		metadata.Chunks[i] = ChunkInfo{
			ID:     chunk.ID,
			Server: servers[i%len(servers)].ID,
			Size:   chunk.Size,
		}
	}

	if err := s.saveMetadata(ctx, metadata); err != nil {
		s.cleanupChunks(ctx, chunks, servers)
		return "", err
	}

	return fileID, nil
}

func (s *FileService) Download(ctx context.Context, fileID string) (io.ReadCloser, int64, error) {
	metadata, err := s.loadMetadata(ctx, fileID)
	if err != nil {
		return nil, 0, fmt.Errorf("file not found: %w", err)
	}

	reader, writer := io.Pipe()

	go func() {
		defer writer.Close()

		chunkReaders := make([]io.ReadCloser, len(metadata.Chunks))
		errs := make(chan error, len(metadata.Chunks))

		for i, chunkInfo := range metadata.Chunks {
			go func(i int, chunk ChunkInfo) {
				server := &registry.StorageServer{ID: chunk.Server}
				chunkReader, err := s.storageClient.RetrieveChunk(ctx, server, chunk.ID)
				if err != nil {
					errs <- fmt.Errorf("chunk %s: %w", chunk.ID, err)
					return
				}
				chunkReaders[i] = chunkReader
				errs <- nil
			}(i, chunkInfo)
		}

		for range metadata.Chunks {
			if err := <-errs; err != nil {
				writer.CloseWithError(err)
				return
			}
		}

		// if err := s.chunkManager.Join(chunkReaders, writer); err != nil {
		// 	writer.CloseWithError(err)
		// }

		// for _, cr := range chunkReaders {
		// 	cr.Close()
		// }
	}()

	return reader, metadata.Size, nil
}

func (s *FileService) uploadChunksStreaming(ctx context.Context, chunks []chunk.Chunk, servers []*registry.StorageServer) error {
	var wg sync.WaitGroup
	errCh := make(chan error, len(chunks))

	for i, ch := range chunks {
		s.logger.Debug("Start uploading chunk", "number", i)
		wg.Add(1)
		go func(i int, chunk chunk.Chunk) {
			defer wg.Done()
			//defer chunk.Data.(io.ReadCloser).Close() TBD

			server := servers[i%len(servers)]
			fmt.Println("BEFORE STORE")
			err := s.storageClient.StoreChunk(ctx, server, chunk.ID, chunk.Data)
			fmt.Println("GOT ERROR", i)
			if err != nil {
				errCh <- fmt.Errorf("chunk %s: %w", chunk.ID, err)
				s.logger.Debug("Stored error about uploading chunk", "number", i)
				fmt.Println("haha1")
			}
			s.logger.Debug("No error when uploading chunk", "number", i)
			fmt.Println("haha2")
		}(i, ch)
	}

	wg.Wait()
	s.logger.Debug("Closing channel")
	close(errCh)

	for err := range errCh {
		s.logger.Debug("Waiting for errors")
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *FileService) cleanupChunks(ctx context.Context, chunks []chunk.Chunk, servers []*registry.StorageServer) {
	for i, ch := range chunks {
		server := servers[i%len(servers)]
		go s.storageClient.DeleteChunk(context.Background(), server, ch.ID)
	}
}

func (s *FileService) saveMetadata(ctx context.Context, metadata *FileMetadata) error {
	// temporary in-memory implementation
	return nil
}

func (s *FileService) loadMetadata(ctx context.Context, fileID string) (*FileMetadata, error) {
	// temporary implementation
	return nil, fmt.Errorf("metadata not found")
}

func generateFileID() string {
	hash := sha256.Sum256([]byte(time.Now().String() + fmt.Sprintf("%d", time.Now().UnixNano())))
	return hex.EncodeToString(hash[:16])
}
