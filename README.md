# An interactive git shortlog (GitHub CLI extension TUI)

[fzf](https://github.com/junegunn/fzf#installation)-built [GitHub CLI](https://github.com/cli/cli) extension [TUI](https://en.wikipedia.org/wiki/Text-based_user_interface) `git shortlog` frontend — to interactively explore contributor data for any repo

![image](https://github.com/user-attachments/assets/76772163-b8e6-4fc0-9c75-6df68b6b59f4)

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
gh shortlog
```

- Type a date into the prompt and then do `Ctrl-O`: `gh-shortlog` will change to showing a log/history for only those changes made after your specified date.
- Type a name or e-mail address into the prompt: `gh-shortlog` will dynamically filter the list of authors down to just those who match what you typed into the prompt.

| Key    | Action                                                        |
| ------ | ------------------------------------------------------------- |
| `Ctrl‑O` | Filter the log down to use the commits made after the date entered into the prompt.|
| `Enter`  | Open a new window with a diffs-included log of all commits by the author at the current pointer. |
| `Ctrl‑C` | Exit `gh-shortlog`, or else go back to the previous screen withing `gh-shortlog`. |
| `Ctrl‑F` | Scroll the preview window one page forward. |
| `Ctrl‑B` | Scroll the preview window one page back. |
| `Ctrl‑J` | Scroll the pointer/selection down. |
| `Ctrl‑K` | Scroll the pointer/selection up. |
| `Ctrl‑N` | Scroll the pointer/selection down. |
| `Ctrl‑P` | Scroll the pointer/selection up. |
| `Ctrl‑U` | Clear the prompt. |

You can also use your mouse: clicking moves the selection; double-clicking has the same effect as the `Enter` key.
