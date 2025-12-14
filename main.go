package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/hmarr/codeowners"
	"gopkg.in/yaml.v3"
)

type config struct {
	Directories []dirSpec `yaml:"directories"`
}

type dirSpec struct {
	Path  string `yaml:"path"`
	Level int    `yaml:"level"`
}

func main() {
	var configPath string
	var codeownersPath string

	flag.StringVar(&configPath, "config", "", "path to config file (default: .requirecodeowners.yml)")
	flag.StringVar(&codeownersPath, "codeowners-path", "", "path to CODEOWNERS file (auto-detected if not specified)")
	flag.Parse()

	cfg, err := loadConfig(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if len(cfg.Directories) == 0 {
		fmt.Fprintln(os.Stderr, "error: no directories configured")
		os.Exit(1)
	}

	ruleset, err := loadCodeowners(codeownersPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	errors := validate(cfg.Directories, ruleset)
	if len(errors) > 0 {
		fmt.Fprintln(os.Stderr, "validation failed:")
		for _, e := range errors {
			fmt.Fprintf(os.Stderr, "  - %s\n", e)
		}
		os.Exit(1)
	}

	fmt.Println("all directories have CODEOWNERS entries")
}

func loadConfig(path string) (*config, error) {
	if path == "" {
		path = ".requirecodeowners.yml"
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file %s: %w", path, err)
	}

	var cfg config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}

	// Validate config
	for i, d := range cfg.Directories {
		if d.Path == "" {
			return nil, fmt.Errorf("directory at index %d has no path", i)
		}
		if d.Level < 0 {
			return nil, fmt.Errorf("directory %s has invalid level %d (must be >= 0)", d.Path, d.Level)
		}
	}

	return &cfg, nil
}

func loadCodeowners(path string) (codeowners.Ruleset, error) {
	if path != "" {
		return parseCodeownersFile(path)
	}

	locations := []string{
		".github/CODEOWNERS",
		"CODEOWNERS",
		"docs/CODEOWNERS",
	}
	for _, loc := range locations {
		if _, err := os.Stat(loc); err == nil {
			return parseCodeownersFile(loc)
		}
	}
	return nil, fmt.Errorf("CODEOWNERS not found in standard locations (.github/, root, docs/)")
}

func parseCodeownersFile(path string) (codeowners.Ruleset, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()
	return codeowners.ParseFile(f)
}

func validate(specs []dirSpec, ruleset codeowners.Ruleset) []string {
	var errors []string

	for _, spec := range specs {
		info, err := os.Stat(spec.Path)
		if os.IsNotExist(err) {
			errors = append(errors, fmt.Sprintf("directory does not exist: %s", spec.Path))
			continue
		}
		if err != nil {
			errors = append(errors, fmt.Sprintf("checking directory %s: %v", spec.Path, err))
			continue
		}
		if !info.IsDir() {
			errors = append(errors, fmt.Sprintf("not a directory: %s", spec.Path))
			continue
		}

		dirsToCheck, err := getDirsAtLevel(spec.Path, spec.Level)
		if err != nil {
			errors = append(errors, fmt.Sprintf("reading directory %s: %v", spec.Path, err))
			continue
		}

		if spec.Level > 0 && len(dirsToCheck) == 0 {
			errors = append(errors, fmt.Sprintf("no subdirectories found at level %d in: %s", spec.Level, spec.Path))
			continue
		}

		for _, d := range dirsToCheck {
			if !hasCodeownersCoverage(ruleset, d) {
				errors = append(errors, fmt.Sprintf("no CODEOWNERS entry covers: %s", d))
			}
		}
	}

	return errors
}

func getDirsAtLevel(dir string, level int) ([]string, error) {
	if level == 0 {
		return []string{dir}, nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading directory: %w", err)
	}

	var results []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		subdirs, err := getDirsAtLevel(filepath.Join(dir, entry.Name()), level-1)
		if err != nil {
			return nil, err
		}
		results = append(results, subdirs...)
	}
	return results, nil
}

func hasCodeownersCoverage(ruleset codeowners.Ruleset, dir string) bool {
	dir = filepath.Clean(dir)

	testPaths := []string{
		dir,
		dir + "/",
		dir + "/file.txt",
	}

	for _, path := range testPaths {
		rule, _ := ruleset.Match(path)
		if rule != nil && len(rule.Owners) > 0 {
			return true
		}
	}
	return false
}
