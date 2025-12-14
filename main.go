package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"

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

type validationError struct {
	path    string
	message string
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
		printErrors(errors)
		os.Exit(1)
	}

	fmt.Println("✓ all directories have CODEOWNERS coverage")
}

func printErrors(errors []validationError) {
	// Sort by path for consistent output
	sort.Slice(errors, func(i, j int) bool {
		return errors[i].path < errors[j].path
	})

	// Text output to stderr (for console)
	fmt.Fprintln(os.Stderr)
	for _, e := range errors {
		fmt.Fprintf(os.Stderr, "  ✗ %s\n", e.path)
		fmt.Fprintf(os.Stderr, "    %s\n", e.message)
	}
	fmt.Fprintln(os.Stderr)
	fmt.Fprintf(os.Stderr, "✗ %d %s failed CODEOWNERS check\n", len(errors), pluralize(len(errors), "directory", "directories"))

	// Markdown output to stdout (for GitHub Actions summary)
	fmt.Println("## ❌ CODEOWNERS Check Failed")
	fmt.Println()
	fmt.Println("| Path | Issue |")
	fmt.Println("|------|-------|")
	for _, e := range errors {
		fmt.Printf("| `%s` | %s |\n", e.path, e.message)
	}
	fmt.Println()
	fmt.Printf("**%d %s** need attention.\n", len(errors), pluralize(len(errors), "directory", "directories"))
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

func pluralize(n int, singular, plural string) string {
	if n == 1 {
		return singular
	}
	return plural
}

func validate(specs []dirSpec, ruleset codeowners.Ruleset) []validationError {
	var errors []validationError

	for _, spec := range specs {
		info, err := os.Stat(spec.Path)
		if os.IsNotExist(err) {
			errors = append(errors, validationError{
				path:    spec.Path,
				message: "directory does not exist. Create it or remove from .requirecodeowners.yml",
			})
			continue
		}
		if err != nil {
			errors = append(errors, validationError{path: spec.Path, message: fmt.Sprintf("error: %v", err)})
			continue
		}
		if !info.IsDir() {
			errors = append(errors, validationError{
				path:    spec.Path,
				message: "path is a file, not a directory. Update .requirecodeowners.yml",
			})
			continue
		}

		dirsToCheck, err := getDirsAtLevel(spec.Path, spec.Level)
		if err != nil {
			errors = append(errors, validationError{path: spec.Path, message: fmt.Sprintf("error reading: %v", err)})
			continue
		}

		if spec.Level > 0 && len(dirsToCheck) == 0 {
			errors = append(errors, validationError{
				path:    spec.Path,
				message: fmt.Sprintf("no subdirectories at level %d. Create subdirectories or set level: 0", spec.Level),
			})
			continue
		}

		for _, d := range dirsToCheck {
			if !hasCodeownersCoverage(ruleset, d) {
				errors = append(errors, validationError{
					path:    d,
					message: fmt.Sprintf("missing CODEOWNERS entry. Add to CODEOWNERS: /%s/ @owner", d),
				})
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
