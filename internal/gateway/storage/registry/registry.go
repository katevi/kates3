package registry

import (
	"fmt"
	"sync"
)

type StorageServer struct {
	ID      string
	Address string
	Weight  int
}

type StorageRegistry interface {
	SelectServers(count int) ([]*StorageServer, error)
	RegisterServer(server *StorageServer)
}

type SimpleRegistry struct {
	servers []*StorageServer
	mu      sync.RWMutex
	current int
}

func NewSimpleRegistry() *SimpleRegistry {
	return &SimpleRegistry{
		servers: make([]*StorageServer, 0),
	}
}

func (s *SimpleRegistry) RegisterServer(server *StorageServer) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.servers = append(s.servers, server)
	fmt.Printf("Registered storage server: %s at %s\n", server.ID, server.Address)
}

func (s *SimpleRegistry) SelectServers(count int) ([]*StorageServer, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.servers) < count {
		return nil, fmt.Errorf("not enough servers. Have %d, need %d", len(s.servers), count)
	}

	result := make([]*StorageServer, count)
	for i := 0; i < count; i++ {
		result[i] = s.servers[(s.current+i)%len(s.servers)]
	}
	s.current = (s.current + count) % len(s.servers)

	return result, nil
}

func (s *SimpleRegistry) GetAllServers() []*StorageServer {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]*StorageServer{}, s.servers...)
}
