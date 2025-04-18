```markdown
# lazyls

[![Go Report Card](https://goreportcard.com/badge/github.com/AlexandrosLiaskos/lazyls)](https://goreportcard.com/report/github.com/AlexandrosLiaskos/lazyls)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A simple, lazy-loaded terminal file browser with directory stats and Git integration, written in Go using the [`gocui`](https://github.com/jroimartin/gocui) library.

## Features

*   **Dual-Pane Layout:** Separate views for folders and files.
*   **Directory Statistics:** Displays total directory size and identifies the largest file within (calculated asynchronously).
*   **Git Integration:** Shows the current Git branch status for the directory.
*   **Nerd Font Icons:** Uses Nerd Font icons for files and folders based on name/extension.
*   **Hidden File Toggling:** Easily show/hide hidden files (starting with `.`).
*   **File Content Viewer:** View text file content directly within the application.
    *   Line numbers displayed.
    *   Handles large files (up to 20 MiB by default).
    *   Basic binary file detection (prevents viewing binary content).
    *   Tab-to-space conversion for better readability.
*   **Action Menu:** Perform actions on the selected file/folder:
    *   Copy Full Path
    *   Copy Relative Path
    *   View Content (Files only)
    *   Copy Content (Files only, up to 5 MiB limit by default)
*   **Clipboard Integration:** Copies paths or file content to the system clipboard.
*   **Navigation:** Standard Vim-like (`j/k`, `g/G`) and arrow key navigation.
*   **Logging:** Logs activity and errors to `lazyls.log` in the directory where it's run.
*   **Responsive UI:** Layout adjusts to terminal size.

## Requirements

*   **Go:** Version 1.18 or later (for building).
*   **Nerd Font:** Required to display icons correctly. Download and install from [Nerd Fonts](https://www.nerdfonts.com/). Make sure your terminal is configured to use a Nerd Font.
*   **`git` command:** Must be installed and in your system's PATH for Git status integration.
*   **Clipboard tool:** A working clipboard utility (like `xclip`/`xsel` on Linux, `pbcopy`/`pbpaste` on macOS, or standard clipboard on Windows) for the copy actions. The [`atotto/clipboard`](https://github.com/atotto/clipboard) library attempts to handle this automatically.

## Installation

### Using `go install`

```bash
go install github.com/AlexandrosLiaskos/lazyls@latest
```

This will download the source and install the `lazyls` binary in your `$GOPATH/bin` directory. Ensure this directory is in your system's `PATH`.

### Build from Source

1.  Clone the repository:
    ```bash
    git clone https://github.com/AlexandrosLiaskos/lazyls.git
    cd lazyls
    ```
2.  Build the binary:
    ```bash
    go build -o lazyls
    ```
3.  Move the binary to a directory in your `PATH`, for example:
    ```bash
    sudo mv lazyls /usr/local/bin/
    ```

## Usage

Simply run the command in your terminal:

```bash
lazyls
```

It will display the contents of the current working directory.

## Keybindings

| Key(s)         | Context        | Action                                             |
| -------------- | -------------- | -------------------------------------------------- |
| `Ctrl+C`       | Global         | Quit application                                   |
| `q`            | Global         | Quit application                                   |
| `q` / `Esc`    | File Viewer    | Close the file viewer                              |
| `q` / `Esc`    | Action Menu    | Close the action menu                              |
| `.`            | Main Panes     | Toggle display of hidden files/folders             |
| `Tab`          | Main Panes     | Switch focus between Folders and Files panes       |
| `↓` / `j`      | List Panes     | Move cursor down                                   |
| `↑` / `k`      | List Panes     | Move cursor up                                     |
| `PgDn` / `Space` | List Panes     | Move down one page                                 |
| `PgUp` / `b`   | List Panes     | Move up one page                                   |
| `g` / `Home`   | List Panes     | Go to the top of the list                          |
| `G` / `End`    | List Panes     | Go to the bottom of the list                       |
| `Enter`        | List Panes     | Open Action Menu for the selected item             |
| `↓` / `j`      | Action Menu    | Navigate down                                      |
| `↑` / `k`      | Action Menu    | Navigate up                                        |
| `Enter`        | Action Menu    | Execute the selected action                        |
| `↓` / `j`      | File Viewer    | Scroll down one line                               |
| `↑` / `k`      | File Viewer    | Scroll up one line                                 |
| `PgDn` / `Space` | File Viewer    | Scroll down one page                               |
| `PgUp` / `b`   | File Viewer    | Scroll up one page                                 |
| `g` / `Home`   | File Viewer    | Go to the start of the file                        |
| `G` / `End`    | File Viewer    | Go to the end of the file                          |

*Note: "Main Panes" context means when focus is on either the Folders or Files list view and no overlay (like the Action Menu or File Viewer) is active.*

## Configuration

Currently, `lazyls` does not use a configuration file. Behavior like file size limits for viewing/copying are defined as constants in the source code (`handlers.go`).

## Contributing

Contributions are welcome! Feel free to open issues or submit pull requests.

## License

This project is licensed under the MIT License - see the [LICENSE](https://github.com/AlexandrosLiaskos/lazyls/blob/main/LICENSE) file for details.
```