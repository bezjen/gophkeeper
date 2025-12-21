package main

import (
	"flag"
	"fmt"
	"github.com/bezjen/gophkeeper/internal/server/auth"
	"github.com/bezjen/gophkeeper/internal/server/database"
	"github.com/bezjen/gophkeeper/internal/server/middleware"
	"github.com/bezjen/gophkeeper/internal/server/service"
	"github.com/bezjen/gophkeeper/internal/server/storage"
	"log"
	"net"

	pb "github.com/bezjen/gophkeeper/pkg/proto"

	"google.golang.org/grpc"
)

func main() {
	port := flag.String("port", "8080", "Server port")
	dbDSN := flag.String("db-dsn", "gophkeeper.db", "Database DSN")
	jwtSecret := flag.String("jwt-secret", "test-secret-key", "JWT secret key")
	flag.Parse()

	db, err := database.InitDatabase(*dbDSN)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	storage := storage.NewStorage(db)
	authService := auth.NewAuthService(*jwtSecret, storage)
	service := service.NewService(storage, authService)

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(middleware.NewAuthInterceptor(authService)),
	)

	pb.RegisterGophKeeperServer(grpcServer, service)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", *port))
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	log.Printf("Server starting on port %s", *port)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
