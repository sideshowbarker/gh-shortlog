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
gh shortlog [--no-mouse]
```

- Type a date into the prompt and then do `Ctrl-O`: then, `gh-shortlog` will change to showing a log/history for only those changes made after your specified date.
- Type a name or e-mail address into the prompt: then, `gh-shortlog` will dynamically filter the list of authors down to just those who match what you typed into the prompt.

| Key    | Action                                                        |
| ------ | ------------------------------------------------------------- |
| `Ctrl‑O`     | Filter the log down to use the commits made after the date entered into the prompt.|
| `Enter`      | Show a diffs-included log of all commits by the author at the current pointer. |
| `Ctrl‑W`     | Open GitHub log in a web browser |
| `Ctrl‑C`     | Exit `gh-shortlog`, or else go back to the previous screen within `gh-shortlog`. |
| `Ctrl‑F`     | Scroll the preview window one page forward. |
| `Ctrl‑B`     | Scroll the preview window one page back. |
| `Ctrl‑J`     | Scroll the pointer in the main window to the next name down. |
| `Ctrl‑K`     | Scroll the pointer in the main window to the next name up. |
| `Ctrl‑N`     | Scroll the pointer in the main window to the next name down. |
| `Ctrl‑P`     | Scroll the pointer in the main window to the next name up. |
| `Ctrl‑U`     | Clear the prompt. |
| `tab`        | Toggle selection of the item (author name) at the pointer and move to the next name down. |
| `shift‑tab`  | Toggle selection of the item (author name) at the pointer and move to the next name up. |
| `Ctrl‑Q`     | Exit (or go back one screen) — and then, on final program exit, output the list of items selected. |

You can also use your mouse: click in main window moves the selection; double-click has the same effect as the `Enter` key; mouse scroll in either main window or preview window scrolls the window contents.

If you don’t want that mouse behavior, use the `--no-mouse` option.
