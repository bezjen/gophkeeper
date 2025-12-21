//go:generate mockery --name=UserStorage --inpackage --case=underscore
//go:generate mockery --name=AuthServiceInterface --inpackage --case=underscore
package server

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AuthServiceInterface interface {
	Register(ctx context.Context, req *AuthRequest) (*AuthResponse, error)
	Login(ctx context.Context, req *AuthRequest) (*AuthResponse, error)
	GenerateToken(userID, username string) (string, error)
	ValidateToken(tokenString string) (string, error)
}

type UserStorage interface {
	CreateUser(ctx context.Context, user *User) error
	GetUserByUsername(ctx context.Context, username string) (*User, error)
	GetUserByUsernameOrEmail(ctx context.Context, username, email string) (*User, error)
}

type AuthService struct {
	secret  []byte
	storage UserStorage
}

type User struct {
	ID        string
	Username  string
	Password  string
	Email     string
	CreatedAt time.Time
}

type AuthRequest struct {
	Username string
	Password string
	Email    string
}

type AuthResponse struct {
	UserId string
	Token  string
}

func NewAuthService(secret string, storage UserStorage) *AuthService {
	return &AuthService{
		secret:  []byte(secret),
		storage: storage,
	}
}

func (a *AuthService) Register(ctx context.Context, req *AuthRequest) (*AuthResponse, error) {
	existingUser, err := a.storage.GetUserByUsernameOrEmail(ctx, req.Username, req.Email)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return nil, status.Errorf(codes.Internal, "failed to check user existence: %v", err)
	}
	if existingUser != nil {
		return nil, status.Error(codes.AlreadyExists, "user already exists")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to hash password: %v", err)
	}

	user := &User{
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

	return &AuthResponse{
		UserId: user.ID,
		Token:  token,
	}, nil
}

func (a *AuthService) Login(ctx context.Context, req *AuthRequest) (*AuthResponse, error) {
	user, err := a.storage.GetUserByUsername(ctx, req.Username)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
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

	return &AuthResponse{
		Token:  token,
		UserId: user.ID,
	}, nil
}

func (a *AuthService) GenerateToken(userID, username string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":  userID,
		"username": username,
		"exp":      time.Now().Add(24 * time.Hour).Unix(),
		"iat":      time.Now().Unix(),
	})

	return token.SignedString(a.secret)
}

func (a *AuthService) ValidateToken(tokenString string) (string, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return a.secret, nil
	})

	if err != nil || !token.Valid {
		return "", ErrInvalidToken
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
