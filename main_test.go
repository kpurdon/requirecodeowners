package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hmarr/codeowners"
)

func TestParseDirectories(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "newline separated",
			input: "src\npkg\ninternal",
			want:  []string{"src", "pkg", "internal"},
		},
		{
			name:  "comma separated",
			input: "src,pkg,internal",
			want:  []string{"src", "pkg", "internal"},
		},
		{
			name:  "mixed with whitespace",
			input: "src\n  pkg  \n\ninternal",
			want:  []string{"src", "pkg", "internal"},
		},
		{
			name:  "empty",
			input: "",
			want:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseDirectories(tt.input)
			if len(got) != len(tt.want) {
				t.Errorf("parseDirectories() = %v, want %v", got, tt.want)
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("parseDirectories()[%d] = %v, want %v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestValidate(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test directories
	os.MkdirAll(filepath.Join(tmpDir, "src"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, "pkg"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, ".github"), 0755)

	// Create a file (not a directory)
	os.WriteFile(filepath.Join(tmpDir, "file.txt"), []byte("test"), 0644)

	// Create CODEOWNERS
	codeownersContent := `/src/ @team-a
`
	os.WriteFile(filepath.Join(tmpDir, ".github", "CODEOWNERS"), []byte(codeownersContent), 0644)

	// Change to temp dir
	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	ruleset, err := loadCodeowners("")
	if err != nil {
		t.Fatalf("loading CODEOWNERS: %v", err)
	}

	tests := []struct {
		name     string
		dirs     []string
		wantErrs int
	}{
		{
			name:     "covered directory passes",
			dirs:     []string{"src"},
			wantErrs: 0,
		},
		{
			name:     "uncovered directory fails",
			dirs:     []string{"pkg"},
			wantErrs: 1,
		},
		{
			name:     "nonexistent directory fails",
			dirs:     []string{"nonexistent"},
			wantErrs: 1,
		},
		{
			name:     "file instead of directory fails",
			dirs:     []string{"file.txt"},
			wantErrs: 1,
		},
		{
			name:     "multiple directories mixed results",
			dirs:     []string{"src", "pkg", "nonexistent"},
			wantErrs: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := validate(tt.dirs, ruleset)
			if len(errs) != tt.wantErrs {
				t.Errorf("validate() errors = %v, want %d errors", errs, tt.wantErrs)
			}
		})
	}
}

func TestLoadCodeowners(t *testing.T) {
	tmpDir := t.TempDir()

	// Test auto-detection from .github/
	os.MkdirAll(filepath.Join(tmpDir, ".github"), 0755)
	os.WriteFile(filepath.Join(tmpDir, ".github", "CODEOWNERS"), []byte("/src/ @team\n"), 0644)

	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	ruleset, err := loadCodeowners("")
	if err != nil {
		t.Fatalf("loadCodeowners() error = %v", err)
	}
	if ruleset == nil {
		t.Error("loadCodeowners() returned nil ruleset")
	}
}

func TestHasCodeownersCoverage(t *testing.T) {
	// Create a ruleset from string
	content := `/src/ @team-a
/pkg/** @team-b
internal/ @team-c
`
	ruleset, err := codeowners.ParseFile(strings.NewReader(content))
	if err != nil {
		t.Fatalf("parsing CODEOWNERS: %v", err)
	}

	tests := []struct {
		name string
		dir  string
		want bool
	}{
		{"exact match with slash", "src", true},
		{"glob pattern", "pkg", true},
		{"unanchored pattern", "internal", true},
		{"uncovered", "other", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hasCodeownersCoverage(ruleset, tt.dir)
			if got != tt.want {
				t.Errorf("hasCodeownersCoverage(%q) = %v, want %v", tt.dir, got, tt.want)
			}
		})
	}
}
