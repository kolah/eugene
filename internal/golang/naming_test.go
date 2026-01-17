package golang

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPascalCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello_world", "HelloWorld"},
		{"hello-world", "HelloWorld"},
		{"hello world", "HelloWorld"},
		{"helloWorld", "HelloWorld"},
		{"HelloWorld", "HelloWorld"},
		{"api_key", "APIKey"},
		{"user_id", "UserID"},
		{"http_url", "HTTPURL"},
		{"json_data", "JSONData"},
		{"html_content", "HTMLContent"},
		{"uuid", "UUID"},
		{"pet_store", "PetStore"},
		{"get_pets_by_id", "GetPetsByID"},
		{"list_api_keys", "ListAPIKeys"},
		{"", ""},
		{"a", "A"},
		{"A", "A"},
		{"abc", "Abc"},
		{"ABC", "Abc"},
		{"petId", "PetID"},
		{"userId", "UserID"},
		{"CVV", "CVV"},
		{"card_cvv", "CardCVV"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := PascalCase(tt.input)
			require.Equal(t, tt.expected, got)
		})
	}
}

func TestCamelCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello_world", "helloWorld"},
		{"hello-world", "helloWorld"},
		{"HelloWorld", "helloWorld"},
		{"api_key", "apiKey"},
		{"user_id", "userID"},
		{"http_url", "httpURL"},
		{"json_data", "jsonData"},
		{"", ""},
		{"a", "a"},
		{"A", "a"},
		{"petId", "petID"},
		{"UserId", "userID"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := CamelCase(tt.input)
			require.Equal(t, tt.expected, got)
		})
	}
}

func TestSnakeCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"HelloWorld", "hello_world"},
		{"helloWorld", "hello_world"},
		{"hello_world", "hello_world"},
		{"APIKey", "apikey"},
		{"userID", "user_id"},
		{"", ""},
		{"a", "a"},
		{"ABC", "abc"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := SnakeCase(tt.input)
			require.Equal(t, tt.expected, got)
		})
	}
}

func TestToGoIdentifier(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello_world", "HelloWorld"},
		{"123abc", "X123abc"},
		{"1", "X1"},
		{"", "X"},
		{"api_key", "APIKey"},
		{"user-name", "UserName"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := ToGoIdentifier(tt.input)
			require.Equal(t, tt.expected, got)
		})
	}
}

func TestEscapeKeyword(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"type", "type_"},
		{"Type", "Type_"},
		{"package", "package_"},
		{"return", "return_"},
		{"name", "name"},
		{"hello", "hello"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := EscapeKeyword(tt.input)
			require.Equal(t, tt.expected, got)
		})
	}
}
