package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

const (
	version = "2.0.0"

	// ANSI color codes
	colorReset     = "\033[0m"
	colorGreen     = "\033[1;32m"
	colorWhite     = "\033[1;37m"
	colorCyan      = "\033[0;36m"
	colorBoldCyan  = "\033[1;36m"
	colorYellow    = "\033[1;33m"

	// Help text shown in preview when ? is pressed
	helpText = `
` + "\033[1;36m" + `KEYBINDINGS` + "\033[0m" + `

` + "\033[1;33m" + `Navigation` + "\033[0m" + `
  ↑/↓, ^J/^K, ^N/^P Move cursor up/down
  ^F/^B             Scroll preview page down/up
  ^T                Toggle multi-select for current item

` + "\033[1;33m" + `Actions` + "\033[0m" + `
  Tab               Show commits with diffs for selected author(s)
  Enter             Filter by date (type a date first, then Enter)
  ^W                Open author's commits in GitHub browser

` + "\033[1;33m" + `Other` + "\033[0m" + `
  ?                 Toggle this help
  ^Q                Exit and output selected items
  ^C/Esc            Exit

` + "\033[1;33m" + `Tips` + "\033[0m" + `
  • Type to filter authors by name or email
  • Use ^T to select multiple authors, then Tab to view their diffs
  • Type a date (e.g., "2024-01-01" or "3 months ago") then Enter to filter

` + "\033[0;36m" + `Press ? again to return to commit preview` + "\033[0m" + `

This is an fzf-based application. If you like it, consider
sponsoring fzf's creator, ` + "\033[1;33m" + `Junegunn Choi` + "\033[0m" + `:

    ` + "\033[0;36m" + `https://github.com/sponsors/junegunn` + "\033[0m" + `
`
)

// Global state
var (
	gitArgs     []string // Arguments to pass to git
	workDir     string   // Working directory for git commands
	noMouse     bool     // Disable mouse in fzf
	dateFile    string   // Temp file for storing date filter
	baseURL     string   // GitHub commit URL base
	orgAndRepo  string   // GitHub org/repo
	selfPath    string   // Path to this executable
)

func main() {
	// Get path to self for fzf callbacks
	var err error
	selfPath, err = os.Executable()
	if err != nil {
		selfPath = os.Args[0]
	}

	// Parse arguments
	args := os.Args[1:]
	if len(args) > 0 {
		switch args[0] {
		case "--help", "-h":
			printHelp()
			return
		case "--version", "-v":
			fmt.Println("gh-shortlog", version)
			return
		case "_preview":
			// Internal: preview for fzf
			runPreviewSubcommand(args[1:])
			return
		case "_diffs":
			// Internal: show diffs
			runDiffsSubcommand(args[1:])
			return
		case "_browser":
			// Internal: open in browser
			runBrowserSubcommand(args[1:])
			return
		case "_help":
			// Internal: show help in preview
			fmt.Print(helpText)
			return
		}
	}

	// Main interactive mode
	parseArgs(args)
	setup()
	runInteractive()
}

func printHelp() {
	fmt.Println(`gh-shortlog - Interactive git shortlog explorer

Usage: gh-shortlog [options] [<revision-range>] [[--] <path>...]

Options:
  --no-mouse    Disable mouse support in fzf
  --help, -h    Show this help message
  --version     Show version

All other options are passed directly to git shortlog/log.
See 'git shortlog --help' for available options.

Examples:
  gh shortlog                           # Full history
  gh shortlog ~/other-repo              # Different repository
  gh shortlog --since="1 month ago"     # Recent commits
  gh shortlog origin..HEAD              # Commits not yet pushed
  gh shortlog -- src/                   # Only changes in src/

Interactive keys (press ? in the UI for full help):
  ?          Show/hide keybindings help in preview
  Tab        View commits with diffs for selected author(s)
  Enter      Filter by date (type date first, then Enter)
  Ctrl-T     Toggle multi-select for current author
  Ctrl-W     Open author's commits in GitHub
  Ctrl-Q     Exit and output selected items
  Ctrl-C     Exit`)
}

func parseArgs(args []string) {
	// Check for environment variables (used by subcommands)
	if envArgs := os.Getenv("GH_SHORTLOG_ARGS"); envArgs != "" {
		gitArgs = strings.Split(envArgs, "\x1f") // Unit separator
	}
	if envDir := os.Getenv("GH_SHORTLOG_DIR"); envDir != "" {
		workDir = envDir
	}
	if envDate := os.Getenv("GH_SHORTLOG_DATE_FILE"); envDate != "" {
		dateFile = envDate
	}
	if envBase := os.Getenv("GH_SHORTLOG_BASE_URL"); envBase != "" {
		baseURL = envBase
	}
	if envOrg := os.Getenv("GH_SHORTLOG_ORG_REPO"); envOrg != "" {
		orgAndRepo = envOrg
	}

	// Parse command line args
	var remaining []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--no-mouse":
			noMouse = true
		case arg == "--":
			// Everything after -- is a path
			// Check if the first path after -- needs workDir resolution
			if i+1 < len(args) && workDir == "" {
				pathArg := args[i+1]
				if info, err := os.Stat(pathArg); err == nil && info.IsDir() {
					absPath, _ := filepath.Abs(pathArg)
					repoRoot := findGitRoot(absPath)
					if repoRoot != "" {
						workDir = repoRoot
						// Convert remaining paths to relative
						remaining = append(remaining, "--")
						for _, p := range args[i+1:] {
							if abs, err := filepath.Abs(p); err == nil {
								if rel, err := filepath.Rel(repoRoot, abs); err == nil {
									remaining = append(remaining, rel)
									continue
								}
							}
							remaining = append(remaining, p)
						}
						i = len(args)
						continue
					}
				}
			}
			remaining = append(remaining, args[i:]...)
			i = len(args)
		default:
			// Check if it's a directory path (first non-flag arg)
			if workDir == "" && !strings.HasPrefix(arg, "-") {
				if info, err := os.Stat(arg); err == nil && info.IsDir() {
					absPath, err := filepath.Abs(arg)
					if err == nil {
						// Check if this directory is a git repo root
						gitDir := filepath.Join(absPath, ".git")
						if _, err := os.Stat(gitDir); err == nil {
							workDir = absPath
							continue
						}
						// Check if it's inside a git repo (find repo root)
						repoRoot := findGitRoot(absPath)
						if repoRoot != "" {
							workDir = repoRoot
							// Convert to relative path for filtering
							relPath, err := filepath.Rel(repoRoot, absPath)
							if err == nil {
								remaining = append(remaining, "--", relPath)
								continue
							}
						}
					}
				}
			}
			remaining = append(remaining, arg)
		}
	}

	gitArgs = remaining
}

func setup() {
	// Create temp file for date
	if dateFile == "" {
		tmpFile, err := os.CreateTemp("", "gh-shortlog-date-*")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating temp file: %v\n", err)
			os.Exit(1)
		}
		dateFile = tmpFile.Name()
		tmpFile.Close()
	}

	// Get GitHub info
	if baseURL == "" || orgAndRepo == "" {
		setupGitHubInfo()
	}
}

func setupGitHubInfo() {
	// Check for upstream first, then origin
	remoteURL := ""
	for _, remote := range []string{"upstream", "origin"} {
		cmd := gitCommand("config", "--get", "remote."+remote+".url")
		out, err := cmd.Output()
		if err == nil && len(out) > 0 {
			remoteURL = strings.TrimSpace(string(out))
			break
		}
	}

	if remoteURL == "" {
		return
	}

	// Parse GitHub URL
	// Handles: git@github.com:org/repo.git, https://github.com/org/repo.git
	re := regexp.MustCompile(`(?:@|//)([^/:]+)[:/]([^/]+)/([^/]+?)(?:\.git)?$`)
	matches := re.FindStringSubmatch(remoteURL)
	if len(matches) >= 4 {
		host := matches[1]
		org := matches[2]
		repo := matches[3]
		orgAndRepo = org + "/" + repo
		baseURL = "https://" + host + "/" + orgAndRepo + "/commit"
	}
}

// findGitRoot walks up from dir to find the git repository root
func findGitRoot(dir string) string {
	for {
		gitDir := filepath.Join(dir, ".git")
		if _, err := os.Stat(gitDir); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root
			return ""
		}
		dir = parent
	}
}

func gitCommand(args ...string) *exec.Cmd {
	if workDir != "" {
		args = append([]string{"-C", workDir}, args...)
	}
	return exec.Command("git", args...)
}

func runInteractive() {
	// State stack for back navigation
	// Each entry is a date filter (empty string = full history)
	var dateStack []string
	currentDate := ""

	for {
		// Generate shortlog for current date filter
		shortlogOutput := generateShortlog(currentDate)

		// Launch fzf and get result
		action, query, selections := launchFzf(shortlogOutput, currentDate)

		switch action {
		case "ctrl-o":
			// Push current state and apply new date filter
			if query != "" {
				dateStack = append(dateStack, currentDate)
				currentDate = query
			}
			// Loop continues with new filter

		case "back":
			// Go back to previous state (no output on exit)
			if len(dateStack) > 0 {
				currentDate = dateStack[len(dateStack)-1]
				dateStack = dateStack[:len(dateStack)-1]
				// Loop continues with previous filter
			} else {
				// At root, exit silently
				return
			}

		case "quit":
			// Go back, or exit with selection output
			if len(dateStack) > 0 {
				currentDate = dateStack[len(dateStack)-1]
				dateStack = dateStack[:len(dateStack)-1]
				// Loop continues with previous filter
			} else {
				// At root, output selections and exit
				for _, sel := range selections {
					fmt.Println(sel)
				}
				return
			}

		default:
			// Exit (accept or any other)
			return
		}
	}
}

func generateShortlog(sinceDate string) string {
	args := []string{"shortlog", "-n", "-s", "-e"}
	if sinceDate != "" {
		args = append(args, "--since="+sinceDate)
	}

	// Check if gitArgs contains a revision (something not starting with - and not after --)
	// and find where to insert HEAD if needed
	hasRevision := false
	doubleDashIndex := -1
	for i, arg := range gitArgs {
		if arg == "--" {
			doubleDashIndex = i
			break
		}
		if !strings.HasPrefix(arg, "-") {
			hasRevision = true
		}
	}

	// Build args with HEAD in correct position (before -- if present)
	if !hasRevision {
		if doubleDashIndex >= 0 {
			// Insert HEAD before --
			args = append(args, gitArgs[:doubleDashIndex]...)
			args = append(args, "HEAD")
			args = append(args, gitArgs[doubleDashIndex:]...)
		} else {
			// No --, just append everything then HEAD
			args = append(args, gitArgs...)
			args = append(args, "HEAD")
		}
	} else {
		args = append(args, gitArgs...)
	}

	cmd := gitCommand(args...)
	out, err := cmd.Output()
	if err != nil {
		return ""
	}

	return formatShortlogOutput(string(out))
}

func formatShortlogOutput(output string) string {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) == 0 || (len(lines) == 1 && lines[0] == "") {
		return ""
	}

	type entry struct {
		count int
		name  string
		email string
	}

	var entries []entry
	maxCount := 0
	maxName := 0

	// Parse each line
	emailRe := regexp.MustCompile(`<[^>]+>$`)
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Format: "  123\tName <email>"
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) != 2 {
			continue
		}

		count, _ := strconv.Atoi(strings.TrimSpace(parts[0]))
		rest := strings.TrimSpace(parts[1])

		email := emailRe.FindString(rest)
		name := strings.TrimSpace(strings.TrimSuffix(rest, email))

		entries = append(entries, entry{count, name, email})

		countLen := len(strconv.Itoa(count))
		if countLen > maxCount {
			maxCount = countLen
		}
		if len(name) > maxName {
			maxName = len(name)
		}
	}

	// Format output with line numbers and alignment
	var result strings.Builder
	for i, e := range entries {
		fmt.Fprintf(&result, "%4d  %s%*d%s  %s%-*s%s  %s%s%s\n",
			i+1,
			colorGreen, maxCount, e.count, colorReset,
			colorWhite, maxName, e.name, colorReset,
			colorCyan, e.email, colorReset)
	}

	return result.String()
}

// launchFzf runs fzf and returns the action taken, query value, and any selections
// action is one of: "ctrl-o", "back", "quit", "accept"
func launchFzf(input string, currentDate string) (action string, query string, selections []string) {
	// Build fzf arguments
	fzfArgs := []string{
		"--ansi",
		"--delimiter", " {2,}",
		"--border", "horizontal",
		"--border=rounded",
		"--layout=reverse",
		"--pointer", "▶",
		"--no-scrollbar",
		"--preview-window=border-line",
		"--multi",
		"--print-query",
		"--expect", "ctrl-o,ctrl-c,ctrl-q,esc,enter", // Capture these keys
		"--color", "fg:15,bg:-1,hl:1",
		"--color", "header:green:italic",
		"--color", "prompt:80,info:40",
		"--color", "border:dim",
	}

	if noMouse {
		fzfArgs = append(fzfArgs, "--no-mouse")
	}

	// Build header (just shows date filter status)
	var header string
	if currentDate != "" {
		header = colorYellow + "Showing commits since " + colorWhite + currentDate + colorReset
	} else {
		header = colorYellow + "Showing full history" + colorReset
	}
	fzfArgs = append(fzfArgs, "--header", header)

	// Prompt with help hint - the help hint appears after the info (counts)
	fzfArgs = append(fzfArgs, "--prompt", "Filter by name/email or date > ")
	fzfArgs = append(fzfArgs, "--info", "inline: │ ? for help │ ")

	// Build environment for subcommands
	env := os.Environ()
	env = append(env, "GH_SHORTLOG_ARGS="+strings.Join(gitArgs, "\x1f"))
	env = append(env, "GH_SHORTLOG_DIR="+workDir)
	env = append(env, "GH_SHORTLOG_DATE_FILE="+dateFile)
	env = append(env, "GH_SHORTLOG_BASE_URL="+baseURL)
	env = append(env, "GH_SHORTLOG_ORG_REPO="+orgAndRepo)

	// Write current date to file for preview/diffs subcommands
	os.WriteFile(dateFile, []byte(currentDate), 0644)

	// Help toggle state file
	helpStateFile, _ := os.CreateTemp("", "gh-shortlog-help-*")
	helpStateFile.Close()
	helpStatePath := helpStateFile.Name()
	env = append(env, "GH_SHORTLOG_HELP_STATE="+helpStatePath)

	// Preview command checks help state file to decide what to show
	previewCmd := fmt.Sprintf("if [ -s $GH_SHORTLOG_HELP_STATE ]; then %s _help; else printf '\\n\\n'; %s _preview {+5}; fi",
		shellQuote(selfPath), shellQuote(selfPath))
	fzfArgs = append(fzfArgs, "--preview", previewCmd)

	// Key bindings
	fzfArgs = append(fzfArgs, "--bind", "ctrl-b:preview-page-up,ctrl-f:preview-page-down")

	// ? toggles help state and refreshes preview
	toggleHelpCmd := fmt.Sprintf("if [ -s %s ]; then : > %s; else echo 1 > %s; fi", helpStatePath, helpStatePath, helpStatePath)
	fzfArgs = append(fzfArgs, "--bind", fmt.Sprintf("?:execute-silent(%s)+refresh-preview", toggleHelpCmd))

	// Tab shows diffs for selected/current author(s)
	fzfArgs = append(fzfArgs, "--bind", fmt.Sprintf("tab:execute(clear; %s _diffs {+5}; printf \"\\nPress any key to go back...\"; read -n 1 -r)", shellQuote(selfPath)))

	// ^T toggles multi-select (replaces default Tab behavior)
	fzfArgs = append(fzfArgs, "--bind", "ctrl-t:toggle")

	// Ctrl-W opens browser (doesn't exit fzf)
	fzfArgs = append(fzfArgs, "--bind", fmt.Sprintf("ctrl-w:execute(%s _browser {5})", shellQuote(selfPath)))

	cmd := exec.Command("fzf", fzfArgs...)
	cmd.Stdin = strings.NewReader(input)
	cmd.Stderr = os.Stderr
	cmd.Env = env

	// Capture stdout to parse the result
	output, _ := cmd.Output()

	// Parse output: first line is query, second line is key pressed, rest is selection
	// Note: fzf may exit with status 1 when using --expect without selection, but output is still valid
	lines := strings.Split(string(output), "\n")

	if len(lines) < 2 {
		// No output - treat as back/exit
		return "back", "", nil
	}

	query = lines[0]
	key := lines[1]

	// Selections are lines 2+ (excluding empty trailing lines)
	for _, line := range lines[2:] {
		if line != "" {
			selections = append(selections, line)
		}
	}

	switch key {
	case "ctrl-o":
		// ^O = date filter shortcut (always uses query as date)
		return "ctrl-o", query, selections
	case "enter":
		// Enter always applies date filter (empty query = full history)
		return "ctrl-o", query, selections
	case "ctrl-c", "esc":
		return "back", query, selections
	case "ctrl-q":
		return "quit", query, selections
	default:
		return "accept", query, selections
	}
}

func shellQuote(s string) string {
	if strings.ContainsAny(s, " \t\n'\"\\") {
		return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
	}
	return s
}

// Subcommand: _preview
func runPreviewSubcommand(args []string) {
	parseArgs(nil) // Load from env

	if len(args) < 1 {
		return
	}

	// Read date from file
	sinceDate := ""
	if data, err := os.ReadFile(dateFile); err == nil {
		sinceDate = strings.TrimSpace(string(data))
	}

	// Build git log command
	// Support multiple authors (from fzf {+5} multi-select)
	logArgs := []string{"log", "--no-patch", "--format=fuller", "--notes", "--color"}
	for _, author := range args {
		logArgs = append(logArgs, "--author="+author)
	}
	if sinceDate != "" {
		logArgs = append(logArgs, "--since="+sinceDate)
	}
	logArgs = append(logArgs, gitArgs...)

	cmd := gitCommand(logArgs...)
	out, err := cmd.Output()
	if err != nil {
		return
	}

	// Replace commit hashes with URLs (handle ANSI codes around commit line)
	output := string(out)
	if baseURL != "" {
		// Match 40-char commit hash, keep first 10 chars and replace with URL
		re := regexp.MustCompile(`(commit )([0-9a-f]{10})([0-9a-f]{30})`)
		output = re.ReplaceAllString(output, "${1}"+baseURL+"/${2}")
	}

	fmt.Print(output)
}

// Subcommand: _diffs
func runDiffsSubcommand(args []string) {
	parseArgs(nil) // Load from env

	if len(args) < 1 {
		return
	}

	// Read date from file
	sinceDate := ""
	if data, err := os.ReadFile(dateFile); err == nil {
		sinceDate = strings.TrimSpace(string(data))
	}

	// Build git log command with diffs
	// Support multiple authors (from fzf {+5} multi-select)
	logArgs := []string{"log", "-w", "--patch-with-stat", "--format=fuller", "--notes", "--color"}
	for _, author := range args {
		logArgs = append(logArgs, "--author="+author)
	}
	if sinceDate != "" {
		logArgs = append(logArgs, "--since="+sinceDate)
	}
	logArgs = append(logArgs, gitArgs...)

	cmd := gitCommand(logArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()
}

// Subcommand: _browser
func runBrowserSubcommand(args []string) {
	parseArgs(nil) // Load from env

	if len(args) < 1 || orgAndRepo == "" {
		return
	}

	email := args[0]
	// Extract author from email: remove <>, handle noreply format
	author := strings.Trim(email, "<>")
	// Handle GitHub noreply: 12345+username@users.noreply.github.com
	re := regexp.MustCompile(`^[^+]+\+([^@]+)@.*$`)
	if matches := re.FindStringSubmatch(author); len(matches) > 1 {
		author = matches[1]
	}

	// Get GitHub login via API
	login := getGitHubLogin(author)
	if login == "" {
		login = author
	}

	// Read date from file
	sinceDate := ""
	if data, err := os.ReadFile(dateFile); err == nil {
		sinceDate = strings.TrimSpace(string(data))
	}

	// Build URL
	url := fmt.Sprintf("https://github.com/%s/commits?author=%s", orgAndRepo, login)
	if sinceDate != "" {
		// Format date for GitHub
		formattedDate := formatDateForGitHub(sinceDate)
		if formattedDate != "" {
			url += "&since=" + formattedDate
		}
	}

	openBrowser(url)
}

func getGitHubLogin(author string) string {
	// Use gh CLI to get login
	cmd := exec.Command("gh", "api", fmt.Sprintf("/repos/%s/commits?author=%s&per_page=1", orgAndRepo, author), "--jq", ".[] | .author.login")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func formatDateForGitHub(date string) string {
	// Try to parse and format the date using GNU date
	var cmd *exec.Cmd
	// Check for gdate (macOS with coreutils)
	if _, err := exec.LookPath("gdate"); err == nil {
		cmd = exec.Command("gdate", "+%Y-%m-%dT%H:%M:%SZ", "--date="+date)
	} else {
		cmd = exec.Command("date", "+%Y-%m-%dT%H:%M:%SZ", "--date="+date)
	}
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		fmt.Fprintf(os.Stderr, "Cannot open browser on %s\n", runtime.GOOS)
		return
	}
	cmd.Run()
}

