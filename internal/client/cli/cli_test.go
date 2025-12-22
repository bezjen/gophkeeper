package cli

import (
	"testing"

	"github.com/urfave/cli/v2"
)

func TestParseMetadata(t *testing.T) {
	testCases := []struct {
		name     string
		input    []string
		expected map[string]string
	}{
		{
			name:     "empty input",
			input:    []string{},
			expected: map[string]string{},
		},
		{
			name:  "single key-value pair",
			input: []string{"key=value"},
			expected: map[string]string{
				"key": "value",
			},
		},
		{
			name:  "multiple key-value pairs",
			input: []string{"key1=value1", "key2=value2", "key3=value3"},
			expected: map[string]string{
				"key1": "value1",
				"key2": "value2",
				"key3": "value3",
			},
		},
		{
			name:  "values with equals sign",
			input: []string{"url=https://example.com?param=value"},
			expected: map[string]string{
				"url": "https://example.com?param=value",
			},
		},
		{
			name:     "invalid format (no equals)",
			input:    []string{"invalid"},
			expected: map[string]string{},
		},
		{
			name:  "mixed valid and invalid",
			input: []string{"key1=value1", "invalid", "key2=value2"},
			expected: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
		},
		{
			name:  "empty value",
			input: []string{"key="},
			expected: map[string]string{
				"key": "",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := parseMetadata(tc.input)

			if len(result) != len(tc.expected) {
				t.Errorf("Expected %d entries, got %d", len(tc.expected), len(result))
			}

			for key, expectedValue := range tc.expected {
				if result[key] != expectedValue {
					t.Errorf("For key %q: expected %q, got %q",
						key, expectedValue, result[key])
				}
			}
		})
	}
}

func TestRegisterCommands(t *testing.T) {
	app := &cli.App{}

	// Проверяем, что функция не паникует
	RegisterCommands(app)

	if len(app.Commands) == 0 {
		t.Error("Expected commands to be registered")
	}

	// Проверяем наличие основных команд
	expectedCommands := []string{
		"register", "login", "store", "get", "list", "delete", "sync",
	}

	commandNames := make(map[string]bool)
	for _, cmd := range app.Commands {
		commandNames[cmd.Name] = true
	}

	for _, expected := range expectedCommands {
		if !commandNames[expected] {
			t.Errorf("Expected command %q to be registered", expected)
		}
	}
}

func TestStoreSubcommands(t *testing.T) {
	app := &cli.App{}
	RegisterCommands(app)

	// Находим команду store
	var storeCmd *cli.Command
	for _, cmd := range app.Commands {
		if cmd.Name == "store" {
			storeCmd = cmd
			break
		}
	}

	if storeCmd == nil {
		t.Fatal("Store command not found")
	}

	// Проверяем подкоманды store
	expectedSubcommands := []string{"login", "text", "card", "file"}
	subcommandNames := make(map[string]bool)
	for _, subcmd := range storeCmd.Subcommands {
		subcommandNames[subcmd.Name] = true
	}

	for _, expected := range expectedSubcommands {
		if !subcommandNames[expected] {
			t.Errorf("Expected store subcommand %q", expected)
		}
	}
}

func TestCommandFlags(t *testing.T) {
	app := &cli.App{}
	RegisterCommands(app)

	testCases := []struct {
		commandName   string
		subcommand    string
		requiredFlags []string
	}{
		{
			commandName:   "register",
			requiredFlags: []string{"username", "password", "email"},
		},
		{
			commandName:   "login",
			requiredFlags: []string{"username", "password"},
		},
		{
			commandName:   "store",
			subcommand:    "login",
			requiredFlags: []string{"name", "username", "password"},
		},
		{
			commandName:   "get",
			requiredFlags: []string{"id"},
		},
		{
			commandName:   "delete",
			requiredFlags: []string{"id"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.commandName, func(t *testing.T) {
			var cmd *cli.Command

			if tc.subcommand != "" {
				// Ищем родительскую команду
				for _, parentCmd := range app.Commands {
					if parentCmd.Name == tc.commandName {
						// Ищем подкоманду
						for _, subcmd := range parentCmd.Subcommands {
							if subcmd.Name == tc.subcommand {
								cmd = subcmd
								break
							}
						}
						break
					}
				}
			} else {
				// Ищем обычную команду
				for _, appCmd := range app.Commands {
					if appCmd.Name == tc.commandName {
						cmd = appCmd
						break
					}
				}
			}

			if cmd == nil {
				t.Fatalf("Command %s not found", tc.commandName)
			}

			// Проверяем флаги
			flagNames := make(map[string]bool)
			for _, flag := range cmd.Flags {
				// Получаем имя флага
				switch f := flag.(type) {
				case *cli.StringFlag:
					flagNames[f.Name] = f.Required
				case *cli.StringSliceFlag:
					flagNames[f.Name] = false
				}
			}

			for _, requiredFlag := range tc.requiredFlags {
				required, exists := flagNames[requiredFlag]
				if !exists {
					t.Errorf("Flag %q not found in command %s", requiredFlag, tc.commandName)
				} else if !required {
					t.Errorf("Flag %q should be required in command %s", requiredFlag, tc.commandName)
				}
			}
		})
	}
}
