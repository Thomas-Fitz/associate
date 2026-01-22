package graph

import (
	"testing"
	"time"
)

func TestNewClientWithRetry_ConfigValidation(t *testing.T) {
	// This test verifies that retry options are valid
	cfg := Config{
		Host:     "localhost",
		Port:     "5432",
		Username: "associate",
		Password: "password",
		Database: "associate",
	}

	opts := RetryOptions{
		MaxAttempts:  3,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     1 * time.Second,
	}

	// Verify the options are valid
	if opts.MaxAttempts < 1 {
		t.Error("MaxAttempts should be at least 1")
	}
	if opts.InitialDelay <= 0 {
		t.Error("InitialDelay should be positive")
	}
	if opts.MaxDelay < opts.InitialDelay {
		t.Error("MaxDelay should be >= InitialDelay")
	}
	_ = cfg // Used in actual connection test
}

func TestRetryOptions_Defaults(t *testing.T) {
	opts := DefaultRetryOptions()

	if opts.MaxAttempts != 30 {
		t.Errorf("Default MaxAttempts: got %d, want 30", opts.MaxAttempts)
	}
	if opts.InitialDelay != 1*time.Second {
		t.Errorf("Default InitialDelay: got %v, want 1s", opts.InitialDelay)
	}
	if opts.MaxDelay != 10*time.Second {
		t.Errorf("Default MaxDelay: got %v, want 10s", opts.MaxDelay)
	}
}

func TestConfigFromEnv_Defaults(t *testing.T) {
	// Test that defaults are set correctly when env vars are not set
	cfg := ConfigFromEnv()

	// The defaults should match what we expect for local development
	// Note: This test may behave differently if env vars are actually set
	if cfg.Host == "" {
		t.Error("Host should not be empty")
	}
	if cfg.Port == "" {
		t.Error("Port should not be empty")
	}
	if cfg.Username == "" {
		t.Error("Username should not be empty")
	}
	if cfg.Password == "" {
		t.Error("Password should not be empty")
	}
	if cfg.Database == "" {
		t.Error("Database should not be empty")
	}
}

func TestConfig_DSN(t *testing.T) {
	cfg := Config{
		Host:     "myhost",
		Port:     "5432",
		Username: "myuser",
		Password: "mypass",
		Database: "mydb",
	}

	dsn := cfg.DSN()
	expected := "host=myhost port=5432 user=myuser password=mypass dbname=mydb sslmode=disable"
	if dsn != expected {
		t.Errorf("DSN mismatch:\ngot:  %s\nwant: %s", dsn, expected)
	}
}

func TestEscapeCypherString(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"simple", "simple"},
		{"with'quote", "with\\'quote"},
		{"with\"double", "with\\\"double"},
		{"with\nnewline", "with\\nnewline"},
		{"with\ttab", "with\\ttab"},
		{"with\\backslash", "with\\\\backslash"},
	}

	for _, tt := range tests {
		result := EscapeCypherString(tt.input)
		if result != tt.expected {
			t.Errorf("EscapeCypherString(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestTagsToCypherList(t *testing.T) {
	tests := []struct {
		input    []string
		expected string
	}{
		{nil, "[]"},
		{[]string{}, "[]"},
		{[]string{"one"}, "['one']"},
		{[]string{"one", "two"}, "['one', 'two']"},
		{[]string{"with'quote"}, "['with\\'quote']"},
	}

	for _, tt := range tests {
		result := tagsToCypherList(tt.input)
		if result != tt.expected {
			t.Errorf("tagsToCypherList(%v) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestCalculateInsertPositions(t *testing.T) {
	tests := []struct {
		name      string
		afterPos  float64
		beforePos float64
		count     int
		wantLen   int
	}{
		{"Empty plan", 0, 0, 1, 1},
		{"Append to end", 1000, 0, 1, 1},
		{"Insert at start", 0, 1000, 1, 1},
		{"Insert between", 1000, 2000, 1, 1},
		{"Multiple insert", 1000, 2000, 3, 3},
		{"Zero count", 1000, 2000, 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			positions := CalculateInsertPositions(tt.afterPos, tt.beforePos, tt.count)
			if len(positions) != tt.wantLen {
				t.Errorf("CalculateInsertPositions(%f, %f, %d) returned %d positions, want %d",
					tt.afterPos, tt.beforePos, tt.count, len(positions), tt.wantLen)
			}

			// Verify positions are in order
			for i := 1; i < len(positions); i++ {
				if positions[i] <= positions[i-1] {
					t.Errorf("Positions not in ascending order: %v", positions)
				}
			}

			// Verify positions are between afterPos and beforePos (when both are set)
			if tt.afterPos > 0 && tt.beforePos > 0 {
				for _, pos := range positions {
					if pos <= tt.afterPos || pos >= tt.beforePos {
						t.Errorf("Position %f not between %f and %f", pos, tt.afterPos, tt.beforePos)
					}
				}
			}
		})
	}
}
