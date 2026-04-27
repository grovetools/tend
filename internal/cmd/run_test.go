package cmd

import (
	"testing"

	"github.com/grovetools/tend/pkg/harness"
)

func TestTagsIntersect(t *testing.T) {
	tests := []struct {
		name string
		a, b []string
		want bool
	}{
		{"both empty", nil, nil, false},
		{"a empty", nil, []string{"slow"}, false},
		{"b empty", []string{"slow"}, nil, false},
		{"match", []string{"slow"}, []string{"slow", "e2e"}, true},
		{"no match", []string{"fast"}, []string{"slow", "e2e"}, false},
		{"multiple match", []string{"fast", "slow"}, []string{"slow"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tagsIntersect(tt.a, tt.b); got != tt.want {
				t.Errorf("tagsIntersect(%v, %v) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestFilterScenarios(t *testing.T) {
	scenarios := []*harness.Scenario{
		{Name: "fast-unit", Tags: []string{"fast", "unit"}},
		{Name: "slow-e2e", Tags: []string{"slow", "e2e"}},
		{Name: "slow-integration", Tags: []string{"slow", "integration"}},
		{Name: "untagged"},
	}

	tests := []struct {
		name      string
		names     []string
		tags      []string
		skip      []string
		run       []string
		wantNames []string
	}{
		{
			name:      "no filters returns all",
			wantNames: []string{"fast-unit", "slow-e2e", "slow-integration", "untagged"},
		},
		{
			name:      "filter by name",
			names:     []string{"slow-*"},
			wantNames: []string{"slow-e2e", "slow-integration"},
		},
		{
			name:      "filter by tags (include)",
			tags:      []string{"e2e"},
			wantNames: []string{"slow-e2e"},
		},
		{
			name:      "skip-tags removes matching",
			skip:      []string{"slow"},
			wantNames: []string{"fast-unit", "untagged"},
		},
		{
			name:      "run-tags keeps only matching",
			run:       []string{"slow"},
			wantNames: []string{"slow-e2e", "slow-integration"},
		},
		{
			name:      "skip-tags and run-tags combined",
			run:       []string{"slow"},
			skip:      []string{"e2e"},
			wantNames: []string{"slow-integration"},
		},
		{
			name:      "tags and skip-tags combined",
			tags:      []string{"slow"},
			skip:      []string{"integration"},
			wantNames: []string{"slow-e2e"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filterScenarios(scenarios, tt.names, tt.tags, tt.skip, tt.run)
			if len(got) != len(tt.wantNames) {
				t.Fatalf("got %d scenarios, want %d", len(got), len(tt.wantNames))
			}
			for i, s := range got {
				if s.Name != tt.wantNames[i] {
					t.Errorf("scenario[%d] = %q, want %q", i, s.Name, tt.wantNames[i])
				}
			}
		})
	}
}
