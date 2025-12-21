package server

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"time"

	pb "github.com/bezjen/gophkeeper/pkg/proto"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Server) StoreData(ctx context.Context, req *pb.StoreRequest) (*pb.StoreResponse, error) {
	userID, err := s.extractUserID(ctx)
	if err != nil {
		return nil, err
	}

	dataID := req.Data.Id
	if dataID == "" {
		dataID = generateDataID(userID, req.Data.Name)
	}

	metadataJSON, _ := json.Marshal(req.Data.Metadata)
	version := int64(1)
	if req.Data.Version > 0 {
		version = req.Data.Version + 1
	}

	updatedAt := time.Now().Unix()
	if req.Data.UpdatedAt > 0 {
		updatedAt = req.Data.UpdatedAt
	}

	_, err = s.db.Exec(`
		INSERT INTO t_user_data (id, user_id, type, name, encrypted_data, metadata, version, updated_at, deleted)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (id) DO UPDATE SET
			encrypted_data = EXCLUDED.encrypted_data,
			metadata = EXCLUDED.metadata,
			version = EXCLUDED.version,
			updated_at = EXCLUDED.updated_at,
			deleted = EXCLUDED.deleted
		WHERE t_user_data.user_id = $2
	`, dataID, userID, req.Data.Type, req.Data.Name,
		req.Data.EncryptedData, metadataJSON, version, updatedAt, req.Data.Deleted)

	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to store data: %v", err)
	}

	return &pb.StoreResponse{
		Id:      dataID,
		Version: version,
	}, nil
}

func (s *Server) RetrieveData(ctx context.Context, req *pb.RetrieveRequest) (*pb.RetrieveResponse, error) {
	userID, err := s.extractUserID(ctx)
	if err != nil {
		return nil, err
	}

	var data pb.DataRecord
	var metadataJSON []byte
	var deleted bool

	err = s.db.QueryRow(`
		SELECT id, type, name, encrypted_data, metadata, version, updated_at, deleted
		FROM t_user_data 
		WHERE id = $1 AND user_id = $2
	`, req.Id, userID).Scan(
		&data.Id, &data.Type, &data.Name,
		&data.EncryptedData, &metadataJSON,
		&data.Version, &data.UpdatedAt, &deleted,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, status.Error(codes.NotFound, "data not found")
		}
		return nil, status.Errorf(codes.Internal, "database error: %v", err)
	}

	if deleted {
		return nil, status.Error(codes.NotFound, "data was deleted")
	}

	data.Deleted = deleted
	if metadataJSON != nil {
		json.Unmarshal(metadataJSON, &data.Metadata)
	}

	return &pb.RetrieveResponse{Data: &data}, nil
}

func (s *Server) DeleteData(ctx context.Context, req *pb.DeleteRequest) (*pb.DeleteResponse, error) {
	userID, err := s.extractUserID(ctx)
	if err != nil {
		return nil, err
	}

	result, err := s.db.Exec(`
		UPDATE t_user_data 
		SET deleted = true, updated_at = $1, version = version + 1
		WHERE id = $2 AND user_id = $3 AND deleted = false
	`, time.Now().Unix(), req.Id, userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete: %v", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get rows affected: %v", err)
	}

	if rows == 0 {
		return nil, status.Error(codes.NotFound, "data not found or already deleted")
	}

	return &pb.DeleteResponse{Success: true}, nil
}

func (s *Server) ListData(ctx context.Context, req *pb.ListRequest) (*pb.ListResponse, error) {
	userID, err := s.extractUserID(ctx)
	if err != nil {
		return nil, err
	}

	var rows *sql.Rows
	if req.FilterType == pb.DataType(-1) {
		rows, err = s.db.Query(`
			SELECT id, type, name, encrypted_data, metadata, version, updated_at, deleted
			FROM t_user_data 
			WHERE user_id = $1 AND deleted = false
			ORDER BY updated_at DESC
		`, userID)
	} else {
		rows, err = s.db.Query(`
			SELECT id, type, name, encrypted_data, metadata, version, updated_at, deleted
			FROM t_user_data 
			WHERE user_id = $1 AND type = $2 AND deleted = false
			ORDER BY updated_at DESC
		`, userID, req.FilterType)
	}

	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to query: %v", err)
	}
	defer rows.Close()

	var items []*pb.DataRecord
	for rows.Next() {
		var data pb.DataRecord
		var metadataJSON []byte
		var deleted bool

		err := rows.Scan(&data.Id, &data.Type, &data.Name,
			&data.EncryptedData, &metadataJSON,
			&data.Version, &data.UpdatedAt, &deleted)
		if err != nil {
			continue
		}

		data.Deleted = deleted
		if metadataJSON != nil {
			json.Unmarshal(metadataJSON, &data.Metadata)
		}
		items = append(items, &data)
	}

	return &pb.ListResponse{Items: items}, nil
}

func (s *Server) Sync(ctx context.Context, req *pb.SyncRequest) (*pb.SyncResponse, error) {
	userID, err := s.extractUserID(ctx)
	if err != nil {
		return nil, err
	}

	for _, change := range req.LocalChanges {
		if change.Deleted {
			s.db.Exec(`
				UPDATE t_user_data 
				SET deleted = true, updated_at = $1, version = version + 1
				WHERE id = $2 AND user_id = $3
			`, time.Now().Unix(), change.Id, userID)
		} else {
			metadataJSON, _ := json.Marshal(change.Metadata)
			s.db.Exec(`
				INSERT INTO t_user_data (id, user_id, type, name, encrypted_data, metadata, version, updated_at, deleted)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
				ON CONFLICT (id) DO UPDATE SET
					encrypted_data = EXCLUDED.encrypted_data,
					metadata = EXCLUDED.metadata,
					version = EXCLUDED.version,
					updated_at = EXCLUDED.updated_at,
					deleted = EXCLUDED.deleted
			`, change.Id, userID, change.Type, change.Name,
				change.EncryptedData, metadataJSON,
				change.Version, change.UpdatedAt, change.Deleted)
		}
	}

	rows, err := s.db.Query(`
		SELECT id, type, name, encrypted_data, metadata, version, updated_at, deleted
		FROM t_user_data 
		WHERE user_id = $1 AND updated_at > $2
		ORDER BY updated_at DESC
	`, userID, req.LastSync)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to query data: %v", err)
	}
	defer rows.Close()

	var serverData []*pb.DataRecord
	for rows.Next() {
		var data pb.DataRecord
		var metadataJSON []byte
		var deleted bool

		err := rows.Scan(&data.Id, &data.Type, &data.Name,
			&data.EncryptedData, &metadataJSON,
			&data.Version, &data.UpdatedAt, &deleted)
		if err != nil {
			continue
		}

		data.Deleted = deleted
		if metadataJSON != nil {
			json.Unmarshal(metadataJSON, &data.Metadata)
		}
		serverData = append(serverData, &data)
	}

	return &pb.SyncResponse{
		ServerData:  serverData,
		CurrentTime: time.Now().Unix(),
	}, nil
}

func generateDataID(userID, name string) string {
	hash := sha256.Sum256([]byte(userID + name + time.Now().String()))
	return hex.EncodeToString(hash[:16])
}
