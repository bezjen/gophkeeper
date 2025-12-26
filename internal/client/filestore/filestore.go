// Package filestore provides local file-based storage for GophKeeper client data.
// It implements caching and JSON serialization for efficient data access.
//
//go:generate mockery --name=StoreInterface --output=../mocks --outpkg=mocks --case=underscore
package filestore

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	pb "github.com/bezjen/gophkeeper/api/gophkeeper/v1"
)

// StoreInterface defines the interface for local data storage operations.
type StoreInterface interface {
	// Save stores a data record to local storage.
	Save(record *pb.DataRecord) error

	// Get retrieves a data record by ID from local storage.
	Get(id string) (*pb.DataRecord, error)

	// List returns all data records, optionally filtered by type.
	List(filterType pb.DataType) ([]*pb.DataRecord, error)

	// Sync updates local storage with multiple records (for synchronization).
	Sync(records []*pb.DataRecord) error

	// GetChangedSince returns records modified after the specified timestamp.
	GetChangedSince(timestamp int64) ([]*pb.DataRecord, error)
}

// FileStore implements StoreInterface with file-based storage and in-memory caching.
type FileStore struct {
	baseDir string
	cache   map[string]*pb.DataRecord
	mu      sync.RWMutex
}

// NewFileStore creates a new FileStore instance with the specified base directory.
func NewFileStore(baseDir string) (*FileStore, error) {
	if err := os.MkdirAll(baseDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	store := &FileStore{
		baseDir: baseDir,
		cache:   make(map[string]*pb.DataRecord),
	}

	if err := store.loadAll(); err != nil {
		return nil, fmt.Errorf("failed to load storage: %w", err)
	}

	return store, nil
}

// Save implements StoreInterface.Save with caching and file persistence.
func (s *FileStore) Save(record *pb.DataRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.cache[record.Id] = record

	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal record: %w", err)
	}

	filename := filepath.Join(s.baseDir, record.Id+".json")
	return os.WriteFile(filename, data, 0600)
}

// Get implements StoreInterface.Get with cache-first lookup.
func (s *FileStore) Get(id string) (*pb.DataRecord, error) {
	s.mu.RLock()
	record, exists := s.cache[id]
	s.mu.RUnlock()

	if exists {
		return record, nil
	}

	filename := filepath.Join(s.baseDir, id+".json")
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("record not found: %w", err)
	}

	var loadedRecord pb.DataRecord
	if err := json.Unmarshal(data, &loadedRecord); err != nil {
		return nil, fmt.Errorf("failed to unmarshal record: %w", err)
	}

	s.mu.Lock()
	s.cache[id] = &loadedRecord
	s.mu.Unlock()

	return &loadedRecord, nil
}

// List implements StoreInterface.List with optional type filtering.
func (s *FileStore) List(filterType pb.DataType) ([]*pb.DataRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*pb.DataRecord
	for _, record := range s.cache {
		if filterType == pb.DataType(-1) || record.Type == filterType {
			result = append(result, record)
		}
	}

	return result, nil
}

// Sync implements StoreInterface.Sync by updating local storage with newer records.
func (s *FileStore) Sync(records []*pb.DataRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, record := range records {
		existing, exists := s.cache[record.Id]
		if !exists || record.UpdatedAt > existing.UpdatedAt {
			s.cache[record.Id] = record

			data, err := json.MarshalIndent(record, "", "  ")
			if err != nil {
				continue
			}

			filename := filepath.Join(s.baseDir, record.Id+".json")
			os.WriteFile(filename, data, 0600)
		}
	}

	return nil
}

// GetChangedSince implements StoreInterface.GetChangedSince for synchronization.
func (s *FileStore) GetChangedSince(timestamp int64) ([]*pb.DataRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*pb.DataRecord
	for _, record := range s.cache {
		if record.UpdatedAt > timestamp {
			result = append(result, record)
		}
	}

	return result, nil
}

// loadAll loads all JSON files from the base directory into the cache.
func (s *FileStore) loadAll() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		return nil
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		filename := filepath.Join(s.baseDir, entry.Name())
		data, err := os.ReadFile(filename)
		if err != nil {
			continue
		}

		var record pb.DataRecord
		if err := json.Unmarshal(data, &record); err != nil {
			continue
		}

		id := entry.Name()[:len(entry.Name())-5]
		s.cache[id] = &record
	}

	return nil
}
