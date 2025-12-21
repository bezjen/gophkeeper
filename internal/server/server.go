package server

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	pb "github.com/bezjen/gophkeeper/pkg/proto"
)

type Server struct {
	pb.UnimplementedGophKeeperServer
	db     *sql.DB
	secret []byte
}

func NewServer(db *sql.DB, secret string) *Server {
	return &Server{
		db:     db,
		secret: []byte(secret),
	}
}

func (s *Server) RegisterServices(grpcServer *grpc.Server) {
	pb.RegisterGophKeeperServer(grpcServer, s)
}

func (s *Server) AuthInterceptor(ctx context.Context, req interface{},
	info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {

	if info.FullMethod == "/gophkeeper.proto.GophKeeper/Login" ||
		info.FullMethod == "/gophkeeper.proto.GophKeeper/Register" {
		return handler(ctx, req)
	}

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing metadata")
	}

	tokens := md.Get("authorization")
	if len(tokens) == 0 {
		return nil, status.Error(codes.Unauthenticated, "missing authorization token")
	}

	token := tokens[0]
	userID, err := s.validateToken(token)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid token")
	}

	ctx = context.WithValue(ctx, "userID", userID)
	return handler(ctx, req)
}

func (s *Server) validateToken(tokenString string) (string, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.secret, nil
	})

	if err != nil || !token.Valid {
		return "", fmt.Errorf("invalid token: %v", err)
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

func (s *Server) generateToken(userID, username string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":  userID,
		"username": username,
		"exp":      time.Now().Add(24 * time.Hour).Unix(),
		"iat":      time.Now().Unix(),
	})

	return token.SignedString(s.secret)
}

func (s *Server) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	log.Printf("Register attempt for user: %s", req.Username)

	var existingID string
	err := s.db.QueryRow(
		"SELECT id FROM t_user WHERE username = $1 OR email = $2",
		req.Username, req.Email,
	).Scan(&existingID)
	if err == nil {
		return nil, status.Error(codes.AlreadyExists, "user already exists")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to hash password: %v", err)
	}

	userID := uuid.New().String()
	createdAt := time.Now()

	_, err = s.db.Exec(
		"INSERT INTO t_user (id, username, password, email, created_at) VALUES ($1, $2, $3, $4, $5)",
		userID, req.Username, string(hashedPassword), req.Email, createdAt,
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create user: %v", err)
	}

	token, err := s.generateToken(userID, req.Username)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to generate token: %v", err)
	}

	return &pb.RegisterResponse{
		UserId: userID,
		Token:  token,
	}, nil
}

func (s *Server) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	var userID, username, passwordHash string
	err := s.db.QueryRow(
		"SELECT id, username, password FROM t_user WHERE username = $1",
		req.Username,
	).Scan(&userID, &username, &passwordHash)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, status.Error(codes.Unauthenticated, "invalid credentials")
		}
		return nil, status.Errorf(codes.Internal, "database error: %v", err)
	}

	err = bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password))
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid credentials")
	}

	token, err := s.generateToken(userID, username)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to generate token: %v", err)
	}

	return &pb.LoginResponse{
		Token:  token,
		UserId: userID,
	}, nil
}

func (s *Server) extractUserID(ctx context.Context) (string, error) {
	userID, ok := ctx.Value("userID").(string)
	if !ok {
		return "", status.Error(codes.Unauthenticated, "authentication required")
	}
	return userID, nil
}
