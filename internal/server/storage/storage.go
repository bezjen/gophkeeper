// Package storage provides data storage interfaces and implementations for the GophKeeper server.
// It includes SQL-based storage for user data and data records with support for versioning and synchronization.
//
//go:generate mockery --name=UserStorage --output=../mocks --outpkg=mocks --case=underscore
//go:generate mockery --name=DataStorage --output=../mocks --outpkg=mocks --case=underscore
//go:generate mockery --name=Storage --output=../mocks --outpkg=mocks --case=underscore
package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/bezjen/gophkeeper/internal/server/errors"
	"github.com/bezjen/gophkeeper/internal/server/models"
	"log"
	"time"

	pb "github.com/bezjen/gophkeeper/api/gophkeeper/v1"
)

// Storage is the unified interface that combines both UserStorage and DataStorage capabilities.
type Storage interface {
	UserStorage
	DataStorage
}

// UserStorage defines operations for user data management.
type UserStorage interface {
	// CreateUser creates a new user record in the database.
	CreateUser(ctx context.Context, user *models.User) error

	// GetUserByUsername retrieves a user by their username.
	GetUserByUsername(ctx context.Context, username string) (*models.User, error)

	// GetUserByUsernameOrEmail retrieves a user by either username or email address.
	GetUserByUsernameOrEmail(ctx context.Context, username, email string) (*models.User, error)
}

// DataStorage defines operations for secure data record management.
type DataStorage interface {
	// StoreData stores or updates a data record for a specific user.
	StoreData(ctx context.Context, userID string, data *pb.DataRecord) error

	// RetrieveData retrieves a specific data record for a user.
	RetrieveData(ctx context.Context, userID, dataID string) (*pb.DataRecord, error)

	// DeleteData marks a data record as deleted (soft delete).
	DeleteData(ctx context.Context, userID, dataID string) error

	// ListData returns all data records for a user, optionally filtered by type.
	ListData(ctx context.Context, userID string, filterType pb.DataType) ([]*pb.DataRecord, error)

	// ProcessSync handles synchronization between client and server data.
	ProcessSync(ctx context.Context, userID string, localChanges []*pb.DataRecord, lastSync int64) ([]*pb.DataRecord, error)

	// Ping checks the database connection health.
	Ping(ctx context.Context) error
}

// SQLStorage implements the Storage interface using SQL database.
type SQLStorage struct {
	db *sql.DB
}

// NewStorage creates a new SQLStorage instance with the provided database connection.
func NewStorage(db *sql.DB) Storage {
	return &SQLStorage{db: db}
}

// CreateUser implements the UserStorage interface for creating new users.
func (s *SQLStorage) CreateUser(ctx context.Context, user *models.User) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	query := `
		INSERT INTO t_user (id, username, password, email, created_at) 
		VALUES ($1, $2, $3, $4, $5)
	`

	_, err := s.db.ExecContext(ctx, query,
		user.ID, user.Username, user.Password, user.Email, user.CreatedAt)
	return err
}

// GetUserByUsername implements the UserStorage interface for retrieving users by username.
func (s *SQLStorage) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	query := `SELECT id, username, password, email, created_at FROM t_user WHERE username = $1`

	var user models.User
	err := s.db.QueryRowContext(ctx, query, username).Scan(
		&user.ID, &user.Username, &user.Password, &user.Email, &user.CreatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}

// GetUserByUsernameOrEmail implements the UserStorage interface for retrieving users by username or email.
func (s *SQLStorage) GetUserByUsernameOrEmail(ctx context.Context, username, email string) (*models.User, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	query := `SELECT id, username, password, email, created_at FROM t_user WHERE username = $1 OR email = $2`

	var user models.User
	err := s.db.QueryRowContext(ctx, query, username, email).Scan(
		&user.ID, &user.Username, &user.Password, &user.Email, &user.CreatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}

// StoreData implements the DataStorage interface for storing data records with version control.
func (s *SQLStorage) StoreData(ctx context.Context, userID string, data *pb.DataRecord) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	if data.Version > 1 {
		var currentVersion int64
		err := tx.QueryRowContext(ctx,
			"SELECT version FROM t_user_data WHERE id = $1 AND user_id = $2",
			data.Id, userID).Scan(&currentVersion)

		if err != nil && err != sql.ErrNoRows {
			return fmt.Errorf("failed to check version: %w", err)
		}

		if currentVersion >= data.Version {
			return errors.ErrVersionConflict
		}
	}

	metadataJSON, err := json.Marshal(data.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		INSERT INTO t_user_data (id, user_id, type, name, encrypted_data, metadata, version, updated_at, deleted)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (id) DO UPDATE SET
			encrypted_data = EXCLUDED.encrypted_data,
			metadata = EXCLUDED.metadata,
			version = EXCLUDED.version,
			updated_at = EXCLUDED.updated_at,
			deleted = EXCLUDED.deleted
		WHERE t_user_data.user_id = $2
	`

	_, err = tx.ExecContext(ctx, query,
		data.Id, userID, data.Type, data.Name,
		data.EncryptedData, metadataJSON,
		data.Version, data.UpdatedAt, data.Deleted)

	if err != nil {
		return fmt.Errorf("failed to store data: %w", err)
	}

	return tx.Commit()
}

// RetrieveData implements the DataStorage interface for retrieving data records.
func (s *SQLStorage) RetrieveData(ctx context.Context, userID, dataID string) (*pb.DataRecord, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	query := `
		SELECT id, type, name, encrypted_data, metadata, version, updated_at, deleted
		FROM t_user_data 
		WHERE id = $1 AND user_id = $2
	`

	var data pb.DataRecord
	var metadataJSON []byte
	var deleted bool

	err := s.db.QueryRowContext(ctx, query, dataID, userID).Scan(
		&data.Id, &data.Type, &data.Name,
		&data.EncryptedData, &metadataJSON,
		&data.Version, &data.UpdatedAt, &deleted)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.ErrNotFound
		}
		return nil, fmt.Errorf("failed to retrieve data: %w", err)
	}

	if deleted {
		return nil, errors.ErrDeleted
	}

	data.Deleted = deleted
	if metadataJSON != nil {
		if err := json.Unmarshal(metadataJSON, &data.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata for record %s: %w", dataID, err)
		}
	}

	return &data, nil
}

// DeleteData implements the DataStorage interface for soft-deleting data records.
func (s *SQLStorage) DeleteData(ctx context.Context, userID, dataID string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(ctx, `
		UPDATE t_user_data 
		SET deleted = true, updated_at = $1, version = version + 1
		WHERE id = $2 AND user_id = $3 AND deleted = false
	`, time.Now().Unix(), dataID, userID)

	if err != nil {
		return fmt.Errorf("failed to delete data: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return errors.ErrNotFound
	}

	return tx.Commit()
}

// ListData implements the DataStorage interface for listing user data records.
func (s *SQLStorage) ListData(ctx context.Context, userID string, filterType pb.DataType) ([]*pb.DataRecord, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	var rows *sql.Rows
	var err error

	noFilter := pb.DataType(-1)

	if filterType == noFilter {
		rows, err = s.db.QueryContext(ctx, `
			SELECT id, type, name, encrypted_data, metadata, version, updated_at, deleted
			FROM t_user_data 
			WHERE user_id = $1 AND deleted = false
			ORDER BY updated_at DESC
		`, userID)
	} else {
		rows, err = s.db.QueryContext(ctx, `
			SELECT id, type, name, encrypted_data, metadata, version, updated_at, deleted
			FROM t_user_data 
			WHERE user_id = $1 AND type = $2 AND deleted = false
			ORDER BY updated_at DESC
		`, userID, filterType)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to query data: %w", err)
	}
	defer rows.Close()

	var items []*pb.DataRecord
	for rows.Next() {
		var data pb.DataRecord
		var metadataJSON []byte
		var deleted bool

		if err := rows.Scan(&data.Id, &data.Type, &data.Name,
			&data.EncryptedData, &metadataJSON,
			&data.Version, &data.UpdatedAt, &deleted); err != nil {
			log.Printf("Failed to scan row in ListData: %v", err)
			continue
		}

		data.Deleted = deleted
		if metadataJSON != nil {
			if err := json.Unmarshal(metadataJSON, &data.Metadata); err != nil {
				log.Printf("Failed to unmarshal metadata for record %s in ListData: %v", data.Id, err)
				data.Metadata = make(map[string]string)
			}
		}
		items = append(items, &data)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return items, nil
}

// ProcessSync implements the DataStorage interface for data synchronization.
func (s *SQLStorage) ProcessSync(ctx context.Context, userID string, localChanges []*pb.DataRecord, lastSync int64) ([]*pb.DataRecord, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	for _, change := range localChanges {
		var metadataJSON []byte
		if change.Metadata != nil {
			var err error
			metadataJSON, err = json.Marshal(change.Metadata)
			if err != nil {
				log.Printf("Failed to marshal metadata for record %s in ProcessSync: %v", change.Id, err)
				metadataJSON = []byte("{}")
			}
		}

		if change.Deleted {
			_, err = tx.ExecContext(ctx, `
				UPDATE t_user_data 
				SET deleted = true, updated_at = $1, version = version + 1
				WHERE id = $2 AND user_id = $3
			`, time.Now().Unix(), change.Id, userID)
		} else {
			_, err = tx.ExecContext(ctx, `
				INSERT INTO t_user_data (id, user_id, type, name, encrypted_data, metadata, version, updated_at, deleted)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
				ON CONFLICT (id) DO UPDATE SET
					encrypted_data = EXCLUDED.encrypted_data,
					metadata = EXCLUDED.metadata,
					version = EXCLUDED.version,
					updated_at = EXCLUDED.updated_at,
					deleted = EXCLUDED.deleted
				WHERE t_user_data.user_id = $2
			`, change.Id, userID, change.Type, change.Name,
				change.EncryptedData, metadataJSON,
				change.Version, change.UpdatedAt, change.Deleted)
		}

		if err != nil {
			return nil, fmt.Errorf("failed to apply change for %s: %w", change.Id, err)
		}
	}

	rows, err := tx.QueryContext(ctx, `
		SELECT id, type, name, encrypted_data, metadata, version, updated_at, deleted
		FROM t_user_data 
		WHERE user_id = $1 AND updated_at > $2
		ORDER BY updated_at DESC
	`, userID, lastSync)

	if err != nil {
		return nil, fmt.Errorf("failed to query server data: %w", err)
	}
	defer rows.Close()

	var serverData []*pb.DataRecord
	for rows.Next() {
		var data pb.DataRecord
		var metadataJSON []byte
		var deleted bool

		if err := rows.Scan(&data.Id, &data.Type, &data.Name,
			&data.EncryptedData, &metadataJSON,
			&data.Version, &data.UpdatedAt, &deleted); err != nil {
			log.Printf("Failed to scan row in ProcessSync: %v", err)
			continue
		}

		data.Deleted = deleted
		if metadataJSON != nil {
			if err := json.Unmarshal(metadataJSON, &data.Metadata); err != nil {
				log.Printf("Failed to unmarshal metadata for record %s in ProcessSync: %v", data.Id, err)
				data.Metadata = make(map[string]string)
			}
		}
		serverData = append(serverData, &data)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return serverData, nil
}

// Ping implements the DataStorage interface for database health check.
func (s *SQLStorage) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}
