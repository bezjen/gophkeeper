package service

import (
	"context"
	"errors"
	"github.com/bezjen/gophkeeper/internal/server/auth"
	errors2 "github.com/bezjen/gophkeeper/internal/server/errors"
	"github.com/bezjen/gophkeeper/internal/server/middleware"
	"github.com/bezjen/gophkeeper/internal/server/models"
	storage2 "github.com/bezjen/gophkeeper/internal/server/storage"
	"time"

	pb "github.com/bezjen/gophkeeper/pkg/proto"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Service struct {
	pb.UnimplementedGophKeeperServer
	storage storage2.Storage
	auth    auth.ServiceInterface
}

func NewService(storage storage2.Storage, auth auth.ServiceInterface) *Service {
	return &Service{
		storage: storage,
		auth:    auth,
	}
}

func (s *Service) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	authReq := &models.AuthRequest{
		Username: req.Username,
		Password: req.Password,
		Email:    req.Email,
	}

	authResp, err := s.auth.Register(ctx, authReq)
	if err != nil {
		return nil, err
	}

	return &pb.RegisterResponse{
		UserId: authResp.UserId,
		Token:  authResp.Token,
	}, nil
}

func (s *Service) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	authReq := &models.AuthRequest{
		Username: req.Username,
		Password: req.Password,
	}

	authResp, err := s.auth.Login(ctx, authReq)
	if err != nil {
		return nil, err
	}

	return &pb.LoginResponse{
		Token:  authResp.Token,
		UserId: authResp.UserId,
	}, nil
}

func (s *Service) StoreData(ctx context.Context, req *pb.StoreRequest) (*pb.StoreResponse, error) {
	userID, err := middleware.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	data := req.Data
	if data.Id == "" {
		data.Id = uuid.New().String()
	}

	if data.Version == 0 {
		data.Version = 1
	} else {
		data.Version = data.Version + 1
	}

	if data.UpdatedAt == 0 {
		data.UpdatedAt = time.Now().Unix()
	}

	if err := s.storage.StoreData(ctx, userID, data); err != nil {
		if errors.Is(err, errors2.ErrVersionConflict) {
			return nil, status.Error(codes.FailedPrecondition, "version conflict")
		}
		return nil, status.Errorf(codes.Internal, "failed to store data: %v", err)
	}

	return &pb.StoreResponse{
		Id:      data.Id,
		Version: data.Version,
	}, nil
}

func (s *Service) RetrieveData(ctx context.Context, req *pb.RetrieveRequest) (*pb.RetrieveResponse, error) {
	userID, err := middleware.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	data, err := s.storage.RetrieveData(ctx, userID, req.Id)
	if err != nil {
		if errors.Is(err, errors2.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "data not found")
		}
		if errors.Is(err, errors2.ErrDeleted) {
			return nil, status.Error(codes.NotFound, "data was deleted")
		}
		return nil, status.Errorf(codes.Internal, "failed to retrieve data: %v", err)
	}

	return &pb.RetrieveResponse{Data: data}, nil
}

func (s *Service) DeleteData(ctx context.Context, req *pb.DeleteRequest) (*pb.DeleteResponse, error) {
	userID, err := middleware.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if err := s.storage.DeleteData(ctx, userID, req.Id); err != nil {
		if errors.Is(err, errors2.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "data not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to delete data: %v", err)
	}

	return &pb.DeleteResponse{Success: true}, nil
}

func (s *Service) ListData(ctx context.Context, req *pb.ListRequest) (*pb.ListResponse, error) {
	userID, err := middleware.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	items, err := s.storage.ListData(ctx, userID, req.FilterType)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list data: %v", err)
	}

	return &pb.ListResponse{Items: items}, nil
}

func (s *Service) Sync(ctx context.Context, req *pb.SyncRequest) (*pb.SyncResponse, error) {
	userID, err := middleware.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	serverData, err := s.storage.ProcessSync(ctx, userID, req.LocalChanges, req.LastSync)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to sync data: %v", err)
	}

	return &pb.SyncResponse{
		ServerData:  serverData,
		CurrentTime: time.Now().Unix(),
	}, nil
}

func (s *Service) Ping(ctx context.Context, req *pb.PingRequest) (*pb.PingResponse, error) {
	err := s.storage.Ping(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to ping data: %v", err)
	}
	return &pb.PingResponse{
		Message:   "pong: " + req.Message,
		Timestamp: time.Now().Unix(),
	}, nil
}
