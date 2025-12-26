// Package middleware provides gRPC interceptors for authentication and authorization.
package middleware

import (
	"context"
	"github.com/bezjen/gophkeeper/internal/server/auth"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// NewAuthInterceptor creates a gRPC unary interceptor that validates JWT tokens.
func NewAuthInterceptor(auth auth.ServiceInterface) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{},
		info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {

		if info.FullMethod == "/gophkeeper.proto.GophKeeper/Login" ||
			info.FullMethod == "/gophkeeper.proto.GophKeeper/Register" ||
			info.FullMethod == "/gophkeeper.proto.GophKeeper/Ping" {
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

		userID, err := auth.ValidateToken(tokens[0])
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, "invalid token")
		}

		ctx = context.WithValue(ctx, UserIDKey{}, userID)
		return handler(ctx, req)
	}
}

// UserIDKey is the context key type for storing user ID.
type UserIDKey struct{}

// GetUserIDFromContext extracts the user ID from the context.
func GetUserIDFromContext(ctx context.Context) (string, error) {
	userID, ok := ctx.Value(UserIDKey{}).(string)
	if !ok {
		return "", status.Error(codes.Unauthenticated, "authentication required")
	}
	return userID, nil
}
