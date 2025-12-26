package filestore

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	pb "github.com/bezjen/gophkeeper/api/gophkeeper/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFileStore(t *testing.T) {
	t.Run("successful creation", func(t *testing.T) {
		tempDir := t.TempDir()

		store, err := NewFileStore(tempDir)
		require.NoError(t, err)
		require.NotNil(t, store)
		assert.Equal(t, tempDir, store.baseDir)
		assert.NotNil(t, store.cache)
	})

	t.Run("creates directory if not exists", func(t *testing.T) {
		tempDir := filepath.Join(t.TempDir(), "subdir", "nested")

		store, err := NewFileStore(tempDir)
		require.NoError(t, err)
		require.NotNil(t, store)

		_, err = os.Stat(tempDir)
		assert.NoError(t, err)
	})

	t.Run("loads existing files", func(t *testing.T) {
		tempDir := t.TempDir()

		// Create some test JSON files
		records := []*pb.DataRecord{
			{
				Id:            "test-id-1",
				Type:          pb.DataType_LOGIN_PASSWORD,
				Name:          "Test Credentials 1",
				EncryptedData: []byte("encrypted data 1"),
				Metadata:      map[string]string{"key1": "value1", "note": "test note"},
				Version:       1,
				UpdatedAt:     1000,
				Deleted:       false,
			},
			{
				Id:            "test-id-2",
				Type:          pb.DataType_TEXT_DATA,
				Name:          "Test Text 2",
				EncryptedData: []byte("encrypted data 2"),
				Metadata:      map[string]string{"category": "personal"},
				Version:       1,
				UpdatedAt:     2000,
				Deleted:       false,
			},
		}

		for _, record := range records {
			data, err := json.MarshalIndent(record, "", "  ")
			require.NoError(t, err)

			filename := filepath.Join(tempDir, record.Id+".json")
			err = os.WriteFile(filename, data, 0600)
			require.NoError(t, err)
		}

		store, err := NewFileStore(tempDir)
		require.NoError(t, err)
		require.NotNil(t, store)

		assert.Equal(t, 2, len(store.cache))

		// Check first record
		record1 := store.cache["test-id-1"]
		require.NotNil(t, record1)
		assert.Equal(t, "test-id-1", record1.Id)
		assert.Equal(t, pb.DataType_LOGIN_PASSWORD, record1.Type)
		assert.Equal(t, "Test Credentials 1", record1.Name)
		assert.Equal(t, []byte("encrypted data 1"), record1.EncryptedData)
		assert.Equal(t, map[string]string{"key1": "value1", "note": "test note"}, record1.Metadata)
		assert.Equal(t, int64(1), record1.Version)
		assert.Equal(t, int64(1000), record1.UpdatedAt)
		assert.False(t, record1.Deleted)

		// Check second record
		record2 := store.cache["test-id-2"]
		require.NotNil(t, record2)
		assert.Equal(t, "test-id-2", record2.Id)
		assert.Equal(t, pb.DataType_TEXT_DATA, record2.Type)
		assert.Equal(t, "Test Text 2", record2.Name)
		assert.Equal(t, []byte("encrypted data 2"), record2.EncryptedData)
		assert.Equal(t, map[string]string{"category": "personal"}, record2.Metadata)
		assert.Equal(t, int64(1), record2.Version)
		assert.Equal(t, int64(2000), record2.UpdatedAt)
		assert.False(t, record2.Deleted)
	})

	t.Run("ignores non-json files", func(t *testing.T) {
		tempDir := t.TempDir()

		// Create a JSON file
		record := &pb.DataRecord{
			Id:            "test-id",
			Type:          pb.DataType_LOGIN_PASSWORD,
			Name:          "Test",
			EncryptedData: []byte("encrypted data"),
			Metadata:      map[string]string{"key": "value"},
			Version:       1,
			UpdatedAt:     1000,
			Deleted:       false,
		}

		data, err := json.MarshalIndent(record, "", "  ")
		require.NoError(t, err)

		filename := filepath.Join(tempDir, "test-id.json")
		err = os.WriteFile(filename, data, 0600)
		require.NoError(t, err)

		// Create a non-JSON file
		err = os.WriteFile(filepath.Join(tempDir, "test.txt"), []byte("not json"), 0600)
		require.NoError(t, err)

		// Create a directory
		err = os.Mkdir(filepath.Join(tempDir, "subdir"), 0700)
		require.NoError(t, err)

		store, err := NewFileStore(tempDir)
		require.NoError(t, err)
		require.NotNil(t, store)

		assert.Equal(t, 1, len(store.cache))
		assert.NotNil(t, store.cache["test-id"])
	})
}

func TestSave(t *testing.T) {
	tempDir := t.TempDir()
	store, err := NewFileStore(tempDir)
	require.NoError(t, err)

	record := &pb.DataRecord{
		Id:            "test-id",
		Type:          pb.DataType_LOGIN_PASSWORD,
		Name:          "Test Credentials",
		EncryptedData: []byte("encrypted test data"),
		Metadata:      map[string]string{"username": "testuser", "url": "example.com"},
		Version:       1,
		UpdatedAt:     time.Now().Unix(),
		Deleted:       false,
	}

	t.Run("save new record", func(t *testing.T) {
		err := store.Save(record)
		require.NoError(t, err)

		// Check cache
		cached, exists := store.cache["test-id"]
		require.True(t, exists)
		assert.Equal(t, record, cached)

		// Check file
		filename := filepath.Join(tempDir, "test-id.json")
		data, err := os.ReadFile(filename)
		require.NoError(t, err)

		var loadedRecord pb.DataRecord
		err = json.Unmarshal(data, &loadedRecord)
		require.NoError(t, err)

		assert.Equal(t, record.Id, loadedRecord.Id)
		assert.Equal(t, record.Type, loadedRecord.Type)
		assert.Equal(t, record.Name, loadedRecord.Name)
		assert.Equal(t, record.EncryptedData, loadedRecord.EncryptedData)
		assert.Equal(t, record.Metadata, loadedRecord.Metadata)
		assert.Equal(t, record.Version, loadedRecord.Version)
		assert.Equal(t, record.UpdatedAt, loadedRecord.UpdatedAt)
		assert.Equal(t, record.Deleted, loadedRecord.Deleted)
	})

	t.Run("update existing record", func(t *testing.T) {
		updatedRecord := &pb.DataRecord{
			Id:            "test-id",
			Type:          pb.DataType_LOGIN_PASSWORD,
			Name:          "Updated Credentials",
			EncryptedData: []byte("updated encrypted data"),
			Metadata:      map[string]string{"username": "newuser", "url": "newexample.com"},
			Version:       2,
			UpdatedAt:     time.Now().Unix() + 1000,
			Deleted:       false,
		}

		err := store.Save(updatedRecord)
		require.NoError(t, err)

		cached, exists := store.cache["test-id"]
		require.True(t, exists)
		assert.Equal(t, updatedRecord, cached)
	})

	t.Run("save record with empty metadata", func(t *testing.T) {
		emptyMetaRecord := &pb.DataRecord{
			Id:            "empty-meta-id",
			Type:          pb.DataType_TEXT_DATA,
			Name:          "Empty Metadata",
			EncryptedData: []byte("data"),
			Metadata:      map[string]string{},
			Version:       1,
			UpdatedAt:     1000,
			Deleted:       false,
		}

		err := store.Save(emptyMetaRecord)
		require.NoError(t, err)

		retrieved, err := store.Get("empty-meta-id")
		require.NoError(t, err)
		assert.Equal(t, map[string]string{}, retrieved.Metadata)
	})

	t.Run("save record with nil metadata", func(t *testing.T) {
		nilMetaRecord := &pb.DataRecord{
			Id:            "nil-meta-id",
			Type:          pb.DataType_BINARY_DATA,
			Name:          "Nil Metadata",
			EncryptedData: []byte("binary data"),
			Metadata:      nil,
			Version:       1,
			UpdatedAt:     2000,
			Deleted:       false,
		}

		err := store.Save(nilMetaRecord)
		require.NoError(t, err)

		retrieved, err := store.Get("nil-meta-id")
		require.NoError(t, err)
		assert.Nil(t, retrieved.Metadata)
	})

	t.Run("save deleted record", func(t *testing.T) {
		deletedRecord := &pb.DataRecord{
			Id:            "deleted-id",
			Type:          pb.DataType_BANK_CARD,
			Name:          "Deleted Card",
			EncryptedData: []byte("card data"),
			Metadata:      map[string]string{"bank": "test"},
			Version:       3,
			UpdatedAt:     3000,
			Deleted:       true,
		}

		err := store.Save(deletedRecord)
		require.NoError(t, err)

		retrieved, err := store.Get("deleted-id")
		require.NoError(t, err)
		assert.True(t, retrieved.Deleted)
	})
}

func TestGet(t *testing.T) {
	tempDir := t.TempDir()

	// Create a record on disk
	record := &pb.DataRecord{
		Id:            "test-id",
		Type:          pb.DataType_LOGIN_PASSWORD,
		Name:          "Test Credentials",
		EncryptedData: []byte("encrypted data"),
		Metadata:      map[string]string{"user": "test", "url": "test.com"},
		Version:       2,
		UpdatedAt:     1000,
		Deleted:       false,
	}

	data, err := json.MarshalIndent(record, "", "  ")
	require.NoError(t, err)

	filename := filepath.Join(tempDir, "test-id.json")
	err = os.WriteFile(filename, data, 0600)
	require.NoError(t, err)

	store, err := NewFileStore(tempDir)
	require.NoError(t, err)

	t.Run("get from cache after load", func(t *testing.T) {
		// Should be in cache after NewFileStore
		retrieved, err := store.Get("test-id")
		require.NoError(t, err)
		assert.Equal(t, record, retrieved)
	})

	t.Run("get from disk when not in cache", func(t *testing.T) {
		// Create another record only on disk
		anotherRecord := &pb.DataRecord{
			Id:            "another-id",
			Type:          pb.DataType_TEXT_DATA,
			Name:          "Another Text",
			EncryptedData: []byte("another encrypted data"),
			Metadata:      map[string]string{"category": "work"},
			Version:       1,
			UpdatedAt:     2000,
			Deleted:       false,
		}

		data, err := json.MarshalIndent(anotherRecord, "", "  ")
		require.NoError(t, err)

		filename := filepath.Join(tempDir, "another-id.json")
		err = os.WriteFile(filename, data, 0600)
		require.NoError(t, err)

		// Should load from disk and add to cache
		retrieved, err := store.Get("another-id")
		require.NoError(t, err)
		assert.Equal(t, anotherRecord, retrieved)

		// Now should be in cache
		assert.Contains(t, store.cache, "another-id")
	})

	t.Run("get non-existent record", func(t *testing.T) {
		_, err := store.Get("non-existent")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "record not found")
	})

	t.Run("get with corrupted file", func(t *testing.T) {
		filename := filepath.Join(tempDir, "corrupted.json")
		err := os.WriteFile(filename, []byte("not json"), 0600)
		require.NoError(t, err)

		_, err = store.Get("corrupted")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to unmarshal")
	})
}

func TestList(t *testing.T) {
	tempDir := t.TempDir()
	store, err := NewFileStore(tempDir)
	require.NoError(t, err)

	records := []*pb.DataRecord{
		{
			Id:            "id1",
			Type:          pb.DataType_LOGIN_PASSWORD,
			Name:          "Credentials 1",
			EncryptedData: []byte("data1"),
			Metadata:      map[string]string{"url": "site1.com"},
			Version:       1,
			UpdatedAt:     1000,
			Deleted:       false,
		},
		{
			Id:            "id2",
			Type:          pb.DataType_TEXT_DATA,
			Name:          "Text Note 1",
			EncryptedData: []byte("data2"),
			Metadata:      map[string]string{"category": "personal"},
			Version:       1,
			UpdatedAt:     2000,
			Deleted:       false,
		},
		{
			Id:            "id3",
			Type:          pb.DataType_LOGIN_PASSWORD,
			Name:          "Credentials 2",
			EncryptedData: []byte("data3"),
			Metadata:      map[string]string{"url": "site2.com"},
			Version:       2,
			UpdatedAt:     3000,
			Deleted:       false,
		},
		{
			Id:            "id4",
			Type:          pb.DataType_BINARY_DATA,
			Name:          "Binary File",
			EncryptedData: []byte("data4"),
			Metadata:      map[string]string{"format": "jpg"},
			Version:       1,
			UpdatedAt:     4000,
			Deleted:       false,
		},
		{
			Id:            "id5",
			Type:          pb.DataType_BANK_CARD,
			Name:          "Bank Card",
			EncryptedData: []byte("card data"),
			Metadata:      map[string]string{"bank": "test bank"},
			Version:       1,
			UpdatedAt:     5000,
			Deleted:       true, // Deleted record
		},
	}

	for _, record := range records {
		err := store.Save(record)
		require.NoError(t, err)
	}

	t.Run("list all records (including deleted)", func(t *testing.T) {
		result, err := store.List(pb.DataType(-1))
		require.NoError(t, err)

		assert.Equal(t, 5, len(result))
		ids := make(map[string]bool)
		for _, r := range result {
			ids[r.Id] = true
		}
		for i := 1; i <= 5; i++ {
			assert.True(t, ids[fmt.Sprintf("id%d", i)])
		}
	})

	t.Run("list with type filter", func(t *testing.T) {
		result, err := store.List(pb.DataType_LOGIN_PASSWORD)
		require.NoError(t, err)

		assert.Equal(t, 2, len(result))
		for _, r := range result {
			assert.Equal(t, pb.DataType_LOGIN_PASSWORD, r.Type)
		}
	})

	t.Run("list with another type filter", func(t *testing.T) {
		result, err := store.List(pb.DataType_TEXT_DATA)
		require.NoError(t, err)

		assert.Equal(t, 1, len(result))
		assert.Equal(t, "id2", result[0].Id)
	})

	t.Run("list with non-existent type filter", func(t *testing.T) {
		result, err := store.List(pb.DataType(999))
		require.NoError(t, err)

		assert.Equal(t, 0, len(result))
	})

	t.Run("list empty store", func(t *testing.T) {
		emptyStore, err := NewFileStore(t.TempDir())
		require.NoError(t, err)

		result, err := emptyStore.List(pb.DataType(-1))
		require.NoError(t, err)

		assert.Empty(t, result)
	})
}

func TestSync(t *testing.T) {
	tempDir := t.TempDir()
	store, err := NewFileStore(tempDir)
	require.NoError(t, err)

	// Initial records
	initialRecords := []*pb.DataRecord{
		{
			Id:            "id1",
			Type:          pb.DataType_LOGIN_PASSWORD,
			Name:          "Initial Credentials",
			EncryptedData: []byte("initial encrypted data 1"),
			Metadata:      map[string]string{"url": "initial.com"},
			Version:       1,
			UpdatedAt:     1000,
			Deleted:       false,
		},
		{
			Id:            "id2",
			Type:          pb.DataType_TEXT_DATA,
			Name:          "Initial Text",
			EncryptedData: []byte("initial encrypted data 2"),
			Metadata:      map[string]string{"category": "initial"},
			Version:       1,
			UpdatedAt:     2000,
			Deleted:       false,
		},
	}

	for _, record := range initialRecords {
		err := store.Save(record)
		require.NoError(t, err)
	}

	t.Run("sync with newer records", func(t *testing.T) {
		syncRecords := []*pb.DataRecord{
			{
				Id:            "id1", // Update existing
				Type:          pb.DataType_LOGIN_PASSWORD,
				Name:          "Updated Credentials",
				EncryptedData: []byte("updated encrypted data 1"),
				Metadata:      map[string]string{"url": "updated.com", "user": "newuser"},
				Version:       2,
				UpdatedAt:     3000, // Newer
				Deleted:       false,
			},
			{
				Id:            "id3", // New record
				Type:          pb.DataType_BINARY_DATA,
				Name:          "New Binary",
				EncryptedData: []byte("new encrypted data 3"),
				Metadata:      map[string]string{"format": "pdf"},
				Version:       1,
				UpdatedAt:     4000,
				Deleted:       false,
			},
		}

		err := store.Sync(syncRecords)
		require.NoError(t, err)

		// Check id1 was updated
		record1, err := store.Get("id1")
		require.NoError(t, err)
		assert.Equal(t, "Updated Credentials", record1.Name)
		assert.Equal(t, []byte("updated encrypted data 1"), record1.EncryptedData)
		assert.Equal(t, int64(3000), record1.UpdatedAt)
		assert.Equal(t, int64(2), record1.Version)

		// Check id2 still exists with old data
		record2, err := store.Get("id2")
		require.NoError(t, err)
		assert.Equal(t, "Initial Text", record2.Name)
		assert.Equal(t, []byte("initial encrypted data 2"), record2.EncryptedData)
		assert.Equal(t, int64(2000), record2.UpdatedAt)

		// Check id3 was added
		record3, err := store.Get("id3")
		require.NoError(t, err)
		assert.Equal(t, "New Binary", record3.Name)
		assert.Equal(t, []byte("new encrypted data 3"), record3.EncryptedData)
		assert.Equal(t, int64(4000), record3.UpdatedAt)
	})

	t.Run("sync with older records should not update", func(t *testing.T) {
		oldRecord := &pb.DataRecord{
			Id:            "id1",
			Type:          pb.DataType_LOGIN_PASSWORD,
			Name:          "Older Credentials",
			EncryptedData: []byte("older encrypted data"),
			Metadata:      map[string]string{"url": "older.com"},
			Version:       1,
			UpdatedAt:     500, // Older than current 3000
			Deleted:       false,
		}

		err := store.Sync([]*pb.DataRecord{oldRecord})
		require.NoError(t, err)

		// Should still have the newer data
		record, err := store.Get("id1")
		require.NoError(t, err)
		assert.Equal(t, "Updated Credentials", record.Name)
		assert.Equal(t, int64(3000), record.UpdatedAt)
	})

	t.Run("sync with deleted record", func(t *testing.T) {
		deletedRecord := &pb.DataRecord{
			Id:            "id2",
			Type:          pb.DataType_TEXT_DATA,
			Name:          "Deleted Text",
			EncryptedData: []byte("deleted data"),
			Metadata:      map[string]string{"category": "deleted"},
			Version:       2,
			UpdatedAt:     5000, // Newer
			Deleted:       true,
		}

		err := store.Sync([]*pb.DataRecord{deletedRecord})
		require.NoError(t, err)

		// Deleted record should be saved
		record, err := store.Get("id2")
		require.NoError(t, err)
		assert.True(t, record.Deleted)
		assert.Equal(t, int64(5000), record.UpdatedAt)
	})

	t.Run("sync empty list", func(t *testing.T) {
		err := store.Sync([]*pb.DataRecord{})
		require.NoError(t, err)

		// Should not affect existing records
		assert.Equal(t, 3, len(store.cache))
	})
}

func TestGetChangedSince(t *testing.T) {
	tempDir := t.TempDir()
	store, err := NewFileStore(tempDir)
	require.NoError(t, err)

	records := []*pb.DataRecord{
		{
			Id:            "id1",
			Type:          pb.DataType_LOGIN_PASSWORD,
			Name:          "Record 1",
			EncryptedData: []byte("data1"),
			Metadata:      map[string]string{"key": "value1"},
			Version:       1,
			UpdatedAt:     1000,
			Deleted:       false,
		},
		{
			Id:            "id2",
			Type:          pb.DataType_TEXT_DATA,
			Name:          "Record 2",
			EncryptedData: []byte("data2"),
			Metadata:      map[string]string{"key": "value2"},
			Version:       1,
			UpdatedAt:     2000,
			Deleted:       false,
		},
		{
			Id:            "id3",
			Type:          pb.DataType_BINARY_DATA,
			Name:          "Record 3",
			EncryptedData: []byte("data3"),
			Metadata:      map[string]string{"key": "value3"},
			Version:       1,
			UpdatedAt:     3000,
			Deleted:       false,
		},
		{
			Id:            "id4",
			Type:          pb.DataType_LOGIN_PASSWORD,
			Name:          "Record 4",
			EncryptedData: []byte("data4"),
			Metadata:      map[string]string{"key": "value4"},
			Version:       1,
			UpdatedAt:     4000,
			Deleted:       true, // Deleted but still has timestamp
		},
	}

	for _, record := range records {
		err := store.Save(record)
		require.NoError(t, err)
	}

	testCases := []struct {
		name      string
		timestamp int64
		expected  []string
	}{
		{
			name:      "timestamp before all records",
			timestamp: 0,
			expected:  []string{"id1", "id2", "id3", "id4"},
		},
		{
			name:      "timestamp in the middle",
			timestamp: 2000,
			expected:  []string{"id3", "id4"},
		},
		{
			name:      "timestamp after all records",
			timestamp: 5000,
			expected:  []string{},
		},
		{
			name:      "exact timestamp match",
			timestamp: 3000,
			expected:  []string{"id4"},
		},
		{
			name:      "timestamp between records",
			timestamp: 2500,
			expected:  []string{"id3", "id4"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := store.GetChangedSince(tc.timestamp)
			require.NoError(t, err)

			assert.Equal(t, len(tc.expected), len(result))

			resultIds := make(map[string]bool)
			for _, r := range result {
				resultIds[r.Id] = true
			}

			for _, expectedId := range tc.expected {
				assert.True(t, resultIds[expectedId], "Expected record %s not found", expectedId)
			}
		})
	}
}

func TestConcurrentAccess(t *testing.T) {
	tempDir := t.TempDir()
	store, err := NewFileStore(tempDir)
	require.NoError(t, err)

	done := make(chan bool)
	errors := make(chan error, 20)

	for i := 0; i < 10; i++ {
		go func(idx int) {
			record := &pb.DataRecord{
				Id:            string(rune('a' + idx)),
				Type:          pb.DataType_LOGIN_PASSWORD,
				Name:          fmt.Sprintf("Record %d", idx),
				EncryptedData: []byte("encrypted data"),
				Metadata:      map[string]string{"index": fmt.Sprintf("%d", idx)},
				Version:       1,
				UpdatedAt:     int64(idx * 1000),
				Deleted:       false,
			}

			if err := store.Save(record); err != nil {
				errors <- err
			}

			if _, err := store.Get(string(rune('a' + idx))); err != nil {
				errors <- err
			}

			if _, err := store.List(pb.DataType(-1)); err != nil {
				errors <- err
			}

			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	close(errors)
	for err := range errors {
		assert.NoError(t, err)
	}

	result, err := store.List(pb.DataType(-1))
	require.NoError(t, err)
	assert.Equal(t, 10, len(result))
}

func BenchmarkSave(b *testing.B) {
	tempDir := b.TempDir()
	store, err := NewFileStore(tempDir)
	if err != nil {
		b.Fatal(err)
	}

	record := &pb.DataRecord{
		Id:            "test-id",
		Type:          pb.DataType_LOGIN_PASSWORD,
		Name:          "Benchmark Record",
		EncryptedData: make([]byte, 1024),
		Metadata:      map[string]string{"benchmark": "true"},
		Version:       1,
		UpdatedAt:     1000,
		Deleted:       false,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		record.Id = string(rune('a' + i))
		if err := store.Save(record); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGet(b *testing.B) {
	tempDir := b.TempDir()
	store, err := NewFileStore(tempDir)
	if err != nil {
		b.Fatal(err)
	}

	// Save 1000 records
	for i := 0; i < 1000; i++ {
		record := &pb.DataRecord{
			Id:            string(rune('a' + i)),
			Type:          pb.DataType_LOGIN_PASSWORD,
			Name:          fmt.Sprintf("Record %d", i),
			EncryptedData: []byte("encrypted data"),
			Metadata:      map[string]string{"index": fmt.Sprintf("%d", i)},
			Version:       1,
			UpdatedAt:     int64(i * 1000),
			Deleted:       false,
		}

		if err := store.Save(record); err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		id := string(rune('a' + (i % 1000)))
		if _, err := store.Get(id); err != nil {
			b.Fatal(err)
		}
	}
}
