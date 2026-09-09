package service

import "testing"

func TestMaxEstimatedTimeToRegainAccess(t *testing.T) {
	tests := []struct {
		name     string
		usage    *WhatsAppBusinessUseCaseUsage
		expected int
	}{
		{name: "nil usage", expected: 0},
		{name: "empty usage", usage: &WhatsAppBusinessUseCaseUsage{}, expected: 0},
		{
			name: "single entry",
			usage: &WhatsAppBusinessUseCaseUsage{
				"waba-1": {{EstimatedTimeToRegainAccess: 30}},
			},
			expected: 30,
		},
		{
			name: "maximum across entries",
			usage: &WhatsAppBusinessUseCaseUsage{
				"waba-1": {{EstimatedTimeToRegainAccess: 30}, {EstimatedTimeToRegainAccess: 120}},
				"waba-2": {{EstimatedTimeToRegainAccess: 60}},
			},
			expected: 120,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if actual := maxEstimatedTimeToRegainAccess(test.usage); actual != test.expected {
				t.Fatalf("got %d, want %d", actual, test.expected)
			}
		})
	}
}
