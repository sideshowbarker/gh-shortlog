# Contributing to gh-shortlog

Read this document if you’re interested in contributing to `gh-shortlog` development.

## Development setup

1. Clone the repository
2. Ensure you have Go 1.21+ installed
3. Run `go build` to build the binary
4. Run `go test ./...` to run tests

## Implementation details

This section explains enough the architecture to help give you a start toward understanding and modifying the code.

### Overview

`gh-shortlog` is an interactive wrapper around `git shortlog` that uses [`fzf`](https://github.com/junegunn/fzf) for the UI. It's written in Go and communicates with `fzf` via subprocess stdin/stdout.

### Main components

These are some details about the core components of the application.

#### Entry Point: `main()`

The `main()` function handles:

- Argument parsing (`--help`, `--version`, `--no-mouse`)
- Dispatching to internal subcommands (`_preview`, `_diffs`, `_browser`, `_help`)
- Launching the interactive mode via `runInteractive()`

#### Interactive Loop: `runInteractive()`

The core UI uses a **loop-based architecture** — rather than nested fzf instances:

```
┌─────────────────────────────────────────────┐
│                                             │
│  ┌─────────────────────────────────────┐    │
│  │  Generate shortlog for current date │    │
│  └─────────────────────────────────────┘    │
│                    │                        │
│                    ▼                        │
│  ┌─────────────────────────────────────┐    │
│  │  Launch fzf, wait for user action   │    │
│  └─────────────────────────────────────┘    │
│                    │                        │
│                    ▼                        │
│  ┌─────────────────────────────────────┐    │
│  │  Parse action from fzf output       │    │
│  │  (ctrl-o, enter, back, quit, etc.)  │    │
│  └─────────────────────────────────────┘    │
│                    │                        │
│         ┌─────────┴─────────┐               │
│         ▼                   ▼               │
│   [date filter]        [back/quit]          │
│   Push to stack,       Pop from stack       │
│   update date          or exit              │
│         │                   │               │
│         └─────────┬─────────┘               │
│                   │                         │
│                   ▼                         │
│            [continue loop]                  │
│                                             │
└─────────────────────────────────────────────┘
```

**State management**: A `dateStack []string` tracks navigation history. When the user applies a date filter, the current date is pushed onto the stack. When they go back (^C/Esc/^Q), it's popped. At the root level, back/quit exits the program.

#### fzf integration: `launchFzf()`

The `launchFzf()` function:

1. Builds `fzf` arguments (colors, layout, key bindings)
2. Sets up environment variables for subcommands
3. Launches `fzf` with shortlog data on stdin
4. Parses `fzf`'s output, to determine user action

**Key fzf options used**:

- `--expect`: Captures specific keys (ctrl-o, ctrl-c, ctrl-q, esc, enter) so we can handle them in Go
- `--print-query`: Outputs the query text (used as date-filter input)
- `--multi`: Enables multi-select with ^T
- `--preview`: Shows commit details via `_preview` subcommand

**Output parsing**: With `--print-query` and `--expect`, fzf outputs:
```
line 1: query text
line 2: key pressed (from --expect list, empty if not in list)
line 3+: selected items
```

#### Subcommands

Internal subcommands are invoked by `fzf` bindings. They receive context via environment variables:

| Subcommand | Purpose | Invoked by |
|------------|---------|------------|
| `_preview` | Show commit log (no diffs) in preview pane | `fzf --preview` |
| `_diffs` | Show commit log with diffs (full screen) | Tab key binding |
| `_browser` | Open GitHub commits page | ^W key binding |
| `_help` | Display keybindings help | ? key binding |

**Environment variables** passed to subcommands:

- `GH_SHORTLOG_DIR`: Working directory for git commands
- `GH_SHORTLOG_ARGS`: Git arguments (joined with `\x1f` separator)
- `GH_SHORTLOG_DATE_FILE`: Temp file containing current date filter
- `GH_SHORTLOG_BASE_URL`: GitHub commit URL base (e.g., `https://github.com/org/repo/commit`)
- `GH_SHORTLOG_ORG_REPO`: GitHub org/repo (e.g., `org/repo`)
- `GH_SHORTLOG_HELP_STATE`: Temp file for help toggle state

#### Key bindings

Key bindings are configured via `fzf`'s `--bind` option:

| Key | Action | Implementation |
|-----|--------|----------------|
| Tab | Show diffs | `execute()` runs `_diffs` subcommand |
| Enter | Date filter | In `--expect`, handled in Go |
| ^T | Toggle select | `fzf`'s built-in `toggle` action |
| ^W | Open browser | `execute()` runs `_browser` subcommand |
| ? | Toggle help | `execute-silent()` + `refresh-preview` |
| ^C/Esc | Back/exit | In `--expect`, handled in Go |
| ^Q | Quit with output | In `--expect`, handled in Go |

### Data flow

```
git shortlog → formatShortlogOutput() → fzf stdin
                                            │
                                            ▼
                                    [user interacts]
                                            │
                                            ▼
                                    fzf stdout → parse
                                            │
                              ┌─────────────┴─────────────┐
                              ▼                           ▼
                    [action in Go]              [action in fzf]
                    (date filter,               (Tab→diffs,
                     back, quit)                 ^W→browser)
```

### Testing

Tests are in `main_test.go`. Key test areas:

- `TestShellQuote`: Shell argument quoting
- `TestFormatShortlogOutput`: Output formatting
- `TestFindGitRoot`: Git repository detection
- `TestParseArgs*`: Argument parsing
- `TestHelpTextKeyBindings`: Help text documents all bindings

Run tests with:
```bash
go test -v ./...
```

### Adding new features

**New key binding**:

1. Add to `--expect` if it needs Go-side handling, or use `--bind` for fzf-side handling
2. Update key parsing in `launchFzf()` if using `--expect`
3. Handle the action in `runInteractive()` if needed
4. Update `helpText` constant and CLI help in `printHelp()`
5. Add test case to `TestHelpTextKeyBindings`

**New subcommand**:

1. Add case in `main()` switch
2. Create `run<Name>Subcommand()` function
3. Add environment variables if needed
4. Create fzf binding that invokes it

## Code style

- Follow standard Go formatting (`go fmt`)
- Keep functions focused and reasonably sized
- Add comments for non-obvious logic
- Use descriptive variable names

## Submitting changes

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run `go test ./...` to ensure tests pass
5. Submit a pull request
