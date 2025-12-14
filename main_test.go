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
		level    int
		wantErrs int
	}{
		{
			name:     "covered directory passes",
			dirs:     []string{"src"},
			level:    0,
			wantErrs: 0,
		},
		{
			name:     "uncovered directory fails",
			dirs:     []string{"pkg"},
			level:    0,
			wantErrs: 1,
		},
		{
			name:     "nonexistent directory fails",
			dirs:     []string{"nonexistent"},
			level:    0,
			wantErrs: 1,
		},
		{
			name:     "file instead of directory fails",
			dirs:     []string{"file.txt"},
			level:    0,
			wantErrs: 1,
		},
		{
			name:     "multiple directories mixed results",
			dirs:     []string{"src", "pkg", "nonexistent"},
			level:    0,
			wantErrs: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := validate(tt.dirs, ruleset, tt.level)
			if len(errs) != tt.wantErrs {
				t.Errorf("validate() errors = %v, want %d errors", errs, tt.wantErrs)
			}
		})
	}
}

func TestValidateWithLevel(t *testing.T) {
	tmpDir := t.TempDir()

	// Create directory structure:
	// services/
	//   foo/
	//   bar/
	//   baz/
	// empty/ (no subdirs)
	os.MkdirAll(filepath.Join(tmpDir, "services", "foo"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, "services", "bar"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, "services", "baz"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, "empty"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, ".github"), 0755)

	// Create CODEOWNERS - only foo and bar have owners
	codeownersContent := `/services/foo/ @team-foo
/services/bar/ @team-bar
`
	os.WriteFile(filepath.Join(tmpDir, ".github", "CODEOWNERS"), []byte(codeownersContent), 0644)

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
		level    int
		wantErrs int
	}{
		{
			name:     "level 0 checks services itself",
			dirs:     []string{"services"},
			level:    0,
			wantErrs: 1, // services/ has no owner
		},
		{
			name:     "level 1 checks subdirs - baz missing",
			dirs:     []string{"services"},
			level:    1,
			wantErrs: 1, // baz has no owner
		},
		{
			name:     "level 1 with no subdirs errors",
			dirs:     []string{"empty"},
			level:    1,
			wantErrs: 1, // no subdirs to check
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := validate(tt.dirs, ruleset, tt.level)
			if len(errs) != tt.wantErrs {
				t.Errorf("validate() errors = %v, want %d errors", errs, tt.wantErrs)
			}
		})
	}
}

func TestGetDirsAtLevel(t *testing.T) {
	tmpDir := t.TempDir()

	// Create nested structure
	os.MkdirAll(filepath.Join(tmpDir, "a", "b", "c"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, "a", "b", "d"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, "a", "e"), 0755)
	// Create a file to ensure it's ignored
	os.WriteFile(filepath.Join(tmpDir, "a", "file.txt"), []byte("test"), 0644)

	tests := []struct {
		name    string
		dir     string
		level   int
		want    []string
		wantErr bool
	}{
		{
			name:  "level 0 returns dir itself",
			dir:   filepath.Join(tmpDir, "a"),
			level: 0,
			want:  []string{filepath.Join(tmpDir, "a")},
		},
		{
			name:  "level 1 returns immediate subdirs",
			dir:   filepath.Join(tmpDir, "a"),
			level: 1,
			want:  []string{filepath.Join(tmpDir, "a", "b"), filepath.Join(tmpDir, "a", "e")},
		},
		{
			name:  "level 2 returns nested subdirs",
			dir:   filepath.Join(tmpDir, "a"),
			level: 2,
			want:  []string{filepath.Join(tmpDir, "a", "b", "c"), filepath.Join(tmpDir, "a", "b", "d")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getDirsAtLevel(tt.dir, tt.level)
			if (err != nil) != tt.wantErr {
				t.Errorf("getDirsAtLevel() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(got) != len(tt.want) {
				t.Errorf("getDirsAtLevel() = %v, want %v", got, tt.want)
				return
			}
			// Sort both for comparison since directory order may vary
			for _, w := range tt.want {
				found := false
				for _, g := range got {
					if g == w {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("getDirsAtLevel() missing %s, got %v", w, got)
				}
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
