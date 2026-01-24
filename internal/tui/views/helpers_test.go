package views

import (
	"testing"
)

func TestTruncate(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{"no truncation needed", "hello", 10, "hello"},
		{"exact length", "hello", 5, "hello"},
		{"truncate with ellipsis", "hello world", 8, "hello..."},
		{"very short max", "hello", 3, "hel"},
		{"max 2", "hello", 2, "he"},
		{"max 1", "hello", 1, "h"},
		{"empty string", "", 5, ""},
		{"zero max", "hello", 0, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncate(tt.input, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
			}
		})
	}
}

func TestTruncateEllipsis(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{"no truncation needed", "hello", 10, "hello"},
		{"exact length", "hello", 5, "hello"},
		{"truncate with unicode ellipsis", "hello world", 8, "hello w…"},
		{"max 1", "hello", 1, "h"},
		{"empty string", "", 5, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateEllipsis(tt.input, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncateEllipsis(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
			}
		})
	}
}

func TestMinInt(t *testing.T) {
	tests := []struct {
		a, b int
		want int
	}{
		{1, 2, 1},
		{2, 1, 1},
		{5, 5, 5},
		{-1, 1, -1},
		{0, 0, 0},
	}

	for _, tt := range tests {
		got := minInt(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("minInt(%d, %d) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestMaxInt(t *testing.T) {
	tests := []struct {
		a, b int
		want int
	}{
		{1, 2, 2},
		{2, 1, 2},
		{5, 5, 5},
		{-1, 1, 1},
		{0, 0, 0},
	}

	for _, tt := range tests {
		got := maxInt(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("maxInt(%d, %d) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestFormatNumberWithCommas(t *testing.T) {
	tests := []struct {
		input int64
		want  string
	}{
		{0, "0"},
		{1, "1"},
		{12, "12"},
		{123, "123"},
		{1234, "1,234"},
		{12345, "12,345"},
		{123456, "123,456"},
		{1234567, "1,234,567"},
		{12345678, "12,345,678"},
		{123456789, "123,456,789"},
		{1234567890, "1,234,567,890"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := formatNumberWithCommas(tt.input)
			if got != tt.want {
				t.Errorf("formatNumberWithCommas(%d) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		input int64
		want  string
	}{
		{0, "0 B"},
		{1, "1 B"},
		{512, "512 B"},
		{1023, "1023 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1572864, "1.5 MB"},
		{1073741824, "1.0 GB"},
		{1610612736, "1.5 GB"},
		{1099511627776, "1.0 TB"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := formatBytes(tt.input)
			if got != tt.want {
				t.Errorf("formatBytes(%d) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestFormatPackets(t *testing.T) {
	tests := []struct {
		input int64
		want  string
	}{
		{0, "0"},
		{1, "1"},
		{999, "999"},
		{1000, "1.0K"},
		{1500, "1.5K"},
		{999999, "1000.0K"},
		{1000000, "1.0M"},
		{1500000, "1.5M"},
		{999999999, "1000.0M"},
		{1000000000, "1.0B"},
		{1500000000, "1.5B"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := formatPackets(tt.input)
			if got != tt.want {
				t.Errorf("formatPackets(%d) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestCleanValue(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", ""},
		{"hello", "hello"},
		{"N/A", ""},
		{"n/a", ""},
		{"ukn", ""},
		{"UKN", ""},
		{"[n/a]", ""},
		{"[N/A]", ""},
		{"unknown", ""},
		{"Unknown", ""},
		{"UNKNOWN", ""},
		{"  n/a  ", ""},
		{"valid value", "valid value"},
		{"  valid  ", "  valid  "},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := cleanValue(tt.input)
			if got != tt.want {
				t.Errorf("cleanValue(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
