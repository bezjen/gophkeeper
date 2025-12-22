package main

import (
	"fmt"
	cli2 "github.com/bezjen/gophkeeper/internal/client/cli"
	"log"
	"os"
	"syscall"

	"github.com/bezjen/gophkeeper/internal/client"
	"github.com/urfave/cli/v2"
	"golang.org/x/term"
)

// Global build information variables
// These are set during build process using ldflags
var (
	buildVersion string
	buildDate    string
	buildCommit  string
)

func main() {
	app := &cli.App{
		Name:    "gophkeeper",
		Version: fmt.Sprintf("%s", buildVersion),
		Usage:   "Secure password manager",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "server",
				Value: "localhost:8080",
				Usage: "Server address",
			},
		},
		Before: func(c *cli.Context) error {
			commandName := c.Args().First()
			if commandName == "" {
				return nil
			}

			skipAuth := map[string]bool{
				"register": true,
				"login":    true,
				"version":  true,
				"help":     true,
			}

			if skipAuth[commandName] {
				return nil
			}

			serverAddr := c.String("server")
			cl, err := client.NewDefaultClient(serverAddr)
			if err != nil {
				return fmt.Errorf("failed to initialize client: %w", err)
			}
			c.App.Metadata["client"] = cl

			if !cl.IsAuthenticated() {
				return fmt.Errorf("you are not logged in. Please run 'login' command first")
			}

			fmt.Print("Enter Master Password to unlock vault: ")
			bytePassword, err := term.ReadPassword(int(syscall.Stdin))
			fmt.Println()
			if err != nil {
				return fmt.Errorf("failed to read password: %w", err)
			}

			if err := cl.InitSession(string(bytePassword)); err != nil {
				return fmt.Errorf("failed to unlock vault: %w", err)
			}

			return nil
		},
		After: func(c *cli.Context) error {
			if cl, ok := c.App.Metadata["client"].(*client.Client); ok && cl != nil {
				return cl.Close()
			}
			return nil
		},
	}

	cli2.RegisterCommands(app)

	app.Commands = append(app.Commands, &cli.Command{
		Name:  "version",
		Usage: "Print version info",
		Action: func(c *cli.Context) error {
			printBuildInfo()
			return nil
		},
	})

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
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
