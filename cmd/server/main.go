package main

import (
	"flag"
	"fmt"
	"github.com/bezjen/gophkeeper/internal/server/auth"
	"github.com/bezjen/gophkeeper/internal/server/database"
	"github.com/bezjen/gophkeeper/internal/server/middleware"
	serv "github.com/bezjen/gophkeeper/internal/server/service"
	stor "github.com/bezjen/gophkeeper/internal/server/storage"
	"log"
	"net"

	pb "github.com/bezjen/gophkeeper/pkg/proto"

	"google.golang.org/grpc"
)

// Global build information variables
// These are set during build process using ldflags
var (
	buildVersion string
	buildDate    string
	buildCommit  string
)

func main() {
	printBuildInfo()
	port := flag.String("port", "8080", "Server port")
	dbDSN := flag.String("db-dsn", "gophkeeper.db", "Database DSN")
	jwtSecret := flag.String("jwt-secret", "test-secret-key", "JWT secret key")
	flag.Parse()

	db, err := database.InitDatabase(*dbDSN)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	storage := stor.NewStorage(db)
	authService := auth.NewAuthService(*jwtSecret, storage)
	service := serv.NewService(storage, authService)

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

// printBuildInfo outputs build version, date and commit information
func printBuildInfo() {
	version := buildVersion
	if version == "" {
		version = "N/A"
	}

	date := buildDate
	if date == "" {
		date = "N/A"
	}

	commit := buildCommit
	if commit == "" {
		commit = "N/A"
	}

	log.Printf("Build version: %s\n", version)
	log.Printf("Build date: %s\n", date)
	log.Printf("Build commit: %s\n", commit)
}
