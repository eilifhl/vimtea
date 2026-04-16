// Example application demonstrating the use of vimtea
// This opens itself and provides a Vim-like interface to edit the file
package main

import (
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/eilifhl/vimtea"
)

func loadExampleSource() ([]byte, string, error) {
	candidates := []string{
		"main.go",
		"example/main.go",
	}

	for _, path := range candidates {
		data, err := os.ReadFile(path)
		if err == nil {
			return data, path, nil
		}
		if !os.IsNotExist(err) {
			return nil, "", err
		}
	}

	return nil, "", os.ErrNotExist
}

func main() {
	// Create a log file
	logFile, err := os.OpenFile("debug.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o666)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}
	defer logFile.Close()

	// Set log output to the file
	log.SetOutput(logFile)

	buf, filePath, err := loadExampleSource()
	if err != nil {
		log.Fatalf("Failed to open source file: %v", err)
	}

	// Create a new editor with the file contents
	// WithFileName is used for syntax highlighting
	editor := vimtea.NewEditor(
		vimtea.WithContent(string(buf)),
		vimtea.WithFileName(filePath),
		vimtea.WithFullScreen(),
	)

	// Add a custom key binding for quitting with Ctrl+C
	editor.AddBinding(vimtea.KeyBinding{
		Key:         "ctrl+c",
		Mode:        vimtea.ModeNormal,
		Description: "Close the editor",
		Handler: func(b vimtea.Buffer) tea.Cmd {
			return tea.Quit
		},
	})

	// Add a custom command that can be invoked with :q
	editor.AddCommand("q", func(b vimtea.Buffer, _ []string) tea.Cmd {
		return tea.Quit
	})

	p := tea.NewProgram(editor, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		log.Printf("Error running program: %v", err)
	}
}
