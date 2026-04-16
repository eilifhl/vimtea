package vimtea

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	atclipboard "github.com/atotto/clipboard"
	"golang.design/x/clipboard"
)

type clipboardBackend int

const (
	clipboardBackendNone clipboardBackend = iota
	clipboardBackendWayland
	clipboardBackendX11
	clipboardBackendAtotto
)

var clipboardBackendOnce sync.Once
var selectedClipboardBackend clipboardBackend
var wlCopyPath string
var wlPastePath string

func detectClipboardBackend() clipboardBackend {
	clipboardBackendOnce.Do(func() {
		if isWaylandSession() {
			if copyPath, err := exec.LookPath("wl-copy"); err == nil {
				if pastePath, err := exec.LookPath("wl-paste"); err == nil {
					wlCopyPath = copyPath
					wlPastePath = pastePath
					selectedClipboardBackend = clipboardBackendWayland
					return
				}
			}
		}

		if clipboard.Init() == nil {
			selectedClipboardBackend = clipboardBackendX11
			return
		}

		selectedClipboardBackend = clipboardBackendAtotto
	})

	return selectedClipboardBackend
}

func isWaylandSession() bool {
	return os.Getenv("WAYLAND_DISPLAY") != "" ||
		strings.EqualFold(os.Getenv("XDG_SESSION_TYPE"), "wayland")
}

func systemClipboardAvailable() bool {
	return detectClipboardBackend() != clipboardBackendNone
}

func startClipboardWatcher(m *editorModel) {
	switch detectClipboardBackend() {
	case clipboardBackendX11:
		go func() {
			ch := clipboard.Watch(context.Background(), clipboard.FmtText)
			for data := range ch {
				m.yankBuffer = string(data)
			}
		}()
	case clipboardBackendWayland:
		go func() {
			ticker := time.NewTicker(250 * time.Millisecond)
			defer ticker.Stop()

			lastValue := ""
			for range ticker.C {
				currentValue := readClipboardText()
				if currentValue == "" || currentValue == lastValue {
					continue
				}
				lastValue = currentValue
				m.yankBuffer = currentValue
			}
		}()
	default:
		return
	}
}

func writeClipboardText(text string) {
	switch detectClipboardBackend() {
	case clipboardBackendWayland:
		cmd := exec.Command(wlCopyPath)
		cmd.Stdin = bytes.NewBufferString(text)
		_ = cmd.Run()
	case clipboardBackendX11:
		clipboard.Write(clipboard.FmtText, []byte(text))
	case clipboardBackendAtotto:
		_ = atclipboard.WriteAll(text)
	}
}

func readClipboardText() string {
	switch detectClipboardBackend() {
	case clipboardBackendWayland:
		cmd := exec.Command(wlPastePath, "--no-newline")
		out, err := cmd.Output()
		if err != nil {
			return ""
		}
		return string(out)
	case clipboardBackendX11:
		data := clipboard.Read(clipboard.FmtText)
		return string(data)
	case clipboardBackendAtotto:
		text, err := atclipboard.ReadAll()
		if err != nil {
			return ""
		}
		return text
	default:
		return ""
	}
}
