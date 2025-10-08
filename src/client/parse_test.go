package client

import (
	"reflect"
	"testing"
)

func TestParseJSONObjects(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		wantObjects   []string
		wantRemaining string
		wantErr       bool
	}{
		{
			name:          "Single complete object",
			input:         `{"key": "value"}`,
			wantObjects:   []string{`{"key": "value"}`},
			wantRemaining: "",
			wantErr:       false,
		},
		{
			name:          "Multiple complete objects",
			input:         `{"key1": "value1"} {"key2": "value2"}`,
			wantObjects:   []string{`{"key1": "value1"}`, `{"key2": "value2"}`},
			wantRemaining: "",
			wantErr:       false,
		},
		{
			name:          "Complete object with whitespace",
			input:         "  \n  {\"key\": \"value\"}  \t  ",
			wantObjects:   []string{`{"key": "value"}`},
			wantRemaining: "",
			wantErr:       false,
		},
		{
			name:          "Complete object followed by incomplete",
			input:         `{"key1": "value1"} {"key2": "val`,
			wantObjects:   []string{`{"key1": "value1"}`},
			wantRemaining: `{"key2": "val`,
			wantErr:       true, // Expecting error because decoder will fail on incomplete JSON
		},
		{
			name:          "Incomplete object only",
			input:         `{"key": "val`,
			wantObjects:   []string{},
			wantRemaining: `{"key": "val`,
			wantErr:       true,
		},
		{
			name:          "Object with nested braces",
			input:         `{"outer": {"inner": "value"}}`,
			wantObjects:   []string{`{"outer": {"inner": "value"}}`},
			wantRemaining: "",
			wantErr:       false,
		},
		{
			name:          "Object with braces in string",
			input:         `{"key": "value with { and } inside"}`,
			wantObjects:   []string{`{"key": "value with { and } inside"}`},
			wantRemaining: "",
			wantErr:       false,
		},
		{
			name:          "Object with escaped quotes",
			input:         `{"key": "value with \"quoted\" text"}`,
			wantObjects:   []string{`{"key": "value with \"quoted\" text"}`},
			wantRemaining: "",
			wantErr:       false,
		},
		{
			name:          "Multiple objects with garbage in between",
			input:         `{"key1": "value1"} garbage {"key2": "value2"}`,
			wantObjects:   []string{`{"key1": "value1"}`},
			wantRemaining: `garbage {"key2": "value2"}`,
			wantErr:       true, // Expecting error because decoder will fail on the garbage
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotObjects, gotRemaining, err := parseJSONObjects(tt.input)

			// Check error status
			if (err != nil) != tt.wantErr {
				t.Errorf("parseJSONObjects() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// For expected errors, we don't validate outputs further
			if tt.wantErr {
				return
			}

			// Check parsed objects
			if !reflect.DeepEqual(gotObjects, tt.wantObjects) {
				t.Errorf("parseJSONObjects() gotObjects = %v, want %v", gotObjects, tt.wantObjects)
			}

			// Check remaining string
			if gotRemaining != tt.wantRemaining {
				t.Errorf("parseJSONObjects() gotRemaining = %q, want %q", gotRemaining, tt.wantRemaining)
			}
		})
	}
}

func TestParseJSONObjects_WhitespaceVariants(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		wantObjects   []string
		wantRemaining string
		wantErr       bool
	}{
		{
			name:          "Object with leading and trailing newlines",
			input:         "\n\n{\"a\":1}\n\n",
			wantObjects:   []string{`{"a":1}`},
			wantRemaining: "",
			wantErr:       false,
		},
		{
			name:          "Object with mixed whitespace",
			input:         "\r\n\t {\"b\":2} \r\n\t",
			wantObjects:   []string{`{"b":2}`},
			wantRemaining: "",
			wantErr:       false,
		},
		{
			name:          "Multiple objects separated by newlines",
			input:         "{\"a\":1}\n{\"b\":2}\r\n{\"c\":3}",
			wantObjects:   []string{`{"a":1}`, `{"b":2}`, `{"c":3}`},
			wantRemaining: "",
			wantErr:       false,
		},
		{
			name:          "Object with only whitespace",
			input:         "   \n\t\r  ",
			wantObjects:   []string{},
			wantRemaining: "",
			wantErr:       false,
		},
		{
			name:          "Incomplete object with whitespace",
			input:         "  {\"a\": ",
			wantObjects:   []string{},
			wantRemaining: "{\"a\":",
			wantErr:       true,
		},
		{
			name:          "Object with whitespace and garbage after",
			input:         " \n {\"a\":1} \r\n garbage",
			wantObjects:   []string{`{"a":1}`},
			wantRemaining: "garbage",
			wantErr:       true, // Should be true to be consistent with the other test cases
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotObjects, gotRemaining, err := parseJSONObjects(tt.input)

			if (err != nil) != tt.wantErr {
				t.Errorf("parseJSONObjects() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if !reflect.DeepEqual(gotObjects, tt.wantObjects) {
				t.Errorf("parseJSONObjects() gotObjects = %v, want %v", gotObjects, tt.wantObjects)
			}
			if gotRemaining != tt.wantRemaining {
				t.Errorf("parseJSONObjects() gotRemaining = %q, want %q", gotRemaining, tt.wantRemaining)
			}
		})
	}
}
