package cmd

import (
	"reflect"
	"testing"
)

func TestFindVariablesWithDefaults(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected []Variable
	}{
		{
			name:    "simple variable without default",
			content: "Hello ${name}!",
			expected: []Variable{
				{Name: "name", DefaultValue: "", HasDefault: false},
			},
		},
		{
			name:    "variable with default value",
			content: "Hello ${name:-World}!",
			expected: []Variable{
				{Name: "name", DefaultValue: "World", HasDefault: true},
			},
		},
		{
			name:    "variable with empty default",
			content: "Hello ${name:-}!",
			expected: []Variable{
				{Name: "name", DefaultValue: "", HasDefault: true},
			},
		},
		{
			name:    "multiple variables mixed",
			content: "Hello ${name:-World}! Your directory is ${directory} and your age is ${age:-25}.",
			expected: []Variable{
				{Name: "name", DefaultValue: "World", HasDefault: true},
				{Name: "directory", DefaultValue: "", HasDefault: false},
				{Name: "age", DefaultValue: "25", HasDefault: true},
			},
		},
		{
			name:    "duplicate variables",
			content: "Hello ${name:-World}! Again, hello ${name:-Universe}!",
			expected: []Variable{
				{Name: "name", DefaultValue: "World", HasDefault: true},
			},
		},
		{
			name:     "no variables",
			content:  "Hello World!",
			expected: nil,
		},
		{
			name:    "complex default values",
			content: "Path: ${path:-/tmp/default/path} and URL: ${url:-https://example.com/api}",
			expected: []Variable{
				{Name: "path", DefaultValue: "/tmp/default/path", HasDefault: true},
				{Name: "url", DefaultValue: "https://example.com/api", HasDefault: true},
			},
		},
		{
			name:    "default with spaces",
			content: "Message: ${msg:-Hello World}",
			expected: []Variable{
				{Name: "msg", DefaultValue: "Hello World", HasDefault: true},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findVariablesWithDefaults(tt.content)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("findVariablesWithDefaults() = %+v, want %+v", result, tt.expected)
			}
		})
	}
}

func TestFindVariablesBackwardCompatibility(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected []string
	}{
		{
			name:     "simple variables",
			content:  "Hello ${name} from ${location}!",
			expected: []string{"name", "location"},
		},
		{
			name:     "variables with defaults should still return names",
			content:  "Hello ${name:-World} from ${location:-Earth}!",
			expected: []string{"name", "location"},
		},
		{
			name:     "mixed variables",
			content:  "Hello ${name} from ${location:-Earth}!",
			expected: []string{"name", "location"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findVariables(tt.content)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("findVariables() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestValidateVariables(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		variables map[string]string
		wantError bool
	}{
		{
			name:      "all required variables provided",
			content:   "Hello ${name} from ${location}!",
			variables: map[string]string{"name": "John", "location": "NYC"},
			wantError: false,
		},
		{
			name:      "missing required variable",
			content:   "Hello ${name} from ${location}!",
			variables: map[string]string{"name": "John"},
			wantError: true,
		},
		{
			name:      "variable with default not provided - should not error",
			content:   "Hello ${name:-World}!",
			variables: map[string]string{},
			wantError: false,
		},
		{
			name:      "mixed required and optional variables",
			content:   "Hello ${name} from ${location:-Earth}!",
			variables: map[string]string{"name": "John"},
			wantError: false,
		},
		{
			name:      "mixed variables with missing required",
			content:   "Hello ${name} from ${location:-Earth}!",
			variables: map[string]string{},
			wantError: true,
		},
		{
			name:      "all variables have defaults",
			content:   "Hello ${name:-World} from ${location:-Earth}!",
			variables: map[string]string{},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateVariables(tt.content, tt.variables)
			if (err != nil) != tt.wantError {
				t.Errorf("validateVariables() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestSubstituteVariables(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		variables map[string]string
		expected  string
	}{
		{
			name:      "simple substitution",
			content:   "Hello ${name}!",
			variables: map[string]string{"name": "John"},
			expected:  "Hello John!",
		},
		{
			name:      "substitution with default - value provided",
			content:   "Hello ${name:-World}!",
			variables: map[string]string{"name": "John"},
			expected:  "Hello John!",
		},
		{
			name:      "substitution with default - value not provided",
			content:   "Hello ${name:-World}!",
			variables: map[string]string{},
			expected:  "Hello World!",
		},
		{
			name:      "multiple variables mixed",
			content:   "Hello ${name:-World} from ${location}!",
			variables: map[string]string{"location": "NYC"},
			expected:  "Hello World from NYC!",
		},
		{
			name:      "empty default value",
			content:   "Hello ${name:-}!",
			variables: map[string]string{},
			expected:  "Hello !",
		},
		{
			name:      "complex default values",
			content:   "Path: ${path:-/tmp/default} URL: ${url:-https://example.com}",
			variables: map[string]string{},
			expected:  "Path: /tmp/default URL: https://example.com",
		},
		{
			name:      "variable not found and no default",
			content:   "Hello ${name}!",
			variables: map[string]string{},
			expected:  "Hello ${name}!",
		},
		{
			name:      "default with spaces",
			content:   "Message: ${msg:-Hello World}",
			variables: map[string]string{},
			expected:  "Message: Hello World",
		},
		{
			name:      "override default with provided value",
			content:   "Message: ${msg:-Hello World}",
			variables: map[string]string{"msg": "Custom Message"},
			expected:  "Message: Custom Message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := substituteVariables(tt.content, tt.variables)
			if result != tt.expected {
				t.Errorf("substituteVariables() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestBackwardCompatibility(t *testing.T) {
	// Test that existing scripts without default syntax continue to work
	content := `---
model: "anthropic:claude-sonnet-4-20250514"
---
Hello ${name}! Please analyze ${directory}.`

	variables := map[string]string{
		"name":      "John",
		"directory": "/tmp",
	}

	// Should not error during validation
	err := validateVariables(content, variables)
	if err != nil {
		t.Errorf("validateVariables() should not error for backward compatibility, got: %v", err)
	}

	// Should substitute correctly
	result := substituteVariables(content, variables)
	expected := `---
model: "anthropic:claude-sonnet-4-20250514"
---
Hello John! Please analyze /tmp.`

	if result != expected {
		t.Errorf("substituteVariables() backward compatibility failed.\nGot:\n%s\nWant:\n%s", result, expected)
	}
}