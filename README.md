# An interactive git shortlog (GitHub CLI extension TUI)

[fzf](https://github.com/junegunn/fzf#installation)-built [GitHub CLI](https://github.com/cli/cli) extension [TUI](https://en.wikipedia.org/wiki/Text-based_user_interface) `git shortlog` frontend — to interactively explore contributor data for any repo

<img alt="screenshot of gh-shortlog" src="https://github.com/user-attachments/assets/413c5c74-d143-4bfc-b564-4d842b83123c" />

## Install

Make sure you have the [GitHub CLI](https://github.com/cli/cli#installation) (the `gh` command) and [fzf](https://github.com/junegunn/fzf#installation) installed. Then:

```sh
# install
gh ext install sideshowbarker/gh-shortlog
# upgrade
gh ext upgrade sideshowbarker/gh-shortlog
# uninstall
gh ext remove sideshowbarker/gh-shortlog
```

## Usage

```
gh shortlog [options] [<revision-range>] [[--] <path>...]
```

All options supported by `git shortlog` and `git log` are passed through. Examples:

```sh
gh shortlog                           # Full history of current repo
gh shortlog ~/other-repo              # Different repository (directory path)
gh shortlog --since="1 month ago"     # Recent commits only
gh shortlog origin..HEAD              # Commits not yet pushed
gh shortlog -- src/                   # Only changes in src/
gh shortlog HEAD~100..HEAD -- "*.go"  # Last 100 commits touching Go files
gh shortlog --no-mouse                # Disable mouse support in fzf
```

- Type a date into the prompt and then press `Enter`: then, `gh-shortlog` will change to showing a log/history for only those changes made after your specified date.
- Type a name or e-mail address into the prompt: then, `gh-shortlog` will dynamically filter the list of authors down to just those who match what you typed into the prompt.

| Key          | Action                                                                              |
| ------------ | ----------------------------------------------------------------------------------- |
| `Enter`      | Filter the log to show only commits made after the date entered into the prompt.    |
| `Tab`        | Show a diffs-included log of all commits by the selected author(s).                 |
| `Ctrl‑T`     | Toggle selection of the item (author name) at the pointer.                          |
| `Ctrl‑W`     | Open GitHub log in a web browser.                                                   |
| `Ctrl‑C`     | Exit `gh-shortlog`, or go back to the previous screen within `gh-shortlog`.         |
| `Esc`        | Exit `gh-shortlog`, or go back to the previous screen within `gh-shortlog`.         |
| `Ctrl‑Q`     | Exit (or go back one screen) — and on final exit, output the list of items selected.|
| `?`          | Toggle keybindings help in the preview pane.                                        |
| `Ctrl‑F`     | Scroll the preview window one page forward.                                         |
| `Ctrl‑B`     | Scroll the preview window one page back.                                            |
| `Ctrl‑J`     | Move the pointer in the main window to the next name down.                          |
| `Ctrl‑K`     | Move the pointer in the main window to the next name up.                            |
| `Ctrl‑N`     | Move the pointer in the main window to the next name down.                          |
| `Ctrl‑P`     | Move the pointer in the main window to the next name up.                            |
| `Ctrl‑U`     | Clear the prompt.                                                                   |

You can also use your mouse: click in main window moves the selection; double-click has the same effect as the `Enter` key; mouse scroll in either main window or preview window scrolls the window contents.

If you don't want that mouse behavior, use the `--no-mouse` option.

## Building from source

Requires Go 1.21 or later:

```sh
git clone https://github.com/sideshowbarker/gh-shortlog
cd gh-shortlog
go build -o gh-shortlog .
```

Then copy the `gh-shortlog` binary to somewhere in your PATH, or install as a GitHub CLI extension:

```sh
gh ext install .
```
