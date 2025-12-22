//go:generate mockery --name=AuthServiceInterface --output=../mocks --outpkg=mocks --case=underscore
package auth

import (
	"context"
	"errors"
	"fmt"
	errors2 "github.com/bezjen/gophkeeper/internal/server/errors"
	"github.com/bezjen/gophkeeper/internal/server/models"
	"github.com/bezjen/gophkeeper/internal/server/storage"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ServiceInterface interface {
	Register(ctx context.Context, req *models.AuthRequest) (*models.AuthResponse, error)
	Login(ctx context.Context, req *models.AuthRequest) (*models.AuthResponse, error)
	GenerateToken(userID, username string) (string, error)
	ValidateToken(tokenString string) (string, error)
}

type Service struct {
	secret  []byte
	storage storage.UserStorage
}

func NewAuthService(secret string, storage storage.UserStorage) *Service {
	return &Service{
		secret:  []byte(secret),
		storage: storage,
	}
}

func (a *Service) Register(ctx context.Context, req *models.AuthRequest) (*models.AuthResponse, error) {
	existingUser, err := a.storage.GetUserByUsernameOrEmail(ctx, req.Username, req.Email)
	if err != nil && !errors.Is(err, errors2.ErrNotFound) {
		return nil, status.Errorf(codes.Internal, "failed to check user existence: %v", err)
	}
	if existingUser != nil {
		return nil, status.Error(codes.AlreadyExists, "user already exists")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to hash password: %v", err)
	}

	user := &models.User{
		ID:        uuid.New().String(),
		Username:  req.Username,
		Password:  string(hashedPassword),
		Email:     req.Email,
		CreatedAt: time.Now(),
	}

	if err := a.storage.CreateUser(ctx, user); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create user: %v", err)
	}

	token, err := a.GenerateToken(user.ID, user.Username)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to generate token: %v", err)
	}

	return &models.AuthResponse{
		UserId: user.ID,
		Token:  token,
	}, nil
}

func (a *Service) Login(ctx context.Context, req *models.AuthRequest) (*models.AuthResponse, error) {
	user, err := a.storage.GetUserByUsername(ctx, req.Username)
	if err != nil {
		if errors.Is(err, errors2.ErrNotFound) {
			return nil, status.Error(codes.Unauthenticated, "invalid credentials")
		}
		return nil, status.Errorf(codes.Internal, "failed to get user: %v", err)
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password))
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid credentials")
	}

	token, err := a.GenerateToken(user.ID, user.Username)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to generate token: %v", err)
	}

	return &models.AuthResponse{
		Token:  token,
		UserId: user.ID,
	}, nil
}

func (a *Service) GenerateToken(userID, username string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":  userID,
		"username": username,
		"exp":      time.Now().Add(24 * time.Hour).Unix(),
		"iat":      time.Now().Unix(),
	})

	return token.SignedString(a.secret)
}

func (a *Service) ValidateToken(tokenString string) (string, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return a.secret, nil
	})

	if err != nil || !token.Valid {
		return "", errors2.ErrInvalidToken
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", fmt.Errorf("invalid token claims")
	}

	userID, ok := claims["user_id"].(string)
	if !ok {
		return "", fmt.Errorf("invalid user_id in token")
	}

	return userID, nil
}
