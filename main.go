package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/hmarr/codeowners"
	"gopkg.in/yaml.v3"
)

type config struct {
	Directories []dirSpec     `yaml:"directories"`
	Exclude     []excludeSpec `yaml:"exclude"`
}

type dirSpec struct {
	Path  string `yaml:"path"`
	Level int    `yaml:"level"`
	Match string `yaml:"match"`
}

type excludeSpec struct {
	Path   string `yaml:"path"`
	Reason string `yaml:"reason"`
	Owner  string `yaml:"owner"`
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

	actualConfigPath := configPath
	if actualConfigPath == "" {
		actualConfigPath = ".requirecodeowners.yml"
	}

	printExclusions(cfg.Exclude)

	errors := validate(cfg.Directories, cfg.Exclude, ruleset, actualConfigPath)
	if len(errors) > 0 {
		printErrors(errors)
		os.Exit(1)
	}

	fmt.Println("✓ all directories have CODEOWNERS coverage")
}

func printExclusions(excludes []excludeSpec) {
	if len(excludes) == 0 {
		return
	}
	fmt.Fprintf(os.Stderr, "ℹ %d %s opted out:\n", len(excludes), pluralize(len(excludes), "path", "paths"))
	for _, e := range excludes {
		fmt.Fprintf(os.Stderr, "  - %s (%s) — %s\n", e.Path, e.Owner, e.Reason)
		if dirs := resolvedExclusionDirs(e.Path); len(dirs) > 0 {
			fmt.Fprintf(os.Stderr, "      ↳ %s\n", strings.Join(dirs, ", "))
		}
	}
}

func resolvedExclusionDirs(pattern string) []string {
	dirs, err := expandPath(pattern)
	if err != nil || len(dirs) == 0 {
		return nil
	}
	if len(dirs) == 1 && filepath.Clean(dirs[0]) == filepath.Clean(pattern) {
		return nil
	}
	sort.Strings(dirs)
	return dirs
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
		if d.Match != "" && d.Match != "exact" && d.Match != "coverage" {
			return nil, fmt.Errorf("directory %s has invalid match %q (must be \"exact\" or \"coverage\")", d.Path, d.Match)
		}
	}

	for i, e := range cfg.Exclude {
		if e.Path == "" {
			return nil, fmt.Errorf("exclude at index %d has no path", i)
		}
		if strings.TrimSpace(e.Reason) == "" {
			return nil, fmt.Errorf("exclude %s has no reason (a reason is required to document why the opt-out exists)", e.Path)
		}
		if strings.TrimSpace(e.Owner) == "" {
			return nil, fmt.Errorf("exclude %s has no owner (an owner is required to attribute the opt-out)", e.Path)
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

func expandPath(pattern string) ([]string, error) {
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid pattern: %w", err)
	}

	// Filter to only directories
	var dirs []string
	for _, match := range matches {
		info, err := os.Stat(match)
		if err != nil {
			continue
		}
		if info.IsDir() {
			dirs = append(dirs, match)
		}
	}
	return dirs, nil
}

func validate(specs []dirSpec, excludes []excludeSpec, ruleset codeowners.Ruleset, configPath string) []validationError {
	var errors []validationError

	excluded, exErrs := buildExcluder(excludes, configPath)
	errors = append(errors, exErrs...)

	for _, spec := range specs {
		matchedDirs, err := expandPath(spec.Path)
		if err != nil {
			errors = append(errors, validationError{
				path:    spec.Path,
				message: fmt.Sprintf("Invalid path pattern: %v", err),
			})
			continue
		}
		if len(matchedDirs) == 0 {
			errors = append(errors, validationError{
				path:    spec.Path,
				message: fmt.Sprintf("No directories match this path. Check %s.", configPath),
			})
			continue
		}

		matchMode := spec.Match
		if matchMode == "" {
			matchMode = "exact"
		}

		for _, dir := range matchedDirs {
			errs := validateDirectory(dir, spec.Level, matchMode, ruleset, configPath, excluded)
			errors = append(errors, errs...)
		}
	}

	return errors
}

func buildExcluder(excludes []excludeSpec, configPath string) (func(string) bool, []validationError) {
	var errors []validationError
	var roots []string

	for _, e := range excludes {
		dirs, err := expandPath(e.Path)
		if err != nil {
			errors = append(errors, validationError{
				path:    e.Path,
				message: fmt.Sprintf("Invalid exclude pattern: %v", err),
			})
			continue
		}
		if len(dirs) == 0 {
			errors = append(errors, validationError{
				path:    e.Path,
				message: fmt.Sprintf("Exclude matches no directories. Remove it from %s.", configPath),
			})
			continue
		}
		for _, d := range dirs {
			roots = append(roots, filepath.Clean(d))
		}
	}

	excluded := func(dir string) bool {
		dir = filepath.Clean(dir)
		for _, r := range roots {
			rel, err := filepath.Rel(r, dir)
			if err != nil {
				continue
			}
			if rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
				return true
			}
		}
		return false
	}

	return excluded, errors
}

func validateDirectory(path string, level int, matchMode string, ruleset codeowners.Ruleset, configPath string, excluded func(string) bool) []validationError {
	var errors []validationError

	if excluded(path) {
		return errors
	}

	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		errors = append(errors, validationError{
			path:    path,
			message: fmt.Sprintf("Directory not found. Create it or remove from %s.", configPath),
		})
		return errors
	}
	if err != nil {
		errors = append(errors, validationError{path: path, message: fmt.Sprintf("Cannot access: %v", err)})
		return errors
	}
	if !info.IsDir() {
		errors = append(errors, validationError{
			path:    path,
			message: fmt.Sprintf("Expected a directory but found a file. Check %s.", configPath),
		})
		return errors
	}

	dirsToCheck, err := getDirsAtLevel(path, level)
	if err != nil {
		errors = append(errors, validationError{path: path, message: fmt.Sprintf("Cannot read: %v", err)})
		return errors
	}

	if level > 0 && len(dirsToCheck) == 0 {
		errors = append(errors, validationError{
			path:    path,
			message: fmt.Sprintf("No subdirectories found at level %d. Add subdirectories or set level to 0 in %s.", level, configPath),
		})
		return errors
	}

	for _, d := range dirsToCheck {
		if excluded(d) {
			continue
		}
		if matchMode == "coverage" {
			if !hasCodeownersCoverage(ruleset, d) {
				errors = append(errors, validationError{
					path:    d,
					message: fmt.Sprintf("Not covered by CODEOWNERS. Add: /%s/ @your-team", d),
				})
			}
		} else {
			if !hasExactCodeownersCoverage(ruleset, d) {
				msg := fmt.Sprintf("Not covered by CODEOWNERS. Add: /%s/ @your-team", d)
				if hasCodeownersCoverage(ruleset, d) {
					msg = fmt.Sprintf("Covered by parent CODEOWNERS rule but missing exact entry. Add: /%s/ @your-team", d)
				}
				errors = append(errors, validationError{
					path:    d,
					message: msg,
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

func hasExactCodeownersCoverage(ruleset codeowners.Ruleset, dir string) bool {
	dir = filepath.Clean(dir)

	rule, _ := ruleset.Match(dir + "/file.txt")
	if rule == nil || len(rule.Owners) == 0 {
		return false
	}

	// If the parent matches the same rule pattern, this dir is only covered by inheritance
	parent := filepath.Dir(dir)
	parentRule, _ := ruleset.Match(parent + "/file.txt")
	if parentRule != nil && rule.RawPattern() == parentRule.RawPattern() {
		return false
	}

	return true
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
