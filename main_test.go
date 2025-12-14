package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hmarr/codeowners"
)

func TestLoadConfig(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name    string
		content string
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			content: `directories:
  - path: src
    level: 0
  - path: services
    level: 1
`,
			wantErr: false,
		},
		{
			name: "missing path",
			content: `directories:
  - level: 1
`,
			wantErr: true,
			errMsg:  "has no path",
		},
		{
			name: "negative level",
			content: `directories:
  - path: src
    level: -1
`,
			wantErr: true,
			errMsg:  "invalid level",
		},
		{
			name:    "invalid yaml",
			content: `not: valid: yaml:`,
			wantErr: true,
			errMsg:  "parsing config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configPath := filepath.Join(tmpDir, tt.name+".yml")
			os.WriteFile(configPath, []byte(tt.content), 0644)

			cfg, err := loadConfig(configPath)
			if tt.wantErr {
				if err == nil {
					t.Errorf("loadConfig() expected error containing %q, got nil", tt.errMsg)
				} else if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("loadConfig() error = %v, want error containing %q", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("loadConfig() unexpected error: %v", err)
				}
				if cfg == nil {
					t.Error("loadConfig() returned nil config")
				}
			}
		})
	}
}

func TestLoadConfigDefaultPath(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".requirecodeowners.yml")
	os.WriteFile(configPath, []byte(`directories:
  - path: src
`), 0644)

	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	cfg, err := loadConfig("")
	if err != nil {
		t.Fatalf("loadConfig() error = %v", err)
	}
	if len(cfg.Directories) != 1 {
		t.Errorf("loadConfig() got %d directories, want 1", len(cfg.Directories))
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
	os.WriteFile(filepath.Join(tmpDir, ".github", "CODEOWNERS"), []byte(`/src/ @team-a
`), 0644)

	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	ruleset, err := loadCodeowners("")
	if err != nil {
		t.Fatalf("loading CODEOWNERS: %v", err)
	}

	tests := []struct {
		name     string
		specs    []dirSpec
		wantErrs int
	}{
		{
			name:     "covered directory passes",
			specs:    []dirSpec{{Path: "src", Level: 0}},
			wantErrs: 0,
		},
		{
			name:     "uncovered directory fails",
			specs:    []dirSpec{{Path: "pkg", Level: 0}},
			wantErrs: 1,
		},
		{
			name:     "nonexistent directory fails",
			specs:    []dirSpec{{Path: "nonexistent", Level: 0}},
			wantErrs: 1,
		},
		{
			name:     "file instead of directory fails",
			specs:    []dirSpec{{Path: "file.txt", Level: 0}},
			wantErrs: 1,
		},
		{
			name: "multiple directories mixed results",
			specs: []dirSpec{
				{Path: "src", Level: 0},
				{Path: "pkg", Level: 0},
			},
			wantErrs: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := validate(tt.specs, ruleset, ".requirecodeowners.yml")
			if len(errs) != tt.wantErrs {
				t.Errorf("validate() errors = %v, want %d errors", errs, tt.wantErrs)
			}
		})
	}
}

func TestValidateWithLevel(t *testing.T) {
	tmpDir := t.TempDir()

	// Create directory structure
	os.MkdirAll(filepath.Join(tmpDir, "services", "foo"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, "services", "bar"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, "services", "baz"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, "empty"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, ".github"), 0755)

	// Create CODEOWNERS - only foo and bar have owners
	os.WriteFile(filepath.Join(tmpDir, ".github", "CODEOWNERS"), []byte(`/services/foo/ @team-foo
/services/bar/ @team-bar
`), 0644)

	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	ruleset, err := loadCodeowners("")
	if err != nil {
		t.Fatalf("loading CODEOWNERS: %v", err)
	}

	tests := []struct {
		name     string
		specs    []dirSpec
		wantErrs int
	}{
		{
			name:     "level 0 checks services itself",
			specs:    []dirSpec{{Path: "services", Level: 0}},
			wantErrs: 1, // services/ has no owner
		},
		{
			name:     "level 1 checks subdirs - baz missing",
			specs:    []dirSpec{{Path: "services", Level: 1}},
			wantErrs: 1, // baz has no owner
		},
		{
			name:     "level 1 with no subdirs errors",
			specs:    []dirSpec{{Path: "empty", Level: 1}},
			wantErrs: 1, // no subdirs to check
		},
		{
			name: "mixed levels",
			specs: []dirSpec{
				{Path: "services", Level: 1},
				{Path: "empty", Level: 0},
			},
			wantErrs: 2, // baz missing + empty uncovered
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := validate(tt.specs, ruleset, ".requirecodeowners.yml")
			if len(errs) != tt.wantErrs {
				t.Errorf("validate() errors = %v, want %d errors", errs, tt.wantErrs)
			}
		})
	}
}

func TestValidateWithGlob(t *testing.T) {
	tmpDir := t.TempDir()

	// Create directory structure: apps/a/services/foo, apps/b/services/bar
	os.MkdirAll(filepath.Join(tmpDir, "apps", "a", "services", "foo"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, "apps", "a", "services", "bar"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, "apps", "b", "services", "baz"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, ".github"), 0755)

	// Only foo and bar have owners, baz is missing
	os.WriteFile(filepath.Join(tmpDir, ".github", "CODEOWNERS"), []byte(`/apps/a/services/foo/ @team-foo
/apps/a/services/bar/ @team-bar
`), 0644)

	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	ruleset, err := loadCodeowners("")
	if err != nil {
		t.Fatalf("loading CODEOWNERS: %v", err)
	}

	tests := []struct {
		name     string
		specs    []dirSpec
		wantErrs int
	}{
		{
			name:     "glob matches multiple dirs",
			specs:    []dirSpec{{Path: "apps/*/services", Level: 1}},
			wantErrs: 1, // baz in apps/b/services is missing
		},
		{
			name:     "glob with no matches",
			specs:    []dirSpec{{Path: "nonexistent/*/path", Level: 0}},
			wantErrs: 1, // no directories match
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := validate(tt.specs, ruleset, ".requirecodeowners.yml")
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
