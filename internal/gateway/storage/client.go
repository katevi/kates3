package storage

import (
	"context"
	"fmt"
	"io"
	"sync"

	"kates3/internal/gateway/storage/registry"
)

type Client interface {
	StoreChunk(ctx context.Context, server *registry.StorageServer, chunkID string, data io.Reader) error
	RetrieveChunk(ctx context.Context, server *registry.StorageServer, chunkID string) (io.ReadCloser, error)
	DeleteChunk(ctx context.Context, server *registry.StorageServer, chunkID string) error
	Close() error
}

type MockClient struct {
	storage map[string][]byte
	mu      sync.RWMutex
}

func NewMockClient() *MockClient {
	return &MockClient{
		storage: make(map[string][]byte),
	}
}

func (m *MockClient) StoreChunk(ctx context.Context, server *registry.StorageServer, chunkID string, data io.Reader) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	chunkData, err := io.ReadAll(data)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("%s-%s", server.ID, chunkID)
	m.storage[key] = chunkData

	fmt.Printf("Stored chunk %s on server %s (%d bytes)\n", chunkID, server.ID, len(chunkData))
	return nil
}

func (m *MockClient) RetrieveChunk(ctx context.Context, server *registry.StorageServer, chunkID string) (io.ReadCloser, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := fmt.Sprintf("%s-%s", server.ID, chunkID)
	chunkData, exists := m.storage[key]
	if !exists {
		return nil, fmt.Errorf("chunk not found")
	}

	return io.NopCloser(io.NewSectionReader(readerForData(chunkData), 0, int64(len(chunkData)))), nil
}

func (m *MockClient) DeleteChunk(ctx context.Context, server *registry.StorageServer, chunkID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%s-%s", server.ID, chunkID)
	delete(m.storage, key)
	fmt.Printf("Deleted chunk %s from server %s\n", chunkID, server.ID)
	return nil
}

func (m *MockClient) Close() error {
	return nil
}

type readerForData []byte

func (r readerForData) ReadAt(p []byte, off int64) (n int, err error) {
	copy(p, r)
	return len(r), io.EOF
}

func (r readerForData) Seek(offset int64, whence int) (int64, error) {
	return 0, nil
}
