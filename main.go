package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hmarr/codeowners"
)

func main() {
	var directories string
	var codeownersPath string
	var level int

	flag.StringVar(&directories, "directories", "", "newline or comma separated list of directories to validate")
	flag.StringVar(&codeownersPath, "codeowners-path", "", "path to CODEOWNERS file (auto-detected if not specified)")
	flag.IntVar(&level, "level", 0, "directory depth to check (0=directory itself, 1=immediate subdirs, etc.)")
	flag.Parse()

	if level < 0 {
		fmt.Fprintln(os.Stderr, "error: --level must be >= 0")
		os.Exit(1)
	}

	if directories == "" {
		fmt.Fprintln(os.Stderr, "error: --directories is required")
		os.Exit(1)
	}

	dirs := parseDirectories(directories)
	if len(dirs) == 0 {
		fmt.Fprintln(os.Stderr, "error: no directories provided")
		os.Exit(1)
	}

	ruleset, err := loadCodeowners(codeownersPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	errors := validate(dirs, ruleset, level)
	if len(errors) > 0 {
		fmt.Fprintln(os.Stderr, "validation failed:")
		for _, e := range errors {
			fmt.Fprintf(os.Stderr, "  - %s\n", e)
		}
		os.Exit(1)
	}

	fmt.Println("all directories have CODEOWNERS entries")
}

func parseDirectories(input string) []string {
	// Handle both newline and comma separated
	input = strings.ReplaceAll(input, ",", "\n")
	lines := strings.Split(input, "\n")

	var dirs []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			dirs = append(dirs, trimmed)
		}
	}
	return dirs
}

func loadCodeowners(path string) (codeowners.Ruleset, error) {
	if path != "" {
		return parseCodeownersFile(path)
	}

	// Try standard locations
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
	defer f.Close()
	return codeowners.ParseFile(f)
}

func validate(dirs []string, ruleset codeowners.Ruleset, level int) []string {
	var errors []string

	for _, dir := range dirs {
		info, err := os.Stat(dir)
		if os.IsNotExist(err) {
			errors = append(errors, fmt.Sprintf("directory does not exist: %s", dir))
			continue
		}
		if err != nil {
			errors = append(errors, fmt.Sprintf("checking directory %s: %v", dir, err))
			continue
		}
		if !info.IsDir() {
			errors = append(errors, fmt.Sprintf("not a directory: %s", dir))
			continue
		}

		// Get directories at the specified level
		dirsToCheck, err := getDirsAtLevel(dir, level)
		if err != nil {
			errors = append(errors, fmt.Sprintf("reading directory %s: %v", dir, err))
			continue
		}

		if level > 0 && len(dirsToCheck) == 0 {
			errors = append(errors, fmt.Sprintf("no subdirectories found at level %d in: %s", level, dir))
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

	// Check if any rule covers this directory
	// Try multiple path variants to handle different pattern styles
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
