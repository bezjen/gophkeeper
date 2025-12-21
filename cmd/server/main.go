package main

import (
	"flag"
	"fmt"
	"log"
	"net"

	"github.com/bezjen/gophkeeper/internal/server"
	pb "github.com/bezjen/gophkeeper/pkg/proto"

	"google.golang.org/grpc"
)

func main() {
	port := flag.String("port", "8080", "Server port")
	dbDriver := flag.String("db-driver", "sqlite", "Database driver (sqlite or postgres)")
	dbDSN := flag.String("db-dsn", "gophkeeper.db", "Database DSN")
	jwtSecret := flag.String("jwt-secret", "test-secret-key", "JWT secret key")
	flag.Parse()

	db, err := server.InitDatabase(*dbDriver, *dbDSN)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	storage := server.NewStorage(db)
	authService := server.NewAuthService(*jwtSecret, storage)
	service := server.NewService(storage, authService)

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(server.NewAuthInterceptor(authService)),
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
