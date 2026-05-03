package springboot

import (
	"testing"
)

func TestMatchRoute(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		pattern  string
		field    string
		expected bool
	}{
		{
			name:     "exact match",
			pattern:  "spring.application.name",
			field:    "spring.application.name",
			expected: true,
		},
		{
			name:     "wildcard suffix",
			pattern:  "spring.datasource.*",
			field:    "spring.datasource.url",
			expected: true,
		},
		{
			name:     "wildcard misses other prefix",
			pattern:  "spring.datasource.*",
			field:    "spring.jpa.show-sql",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := matchRoute(tt.pattern, tt.field); got != tt.expected {
				t.Fatalf("matchRoute(%q, %q) = %v, want %v", tt.pattern, tt.field, got, tt.expected)
			}
		})
	}
}
