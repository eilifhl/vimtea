# VimTea - Vim-like Text Editor for TUIs

VimTea is a lightweight, Vim-inspired text editor for the terminal, built with Go and the [Bubble Tea](https://github.com/charmbracelet/bubbletea) TUI framework. It provides a modular, extensible foundation for building Vim-like text editors in your terminal applications.

[![Go Reference](https://pkg.go.dev/badge/github.com/kujtimiihoxha/vimtea.svg)](https://pkg.go.dev/github.com/kujtimiihoxha/vimtea)
[![Go Report Card](https://goreportcard.com/badge/github.com/kujtimiihoxha/vimtea)](https://goreportcard.com/report/github.com/kujtimiihoxha/vimtea)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

![VimTea Demo](./assets/demo.gif)

## Features

- Multiple editing modes (Normal, Insert, Visual, Command)
- Vim-like keybindings and commands
- Line numbers (absolute and relative)
- Count-based movement commands (e.g. `5j`, `10k`)
- Undo/redo functionality
- Visual mode selection (character and line-wise)
- Command mode
- Clipboard operations (yank, delete, paste)
- Word operations
- Operator-pending motions and text objects for `d`, `c`, and `y`
- Extensible architecture
- Custom key bindings
- Customizable highlighting

## Installation

```bash
go get github.com/eilifhl/vimtea
```

## Code Structure

The codebase has been organized into modular components:

- **model.go**: Main editor model and public interfaces
- **buffer.go**: Text buffer with undo/redo operations
- **cursor.go**: Cursor and text range operations
- **bindings.go**: Key binding registry
- **commands.go**: Command implementations
- **view.go**: Rendering functions
- **highlight.go**: Syntax highlighting
- **styles.go**: UI style definitions

## Usage

### Basic Usage

```go
package main

import (
    "log"

    tea "github.com/charmbracelet/bubbletea"
    "github.com/kujtimiihoxha/vimtea"
)

func main() {
    // Create a new editor with default options
    editor := vimtea.NewEditor(vimtea.WithFullScreen())

    // Run the editor
    p := tea.NewProgram(editor)
    if _, err := p.Run(); err != nil {
        log.Fatal(err)
    }
}
```

### Load Content

```go
package main

import (
    "log"

    tea "github.com/charmbracelet/bubbletea"
    "github.com/kujtimiihoxha/vimtea"
)

func main() {
    content := `This is a sample file
         with multiple lines
         for testing the editor`
    }

    // Create editor with content
    editor := vimtea.NewEditor(
        vimtea.WithContent(content),
        vimtea.WithFileName("example.txt"),
        vimtea.WithFullScreen(),
    )

    p := tea.NewProgram(editor)
    if _, err := p.Run(); err != nil {
        log.Fatal(err)
    }
}
```

### Custom Key Bindings

```go
package main

import (
    "log"

    tea "github.com/charmbracelet/bubbletea"
    "github.com/kujtimiihoxha/vimtea"
)

func main() {
    // Create editor
    editor := vimtea.NewEditor(vimtea.WithFullScreen())

    // Add custom binding
    editor.AddBinding(vimtea.KeyBinding{
        Key:         "ctrl+s",
        Mode:        vimtea.ModeNormal,
        Description: "Save file",
        Handler: func(b vimtea.Buffer) tea.Cmd {
            return vimtea.SetStatusMsg("File saved!")
        },
    })

    p := tea.NewProgram(editor)
    if _, err := p.Run(); err != nil {
        log.Fatal(err)
    }
}
```

### Custom Commands

```go
package main

import (
    "log"

    tea "github.com/charmbracelet/bubbletea"
    "github.com/kujtimiihoxha/vimtea"
)

func main() {
    // Create editor
    editor := vimtea.NewEditor(vimtea.WithFullScreen())

    // Add custom command
    editor.AddCommand("mysave", func(b vimtea.Buffer, args []string) tea.Cmd {
        return vimtea.SetStatusMsg("Custom save executed!")
    })

    p := tea.NewProgram(editor)
    if _, err := p.Run(); err != nil {
        log.Fatal(err)
    }
}
```

### Custom Styling

```go
package main

import (
    "log"

    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"
    "github.com/kujtimiihoxha/vimtea"
)

func main() {
    // Custom styles
    lineNumberStyle := lipgloss.NewStyle().
        Foreground(lipgloss.Color("#888888")).
        Background(lipgloss.Color("#222222")).
        PaddingRight(1)

    currentLineStyle := lipgloss.NewStyle().
        Foreground(lipgloss.Color("white")).
        Background(lipgloss.Color("#444444")).
        Bold(true).
        PaddingRight(1)

    cursorStyle := lipgloss.NewStyle().
        Background(lipgloss.Color("#CC8800")).
        Foreground(lipgloss.Color("black"))

    // Create editor with custom styles
    editor := vimtea.NewEditor(
        vimtea.WithLineNumberStyle(lineNumberStyle),
        vimtea.WithCurrentLineNumberStyle(currentLineStyle),
        vimtea.WithCursorStyle(cursorStyle),
        vimtea.WithRelativeNumbers(true),
        vimtea.WithFullScreen(),
    )

    p := tea.NewProgram(editor)
    if _, err := p.Run(); err != nil {
        log.Fatal(err)
    }
}
```

## Default Key Bindings

### Normal Mode

- `h`, `j`, `k`, `l`: Basic movement (left, down, up, right)
- Number prefixes: `5j`, `10k`: Move multiple lines at once
- `w`: Move to next word start
- `b`: Move to previous word start
- `W` / `B`: Move by whitespace-delimited WORDs
- `e` / `E`: Move to the end of the current word or WORD
- `ge` / `gE`: Move to the end of the previous word or WORD
- `0`: Move to start of line
- `^`: Move to first non-whitespace character in line
- `$`: Move to end of line
- `g_`: Move to the last non-whitespace character in line
- `gg`: Move to start of document
- `G`: Move to end of document
- `%`: Jump to the matching bracket
- `{` / `}`: Move between paragraphs
- `f<char>` / `F<char>`: Find a character forward or backward
- `t<char>` / `T<char>`: Move just before or after a character
- `;` / `,`: Repeat the last find motion
- `H` / `M` / `L`: Move to the top, middle, or bottom visible line
- `i`: Enter insert mode
- `a`: Append after cursor
- `A`: Append at end of line
- `I`: Insert at start of line
- `v`: Enter visual mode
- `V`: Enter visual line mode
- `:`: Enter command mode
- `x`: Delete character at cursor
- `dd`: Delete line
- `D`: Delete from cursor to end of line
- `dw`: Delete word
- `d0`, `d$`, `dgg`, `dG`: Delete with motions
- `yy`: Yank (copy) line
- `p`: Paste after cursor
- `P`: Paste before cursor
- `u`: Undo
- `ctrl+r`: Redo
- `o`: Open line below and enter insert mode
- `O`: Open line above and enter insert mode
- `diw`: Delete inner word
- `yiw`: Yank inner word
- `ciw`: Change inner word
- `daw` / `caw` / `yaw`: Around-word text object
- `di(`: Delete inner parentheses
- `yi(`: Yank inner parentheses
- `ci(`: Change inner parentheses
- `da(` / `ca(` / `ya(`: Around-parentheses text object
- `dib` / `yib` / `cib`: Same inner-parentheses operations using `b`
- `cw`: Change word
- `r<char>`: Replace the character under the cursor
- `zr`: Toggle relative line numbers
- `q`: Quit

### Insert Mode

- `esc`: Return to normal mode
- Arrow keys: Navigate
- `ctrl+a` / `ctrl+e`: Jump to start or end of line
- `ctrl+b` / `ctrl+f`: Move left or right one character
- `ctrl+left` / `ctrl+right`: Move backward or forward by one word
- `ctrl+p` / `ctrl+n`: Move up or down one line
- `ctrl+d` / `delete`: Delete the character under the cursor
- `ctrl+k`: Kill to end of line
- `ctrl+u`: Kill to start of line
- `ctrl+w` / `alt+backspace`: Delete previous word
- `alt+d` / `alt+delete`: Delete next word
- `ctrl+y`: Yank text back into the buffer
- `alt+y`: Cycle through the kill ring after a yank
- `ctrl+t`: Transpose adjacent characters
- `alt+t`: Transpose adjacent words
- `ctrl+o`: Open a new line without leaving insert mode
- `alt+b` / `alt+f`: Move backward or forward by one word
- `alt+<` / `alt+>`: Jump to the start or end of the document
- Regular typing inserts text

### Visual Mode

- `esc`: Return to normal mode
- `h`, `j`, `k`, `l`: Expand selection
- `y`: Yank selection
- `d`, `x`: Delete selection
- `p`: Replace selection with yanked text

### Command Mode

- `esc`: Cancel command
- `enter`: Execute command

## Extending VimTea

VimTea is designed to be easily extendable. You can:

1. Add custom key bindings with `editor.AddBinding()`
2. Create new commands with `editor.AddCommand()`
3. Modify the rendering style with custom style options
4. Access buffer operations directly via the Buffer interface
5. Create custom views by implementing the View interface
6. Customize the editor appearance with style options (WithTextStyle, WithLineNumberStyle, WithCurrentLineNumberStyle, etc.)

## Contributing

Contributions are welcome! Here's how you can contribute:

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

Please make sure to update tests as appropriate and follow the existing code style.

## Development Workflow

1. Clone the repository
2. Install dependencies: `go mod download`
3. Make your changes
4. Format code: `go fmt ./...`
5. Verify imports: `goimports -w .`
6. Run the example: `cd example && go run main.go`
7. Create tests for your changes

## Architecture

VimTea follows a modular architecture centered around these core components:

- **Editor**: The main interface that integrates all components
- **Buffer**: Manages text content with undo/redo operations
- **Cursor**: Handles positioning and selection
- **Bindings**: Registers and manages key bindings
- **Commands**: Implements editor commands (like Vim ex commands)
- **View**: Renders the editor to the terminal

These components follow clean separation of concerns, making it easier to:

- Add new features
- Test individual components
- Understand the codebase
- Customize functionality

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
