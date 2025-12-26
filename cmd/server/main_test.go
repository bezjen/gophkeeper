package main

import (
	"bytes"
	"log"
	"testing"
)

func TestPrintBuildInfo(t *testing.T) {
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

	// Проверяем, что информация выводится
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

func TestPrintBuildInfoWithDefaults(t *testing.T) {
	// Захватываем вывод лога
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer func() {
		log.SetOutput(nil)
	}()

	// Сбрасываем значения
	buildVersion = ""
	buildDate = ""
	buildCommit = ""

	printBuildInfo()

	output := buf.String()

	// Проверяем, что выводится N/A при отсутствии значений
	expectedStrings := []string{
		"Build version: N/A",
		"Build date: N/A",
		"Build commit: N/A",
	}

	for _, expected := range expectedStrings {
		if !bytes.Contains([]byte(output), []byte(expected)) {
			t.Errorf("Expected output to contain %q, got: %s", expected, output)
		}
	}
}
