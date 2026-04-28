package parser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtractActiveRole(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"[host] device", "host"},
		{"[source] sink", "source"},
		{"host [device]", "device"},
		{"[sink]", "sink"},
		{"no brackets", "no brackets"},
		{"  [host] device  ", "host"},
		{"", ""},
	}
	for _, tt := range tests {
		got := extractActiveRole(tt.input)
		if got != tt.want {
			t.Errorf("extractActiveRole(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestReadFile(t *testing.T) {
	dir := t.TempDir()

	path := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(path, []byte("  hello\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if got := readFile(path); got != "hello" {
		t.Errorf("readFile() = %q, want %q", got, "hello")
	}

	if got := readFile(filepath.Join(dir, "nonexistent")); got != "" {
		t.Errorf("readFile(nonexistent) = %q, want empty string", got)
	}
}

func TestParseMilliValue(t *testing.T) {
	tests := []struct {
		input   string
		want    int
		wantErr bool
	}{
		{"5000mV", 5000, false},
		{"3000mA", 3000, false},
		{"  900mA  ", 900, false},
		{"12000mV", 12000, false},
		{"0mV", 0, false},
		{"abc", 0, true},
		{"", 0, true},
	}
	for _, tt := range tests {
		got, err := parseMilliValue(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("parseMilliValue(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			continue
		}
		if !tt.wantErr && got != tt.want {
			t.Errorf("parseMilliValue(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}
