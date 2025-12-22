package client

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	pb "github.com/bezjen/gophkeeper/pkg/proto"

	"github.com/urfave/cli/v2"
)

func RegisterCommands(app *cli.App) {
	app.Commands = []*cli.Command{
		{
			Name:  "register",
			Usage: "Register new user",
			Flags: []cli.Flag{
				&cli.StringFlag{Name: "username", Required: true},
				&cli.StringFlag{Name: "password", Required: true},
				&cli.StringFlag{Name: "email", Required: true},
			},
			Action: func(c *cli.Context) error {
				serverAddr := c.String("server")
				cl, err := NewClient(serverAddr)
				if err != nil {
					return fmt.Errorf("failed to create client: %w", err)
				}
				defer cl.Close()

				return cl.Register(
					c.String("username"),
					c.String("password"),
					c.String("email"),
				)
			},
		},
		{
			Name:  "login",
			Usage: "Login to account",
			Flags: []cli.Flag{
				&cli.StringFlag{Name: "username", Required: true},
				&cli.StringFlag{Name: "password", Required: true},
			},
			Action: func(c *cli.Context) error {
				serverAddr := c.String("server")
				cl, err := NewClient(serverAddr)
				if err != nil {
					return fmt.Errorf("failed to create client: %w", err)
				}
				defer cl.Close()

				return cl.Login(
					c.String("username"),
					c.String("password"),
				)
			},
		},
		{
			Name:  "store",
			Usage: "Store new data",
			Subcommands: []*cli.Command{
				{
					Name:  "login",
					Usage: "Store login/password",
					Flags: []cli.Flag{
						&cli.StringFlag{Name: "name", Required: true},
						&cli.StringFlag{Name: "username", Required: true},
						&cli.StringFlag{Name: "password", Required: true},
						&cli.StringSliceFlag{Name: "meta"},
					},
					Action: func(c *cli.Context) error {
						cl, ok := c.App.Metadata["client"].(*Client)
						if !ok || cl == nil {
							return fmt.Errorf("client not initialized. Please login first")
						}

						metadata := parseMetadata(c.StringSlice("meta"))
						id, err := cl.StoreLoginPassword(
							c.String("name"),
							c.String("username"),
							c.String("password"),
							metadata,
						)
						if err != nil {
							return err
						}
						fmt.Printf("Stored with ID: %s\n", id)
						return nil
					},
				},
				{
					Name:  "text",
					Usage: "Store text data",
					Flags: []cli.Flag{
						&cli.StringFlag{Name: "name", Required: true},
						&cli.StringFlag{Name: "text", Required: true},
						&cli.StringSliceFlag{Name: "meta"},
					},
					Action: func(c *cli.Context) error {
						cl, ok := c.App.Metadata["client"].(*Client)
						if !ok || cl == nil {
							return fmt.Errorf("client not initialized. Please login first")
						}

						metadata := parseMetadata(c.StringSlice("meta"))
						id, err := cl.StoreText(
							c.String("name"),
							c.String("text"),
							metadata,
						)
						if err != nil {
							return err
						}
						fmt.Printf("Stored with ID: %s\n", id)
						return nil
					},
				},
				{
					Name:  "card",
					Usage: "Store bank card",
					Flags: []cli.Flag{
						&cli.StringFlag{Name: "name", Required: true},
						&cli.StringFlag{Name: "number", Required: true},
						&cli.StringFlag{Name: "holder", Required: true},
						&cli.StringFlag{Name: "expiry", Required: true},
						&cli.StringSliceFlag{Name: "meta"},
					},
					Action: func(c *cli.Context) error {
						cl, ok := c.App.Metadata["client"].(*Client)
						if !ok || cl == nil {
							return fmt.Errorf("client not initialized. Please login first")
						}

						metadata := parseMetadata(c.StringSlice("meta"))
						id, err := cl.StoreCard(
							c.String("name"),
							c.String("number"),
							c.String("holder"),
							c.String("expiry"),
							metadata,
						)
						if err != nil {
							return err
						}
						fmt.Printf("Stored with ID: %s\n", id)
						return nil
					},
				},
				{
					Name:  "file",
					Usage: "Store arbitrary binary file",
					Flags: []cli.Flag{
						&cli.StringFlag{Name: "path", Required: true, Usage: "Path to file"},
						&cli.StringFlag{Name: "name", Usage: "Custom name (optional)"},
						&cli.StringFlag{Name: "meta", Usage: "Metadata (key=value,k2=v2)"},
					},
					Action: func(c *cli.Context) error {
						cl, ok := c.App.Metadata["client"].(*Client)
						if !ok || cl == nil {
							return fmt.Errorf("client not initialized. Please login first")
						}

						path := c.String("path")
						metadata := parseMetadata(c.StringSlice("meta"))
						data, err := os.ReadFile(path)
						if err != nil {
							return fmt.Errorf("failed to read file: %w", err)
						}

						name := c.String("name")
						if name == "" {
							name = filepath.Base(path)
						}

						id, err := cl.StoreBinary(
							name,
							data,
							metadata,
						)
						if err != nil {
							return err
						}
						fmt.Printf("Stored with ID: %s\n", id)
						return nil
					},
				},
			},
		},
		{
			Name:  "get",
			Usage: "Get data by ID",
			Flags: []cli.Flag{
				&cli.StringFlag{Name: "id", Required: true},
			},
			Action: func(c *cli.Context) error {
				cl, ok := c.App.Metadata["client"].(*Client)
				if !ok || cl == nil {
					return fmt.Errorf("client not initialized. Please login first")
				}

				item, err := cl.GetData(c.String("id"))
				if err != nil {
					return err
				}

				fmt.Printf("Name: %s\n", item.Name)
				fmt.Printf("Type: %v\n", item.Type)
				fmt.Printf("Updated: %v\n", item.UpdatedAt)

				if len(item.Metadata) > 0 {
					fmt.Println("Metadata:")
					for k, v := range item.Metadata {
						fmt.Printf("  %s: %s\n", k, v)
					}
				}

				return nil
			},
		},
		{
			Name:  "list",
			Usage: "List all data",
			Flags: []cli.Flag{
				&cli.StringFlag{Name: "type"},
			},
			Action: func(c *cli.Context) error {
				cl, ok := c.App.Metadata["client"].(*Client)
				if !ok || cl == nil {
					return fmt.Errorf("client not initialized. Please login first")
				}

				var filterType pb.DataType = pb.DataType(-1)
				if typeStr := c.String("type"); typeStr != "" {
					proto := NewProtocol()
					filterType = proto.StringToDataType(typeStr)
				}

				items, err := cl.ListData(filterType)
				if err != nil {
					return err
				}

				for _, item := range items {
					fmt.Printf("%s: %s (%v)\n", item.ID, item.Name, item.Type)
				}
				return nil
			},
		},
		{
			Name:  "delete",
			Usage: "Delete data",
			Flags: []cli.Flag{
				&cli.StringFlag{Name: "id", Required: true},
			},
			Action: func(c *cli.Context) error {
				cl, ok := c.App.Metadata["client"].(*Client)
				if !ok || cl == nil {
					return fmt.Errorf("client not initialized. Please login first")
				}

				return cl.DeleteData(c.String("id"))
			},
		},
		{
			Name:  "sync",
			Usage: "Synchronize with server",
			Action: func(c *cli.Context) error {
				cl, ok := c.App.Metadata["client"].(*Client)
				if !ok || cl == nil {
					return fmt.Errorf("client not initialized. Please login first")
				}

				return cl.Sync()
			},
		},
	}
}

func parseMetadata(meta []string) map[string]string {
	result := make(map[string]string)
	for _, m := range meta {
		parts := strings.SplitN(m, "=", 2)
		if len(parts) == 2 {
			result[parts[0]] = parts[1]
		}
	}
	return result
}
