package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/bezjen/gophkeeper/internal/server/auth"
	"github.com/bezjen/gophkeeper/internal/server/database"
	"github.com/bezjen/gophkeeper/internal/server/middleware"
	serv "github.com/bezjen/gophkeeper/internal/server/service"
	stor "github.com/bezjen/gophkeeper/internal/server/storage"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	pb "github.com/bezjen/gophkeeper/api/gophkeeper/v1"

	"google.golang.org/grpc"
)

// Global build information variables
// These are set during build process using ldflags
var (
	buildVersion string
	buildDate    string
	buildCommit  string
)

// main is the entry point for the GophKeeper server application.
// It initializes the database, sets up authentication and storage services,
// creates a gRPC server with authentication middleware, and starts listening
// for incoming connections on the specified port.
func main() {
	printBuildInfo()
	port := flag.String("port", "8080", "Server port")
	dbDSN := flag.String("db-dsn", "gophkeeper.db", "Database DSN")
	jwtSecret := flag.String("jwt-secret", "", "JWT secret key (required)")
	flag.Parse()

	if *jwtSecret == "" {
		log.Fatal("JWT secret key is required. Use -jwt-secret flag")
	}

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

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	serverErr := make(chan error, 1)
	go func() {
		log.Printf("Server starting on port %s", *port)
		serverErr <- grpcServer.Serve(lis)
	}()

	select {
	case err := <-serverErr:
		log.Fatalf("Server failed: %v", err)
	case <-stop:
		log.Println("Shutting down server...")

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		stopped := make(chan struct{})
		go func() {
			grpcServer.GracefulStop()
			close(stopped)
		}()

		select {
		case <-stopped:
			log.Println("Server stopped gracefully")
		case <-ctx.Done():
			log.Println("Graceful shutdown timed out, forcing stop")
			grpcServer.Stop()
		}
	}
}

// printBuildInfo outputs build version, date and commit information to the log.
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
