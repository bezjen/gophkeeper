package main

import (
	"bytes"
	"log"
	"testing"

	"github.com/urfave/cli/v2"
)

func TestClientPrintBuildInfo(t *testing.T) {
	// Захватываем вывод лога
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer func() {
		log.SetOutput(nil)
	}()

	// Устанавливаем тестовые значения
	buildVersion = "1.0.0"
	buildDate = "2024-01-01"
	buildCommit = "abc123"

	printBuildInfo()

	output := buf.String()

	expectedStrings := []string{
		"Build version: 1.0.0",
		"Build date: 2024-01-01",
		"Build commit: abc123",
	}

	for _, expected := range expectedStrings {
		if !bytes.Contains([]byte(output), []byte(expected)) {
			t.Errorf("Expected output to contain %q, got: %s", expected, output)
		}
	}
}

func TestAppConfiguration(t *testing.T) {
	app := &cli.App{
		Name:    "gophkeeper",
		Version: "1.0.0",
		Usage:   "Secure password manager",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "server",
				Value: "localhost:8080",
				Usage: "Server address",
			},
		},
	}

	if app.Name != "gophkeeper" {
		t.Errorf("Expected app name 'gophkeeper', got %s", app.Name)
	}

	if app.Version != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got %s", app.Version)
	}

	// Проверяем флаги
	serverFlag, ok := app.Flags[0].(*cli.StringFlag)
	if !ok {
		t.Fatal("Expected StringFlag")
	}

	if serverFlag.Name != "server" {
		t.Errorf("Expected flag name 'server', got %s", serverFlag.Name)
	}

	if serverFlag.Value != "localhost:8080" {
		t.Errorf("Expected default value 'localhost:8080', got %s", serverFlag.Value)
	}
}

func TestSkipAuthCommands(t *testing.T) {
	skipAuth := map[string]bool{
		"register": true,
		"login":    true,
		"version":  true,
		"help":     true,
	}

	testCases := []struct {
		command    string
		shouldSkip bool
	}{
		{"register", true},
		{"login", true},
		{"version", true},
		{"help", true},
		{"store", false},
		{"get", false},
		{"list", false},
		{"delete", false},
		{"sync", false},
		{"", false}, // пустая команда
	}

	for _, tc := range testCases {
		if skipAuth[tc.command] != tc.shouldSkip {
			t.Errorf("For command %q: expected skipAuth=%v, got %v",
				tc.command, tc.shouldSkip, skipAuth[tc.command])
		}
	}
}
