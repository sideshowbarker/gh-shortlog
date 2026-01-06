package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestShellQuote(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// Simple strings that don't need quoting
		{"hello", "hello"},
		{"path/to/file", "path/to/file"},
		{"gh-shortlog", "gh-shortlog"},

		// Strings with spaces need quoting
		{"hello world", "'hello world'"},
		{"path with spaces", "'path with spaces'"},

		// Strings with single quotes need special handling
		{"it's", "'it'\"'\"'s'"},
		{"don't stop", "'don'\"'\"'t stop'"},

		// Strings with other special chars
		{"tab\there", "'tab\there'"},
		{"new\nline", "'new\nline'"},
		{`back\slash`, `'back\slash'`},
		{`"quoted"`, `'"quoted"'`},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := shellQuote(tt.input)
			if result != tt.expected {
				t.Errorf("shellQuote(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestFormatShortlogOutput(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantRows int
	}{
		{
			name:     "empty input",
			input:    "",
			wantRows: 0,
		},
		{
			name:     "whitespace only",
			input:    "   \n  \n   ",
			wantRows: 0,
		},
		{
			name:     "single author",
			input:    "   100\tJohn Doe <john@example.com>",
			wantRows: 1,
		},
		{
			name: "multiple authors",
			input: `   100	John Doe <john@example.com>
    50	Jane Smith <jane@example.com>
    25	Bob Wilson <bob@example.com>`,
			wantRows: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatShortlogOutput(tt.input)
			if tt.wantRows == 0 {
				if result != "" {
					t.Errorf("expected empty output, got %q", result)
				}
				return
			}

			lines := strings.Split(strings.TrimSpace(result), "\n")
			if len(lines) != tt.wantRows {
				t.Errorf("expected %d rows, got %d", tt.wantRows, len(lines))
			}

			// Check that each line has expected structure (line number at start)
			for i, line := range lines {
				// Line should start with line number (right-aligned in 4 chars)
				lineNum := i + 1
				if !strings.Contains(line, string(rune('0'+lineNum))) {
					t.Errorf("line %d doesn't contain line number", i)
				}
			}
		})
	}
}

func TestFormatShortlogOutputAlignment(t *testing.T) {
	// Test that columns are properly aligned
	input := `  1000	Very Long Name Here <long@example.com>
     1	A <a@b.com>`

	result := formatShortlogOutput(input)
	lines := strings.Split(strings.TrimSpace(result), "\n")

	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}

	// Both lines should have same structure with aligned columns
	// The count column should be right-aligned
	// The name column should be left-aligned (padded to same width)

	// Check that ANSI codes are present (colorization)
	if !strings.Contains(result, "\033[") {
		t.Error("expected ANSI color codes in output")
	}
}

func TestFindGitRoot(t *testing.T) {
	// Create a temporary directory structure
	tmpDir, err := os.MkdirTemp("", "gh-shortlog-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create structure: tmpDir/repo/.git and tmpDir/repo/src/pkg
	repoDir := filepath.Join(tmpDir, "repo")
	gitDir := filepath.Join(repoDir, ".git")
	srcDir := filepath.Join(repoDir, "src")
	pkgDir := filepath.Join(srcDir, "pkg")

	for _, dir := range []string{gitDir, pkgDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("failed to create dir %s: %v", dir, err)
		}
	}

	tests := []struct {
		name     string
		startDir string
		expected string
	}{
		{
			name:     "at repo root",
			startDir: repoDir,
			expected: repoDir,
		},
		{
			name:     "in subdirectory",
			startDir: srcDir,
			expected: repoDir,
		},
		{
			name:     "in nested subdirectory",
			startDir: pkgDir,
			expected: repoDir,
		},
		{
			name:     "outside any repo",
			startDir: tmpDir,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findGitRoot(tt.startDir)
			if result != tt.expected {
				t.Errorf("findGitRoot(%q) = %q, want %q", tt.startDir, result, tt.expected)
			}
		})
	}
}

func TestParseGitHubURL(t *testing.T) {
	// Test the regex used in setupGitHubInfo
	re := regexp.MustCompile(`(?:@|//)([^/:]+)[:/]([^/]+)/([^/]+?)(?:\.git)?$`)

	tests := []struct {
		name        string
		url         string
		wantHost    string
		wantOrg     string
		wantRepo    string
		shouldMatch bool
	}{
		{
			name:        "SSH URL",
			url:         "git@github.com:sideshowbarker/gh-shortlog.git",
			wantHost:    "github.com",
			wantOrg:     "sideshowbarker",
			wantRepo:    "gh-shortlog",
			shouldMatch: true,
		},
		{
			name:        "SSH URL without .git",
			url:         "git@github.com:user/repo",
			wantHost:    "github.com",
			wantOrg:     "user",
			wantRepo:    "repo",
			shouldMatch: true,
		},
		{
			name:        "HTTPS URL",
			url:         "https://github.com/cli/cli.git",
			wantHost:    "github.com",
			wantOrg:     "cli",
			wantRepo:    "cli",
			shouldMatch: true,
		},
		{
			name:        "HTTPS URL without .git",
			url:         "https://github.com/org/repo",
			wantHost:    "github.com",
			wantOrg:     "org",
			wantRepo:    "repo",
			shouldMatch: true,
		},
		{
			name:        "GitHub Enterprise",
			url:         "git@github.mycompany.com:team/project.git",
			wantHost:    "github.mycompany.com",
			wantOrg:     "team",
			wantRepo:    "project",
			shouldMatch: true,
		},
		{
			name:        "Invalid URL",
			url:         "not-a-url",
			shouldMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := re.FindStringSubmatch(tt.url)
			if tt.shouldMatch {
				if len(matches) < 4 {
					t.Fatalf("expected match for %q, got none", tt.url)
				}
				if matches[1] != tt.wantHost {
					t.Errorf("host = %q, want %q", matches[1], tt.wantHost)
				}
				if matches[2] != tt.wantOrg {
					t.Errorf("org = %q, want %q", matches[2], tt.wantOrg)
				}
				if matches[3] != tt.wantRepo {
					t.Errorf("repo = %q, want %q", matches[3], tt.wantRepo)
				}
			} else {
				if len(matches) > 0 {
					t.Errorf("expected no match for %q, got %v", tt.url, matches)
				}
			}
		})
	}
}

func TestParseArgsNoMouse(t *testing.T) {
	// Save and restore global state
	oldNoMouse := noMouse
	oldGitArgs := gitArgs
	oldWorkDir := workDir
	defer func() {
		noMouse = oldNoMouse
		gitArgs = oldGitArgs
		workDir = oldWorkDir
	}()

	// Reset state
	noMouse = false
	gitArgs = nil
	workDir = ""

	parseArgs([]string{"--no-mouse", "--since=1 month ago"})

	if !noMouse {
		t.Error("expected noMouse to be true")
	}
	if len(gitArgs) != 1 || gitArgs[0] != "--since=1 month ago" {
		t.Errorf("expected gitArgs to be [--since=1 month ago], got %v", gitArgs)
	}
}

func TestParseArgsPassThrough(t *testing.T) {
	// Save and restore global state
	oldGitArgs := gitArgs
	oldWorkDir := workDir
	oldNoMouse := noMouse
	defer func() {
		gitArgs = oldGitArgs
		workDir = oldWorkDir
		noMouse = oldNoMouse
	}()

	tests := []struct {
		name     string
		args     []string
		wantArgs []string
	}{
		{
			name:     "revision range",
			args:     []string{"HEAD~10..HEAD"},
			wantArgs: []string{"HEAD~10..HEAD"},
		},
		{
			name:     "since flag",
			args:     []string{"--since=2024-01-01"},
			wantArgs: []string{"--since=2024-01-01"},
		},
		{
			name:     "multiple flags",
			args:     []string{"-n", "10", "--since=1 week ago"},
			wantArgs: []string{"-n", "10", "--since=1 week ago"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gitArgs = nil
			workDir = ""
			noMouse = false

			parseArgs(tt.args)

			if len(gitArgs) != len(tt.wantArgs) {
				t.Errorf("gitArgs length = %d, want %d", len(gitArgs), len(tt.wantArgs))
				return
			}
			for i, arg := range gitArgs {
				if arg != tt.wantArgs[i] {
					t.Errorf("gitArgs[%d] = %q, want %q", i, arg, tt.wantArgs[i])
				}
			}
		})
	}
}

func TestParseArgsDirectoryHandling(t *testing.T) {
	// Create a temporary git repo structure
	tmpDir, err := os.MkdirTemp("", "gh-shortlog-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create: tmpDir/repo/.git and tmpDir/repo/src/pkg
	repoDir := filepath.Join(tmpDir, "repo")
	gitDir := filepath.Join(repoDir, ".git")
	srcDir := filepath.Join(repoDir, "src")
	pkgDir := filepath.Join(srcDir, "pkg")

	for _, dir := range []string{gitDir, pkgDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("failed to create dir %s: %v", dir, err)
		}
	}

	// Save and restore global state
	oldGitArgs := gitArgs
	oldWorkDir := workDir
	oldNoMouse := noMouse
	defer func() {
		gitArgs = oldGitArgs
		workDir = oldWorkDir
		noMouse = oldNoMouse
	}()

	tests := []struct {
		name        string
		args        []string
		wantWorkDir string
		wantArgs    []string
	}{
		{
			name:        "repo root as first arg",
			args:        []string{repoDir},
			wantWorkDir: repoDir,
			wantArgs:    []string{},
		},
		{
			name:        "subdirectory as first arg",
			args:        []string{srcDir},
			wantWorkDir: repoDir,
			wantArgs:    []string{"--", "src"},
		},
		{
			name:        "nested subdirectory as first arg",
			args:        []string{pkgDir},
			wantWorkDir: repoDir,
			wantArgs:    []string{"--", "src/pkg"},
		},
		{
			name:        "repo root with additional args",
			args:        []string{repoDir, "--since=1 week ago"},
			wantWorkDir: repoDir,
			wantArgs:    []string{"--since=1 week ago"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gitArgs = nil
			workDir = ""
			noMouse = false

			parseArgs(tt.args)

			if workDir != tt.wantWorkDir {
				t.Errorf("workDir = %q, want %q", workDir, tt.wantWorkDir)
			}
			if len(gitArgs) != len(tt.wantArgs) {
				t.Errorf("gitArgs = %v (len %d), want %v (len %d)", gitArgs, len(gitArgs), tt.wantArgs, len(tt.wantArgs))
				return
			}
			for i, arg := range gitArgs {
				if arg != tt.wantArgs[i] {
					t.Errorf("gitArgs[%d] = %q, want %q", i, arg, tt.wantArgs[i])
				}
			}
		})
	}
}

func TestParseArgsDoubleDash(t *testing.T) {
	// Create a temporary git repo structure
	tmpDir, err := os.MkdirTemp("", "gh-shortlog-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	repoDir := filepath.Join(tmpDir, "repo")
	gitDir := filepath.Join(repoDir, ".git")
	srcDir := filepath.Join(repoDir, "src")

	for _, dir := range []string{gitDir, srcDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("failed to create dir %s: %v", dir, err)
		}
	}

	// Save and restore global state
	oldGitArgs := gitArgs
	oldWorkDir := workDir
	oldNoMouse := noMouse
	defer func() {
		gitArgs = oldGitArgs
		workDir = oldWorkDir
		noMouse = oldNoMouse
	}()

	tests := []struct {
		name        string
		args        []string
		wantWorkDir string
		wantArgs    []string
	}{
		{
			name:        "double dash with directory path",
			args:        []string{"--", srcDir},
			wantWorkDir: repoDir,
			wantArgs:    []string{"--", "src"},
		},
		{
			name:        "flags before double dash",
			args:        []string{"--since=1 week ago", "--", srcDir},
			wantWorkDir: repoDir,
			wantArgs:    []string{"--since=1 week ago", "--", "src"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gitArgs = nil
			workDir = ""
			noMouse = false

			parseArgs(tt.args)

			if workDir != tt.wantWorkDir {
				t.Errorf("workDir = %q, want %q", workDir, tt.wantWorkDir)
			}
			if len(gitArgs) != len(tt.wantArgs) {
				t.Errorf("gitArgs = %v (len %d), want %v (len %d)", gitArgs, len(gitArgs), tt.wantArgs, len(tt.wantArgs))
				return
			}
			for i, arg := range gitArgs {
				if arg != tt.wantArgs[i] {
					t.Errorf("gitArgs[%d] = %q, want %q", i, arg, tt.wantArgs[i])
				}
			}
		})
	}
}

func TestHelpTextKeyBindings(t *testing.T) {
	// Verify help text documents the expected key bindings
	expectedBindings := []string{
		"Tab",    // Show diffs
		"Enter",  // Date filter
		"^T",     // Toggle multi-select
		"^W",     // Open browser
		"^Q",     // Exit with output
		"^C/Esc", // Exit
		"^F/^B",  // Scroll preview
	}

	for _, binding := range expectedBindings {
		if !strings.Contains(helpText, binding) {
			t.Errorf("helpText missing expected key binding: %s", binding)
		}
	}

	// Verify specific descriptions
	if !strings.Contains(helpText, "Toggle multi-select") {
		t.Error("helpText missing ^T description")
	}
	if !strings.Contains(helpText, "Filter by date") {
		t.Error("helpText missing Enter description")
	}
	if !strings.Contains(helpText, "Show commits with diffs") {
		t.Error("helpText missing Tab description")
	}
}

func TestGenerateShortlogHeadInsertion(t *testing.T) {
	// This tests the logic that inserts HEAD when no revision is specified
	// We can't easily test generateShortlog directly (requires git),
	// but we can test the arg-building logic by extracting it

	tests := []struct {
		name            string
		gitArgsInput    []string
		wantHasRevision bool
		wantDDIdx       int
	}{
		{
			name:            "empty args",
			gitArgsInput:    []string{},
			wantHasRevision: false,
			wantDDIdx:       -1,
		},
		{
			name:            "only flags",
			gitArgsInput:    []string{"--since=1 week ago"},
			wantHasRevision: false,
			wantDDIdx:       -1,
		},
		{
			name:            "with revision",
			gitArgsInput:    []string{"HEAD~10..HEAD"},
			wantHasRevision: true,
			wantDDIdx:       -1,
		},
		{
			name:            "path after double dash",
			gitArgsInput:    []string{"--", "src/"},
			wantHasRevision: false,
			wantDDIdx:       0,
		},
		{
			name:            "flags then double dash then path",
			gitArgsInput:    []string{"--since=1 week ago", "--", "src/"},
			wantHasRevision: false,
			wantDDIdx:       1,
		},
		{
			name:            "revision before double dash",
			gitArgsInput:    []string{"origin..HEAD", "--", "src/"},
			wantHasRevision: true,
			wantDDIdx:       1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Replicate the logic from generateShortlog
			hasRevision := false
			doubleDashIndex := -1
			for i, arg := range tt.gitArgsInput {
				if arg == "--" {
					doubleDashIndex = i
					break
				}
				if !strings.HasPrefix(arg, "-") {
					hasRevision = true
				}
			}

			if hasRevision != tt.wantHasRevision {
				t.Errorf("hasRevision = %v, want %v", hasRevision, tt.wantHasRevision)
			}
			if doubleDashIndex != tt.wantDDIdx {
				t.Errorf("doubleDashIndex = %d, want %d", doubleDashIndex, tt.wantDDIdx)
			}
		})
	}
}
