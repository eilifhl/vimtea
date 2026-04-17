package vimtea

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommandExecution(t *testing.T) {
	editor := NewEditor(WithContent("Line 1\nLine 2\nLine 3"))
	model := editor.(*editorModel)

	// Test yank line command (find binding in registry)
	binding := model.registry.FindExact("yy", ModeNormal)
	require.NotNil(t, binding, "Binding for 'yy' not found")

	model.cursor = newCursor(1, 0)
	binding.Command(model)

	assert.Contains(t, model.yankBuffer, "Line 2", "yankLine should set yankBuffer to contain 'Line 2'")

	// Test delete line command
	deleteBinding := model.registry.FindExact("dd", ModeNormal)
	require.NotNil(t, deleteBinding, "Binding for 'dd' not found")

	model.cursor = newCursor(1, 0)
	deleteBinding.Command(model)

	assert.Equal(t, 2, model.buffer.lineCount(), "deleteLine should remove a line")
	assert.Equal(t, "Line 3", model.buffer.Line(1), "After deletion, line 1 should be 'Line 3'")
}

func TestPasteCommands(t *testing.T) {
	editor := NewEditor(WithContent("Line 1\nLine 2\nLine 3"))
	model := editor.(*editorModel)

	// Set up yankBuffer
	model.yankBuffer = "Yanked content"

	// Test paste after command
	pasteAfterBinding := model.registry.FindExact("p", ModeNormal)
	require.NotNil(t, pasteAfterBinding, "Binding for 'p' not found")

	model.cursor = newCursor(0, 5)
	pasteAfterBinding.Command(model)

	expectedContent := "Line 1Yanked content\nLine 2\nLine 3"
	assert.Equal(t, expectedContent, model.buffer.text(), "pasteAfter should insert at cursor position")

	// Test paste before command
	pasteBeforeBinding := model.registry.FindExact("P", ModeNormal)
	require.NotNil(t, pasteBeforeBinding, "Binding for 'P' not found")

	// Reset buffer
	model.buffer.lines = []string{"Line 1", "Line 2", "Line 3"}
	model.cursor = newCursor(0, 5)
	pasteBeforeBinding.Command(model)

	expectedContent = "Line Yanked content1\nLine 2\nLine 3"
	assert.Equal(t, expectedContent, model.buffer.text(), "pasteBefore should insert at cursor position")

	// Test line-wise paste
	// Reset buffer
	model.buffer.lines = []string{"Line 1", "Line 2", "Line 3"}
	model.yankBuffer = "\nYanked line"
	model.cursor = newCursor(1, 0)
	pasteAfterBinding.Command(model)

	expectedContent = "Line 1\nLine 2\nYanked line\nLine 3"
	assert.Equal(t, expectedContent, model.buffer.text(), "pasteAfter with line-wise content should insert as new line")
}

func TestInsertModeCommands(t *testing.T) {
	editor := NewEditor(WithContent("Line 1\nLine 2"))
	model := editor.(*editorModel)

	// Test insert at beginning of line (I command)
	insertStartBinding := model.registry.FindExact("I", ModeNormal)
	require.NotNil(t, insertStartBinding, "Binding for 'I' not found")

	model.cursor = newCursor(1, 2)
	insertStartBinding.Command(model)

	assert.Equal(t, 0, model.cursor.Col, "I command should move cursor to col 0")
	assert.Equal(t, ModeInsert, model.mode, "I command should switch to insert mode")

	// Test insert at end of line (A command)
	appendEndBinding := model.registry.FindExact("A", ModeNormal)
	require.NotNil(t, appendEndBinding, "Binding for 'A' not found")

	model.mode = ModeNormal
	model.cursor = newCursor(1, 2)
	appendEndBinding.Command(model)

	assert.Equal(t, 6, model.cursor.Col, "A command should move cursor to end of line")
	assert.Equal(t, ModeInsert, model.mode, "A command should switch to insert mode")

	// Test insert new line below (o command)
	openBelowBinding := model.registry.FindExact("o", ModeNormal)
	require.NotNil(t, openBelowBinding, "Binding for 'o' not found")

	model.mode = ModeNormal
	model.cursor = newCursor(0, 0)
	openBelowBinding.Command(model)

	assert.Equal(t, 3, model.buffer.lineCount(), "o command should add a new line")
	assert.Equal(t, 1, model.cursor.Row, "o command should position cursor at new line row")
	assert.Equal(t, 0, model.cursor.Col, "o command should position cursor at start of new line")
}

func TestCursorMovementCommands(t *testing.T) {
	editor := NewEditor(WithContent("Line 1\nLine 2\nLine 3"))
	model := editor.(*editorModel)

	// Test move down (j)
	downBinding := model.registry.FindExact("j", ModeNormal)
	require.NotNil(t, downBinding, "Binding for 'j' not found")

	model.cursor = newCursor(0, 0)
	downBinding.Command(model)

	assert.Equal(t, 1, model.cursor.Row, "j command should increase row by 1")

	// Test move up (k)
	upBinding := model.registry.FindExact("k", ModeNormal)
	require.NotNil(t, upBinding, "Binding for 'k' not found")

	upBinding.Command(model)

	assert.Equal(t, 0, model.cursor.Row, "k command should decrease row by 1")

	// Test move right (l)
	rightBinding := model.registry.FindExact("l", ModeNormal)
	require.NotNil(t, rightBinding, "Binding for 'l' not found")

	rightBinding.Command(model)

	assert.Equal(t, 1, model.cursor.Col, "l command should increase col by 1")

	// Test move left (h)
	leftBinding := model.registry.FindExact("h", ModeNormal)
	require.NotNil(t, leftBinding, "Binding for 'h' not found")

	leftBinding.Command(model)

	assert.Equal(t, 0, model.cursor.Col, "h command should decrease col by 1")
}

func TestDeleteWordAndReplaceCharacterCommands(t *testing.T) {
	editor := NewEditor(WithContent("foo bar"))
	model := editor.(*editorModel)

	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}}
	updated, _ := model.Update(keyMsg)
	updatedModel := updated.(*editorModel)

	keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'w'}}
	updated, _ = updatedModel.Update(keyMsg)
	updatedModel = updated.(*editorModel)

	assert.Equal(t, "bar", updatedModel.buffer.text(), "dw should delete the current word and the following space")

	editor = NewEditor(WithContent("hello"))
	model = editor.(*editorModel)

	keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}}
	updated, _ = model.Update(keyMsg)
	updatedModel = updated.(*editorModel)

	keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}}
	updated, _ = updatedModel.Update(keyMsg)
	updatedModel = updated.(*editorModel)

	assert.Equal(t, "xello", updatedModel.buffer.text(), "r should replace the character under the cursor")
	assert.False(t, updatedModel.pendingReplace, "Replace state should clear after the replacement character is used")
}

func TestOperatorPendingDeleteMotions(t *testing.T) {
	editor := NewEditor(WithContent("hello world\nsecond line"))
	model := editor.(*editorModel)

	model.cursor = newCursor(0, 5)
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	updatedModel := updated.(*editorModel)
	updated, _ = updatedModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'0'}})
	updatedModel = updated.(*editorModel)

	assert.Equal(t, " world\nsecond line", updatedModel.buffer.text(), "d0 should delete back to the start of the line")

	editor = NewEditor(WithContent("pre (hello) post"))
	model = editor.(*editorModel)
	model.cursor = newCursor(0, 6)

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	updatedModel = updated.(*editorModel)
	updated, _ = updatedModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	updatedModel = updated.(*editorModel)
	updated, _ = updatedModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'('}})
	updatedModel = updated.(*editorModel)

	assert.Equal(t, "pre  post", updatedModel.buffer.text(), "da( should delete the parens and their contents")
}

func TestInsertModeDeleteForward(t *testing.T) {
	editor := NewEditor(WithContent("hello"))
	model := editor.(*editorModel)

	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}}
	updated, _ := model.Update(keyMsg)
	updatedModel := updated.(*editorModel)

	updatedModel.cursor = newCursor(0, 1)

	updated, _ = updatedModel.Update(tea.KeyMsg{Type: tea.KeyDelete})
	updatedModel = updated.(*editorModel)
	assert.Equal(t, "hllo", updatedModel.buffer.text(), "delete should delete the character under the cursor in insert mode")

	updatedModel.cursor = newCursor(0, len(updatedModel.buffer.Line(0)))
	updated, _ = updatedModel.Update(tea.KeyMsg{Type: tea.KeyCtrlD})
	updatedModel = updated.(*editorModel)
	assert.Equal(t, "hllo", updatedModel.buffer.text(), "delete at end of line should be a no-op")
}

func TestInsertModeEmacsMotions(t *testing.T) {
	editor := NewEditor(WithContent("hello world\nsecond line"))
	model := editor.(*editorModel)

	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}}
	updated, _ := model.Update(keyMsg)
	updatedModel := updated.(*editorModel)

	updatedModel.cursor = newCursor(0, 5)

	updated, _ = updatedModel.Update(tea.KeyMsg{Type: tea.KeyCtrlA})
	updatedModel = updated.(*editorModel)
	assert.Equal(t, 0, updatedModel.cursor.Col, "ctrl+a should move to the start of the line in insert mode")

	updated, _ = updatedModel.Update(tea.KeyMsg{Type: tea.KeyCtrlE})
	updatedModel = updated.(*editorModel)
	assert.Equal(t, 10, updatedModel.cursor.Col, "ctrl+e should move to the end of the line in insert mode")

	updated, _ = updatedModel.Update(tea.KeyMsg{Type: tea.KeyCtrlB})
	updatedModel = updated.(*editorModel)
	assert.Equal(t, 9, updatedModel.cursor.Col, "ctrl+b should move left in insert mode")

	updated, _ = updatedModel.Update(tea.KeyMsg{Type: tea.KeyCtrlF})
	updatedModel = updated.(*editorModel)
	assert.Equal(t, 10, updatedModel.cursor.Col, "ctrl+f should move right in insert mode")

	updated, _ = updatedModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}, Alt: true})
	updatedModel = updated.(*editorModel)
	assert.Equal(t, 6, updatedModel.cursor.Col, "alt+b should move to the previous word in insert mode")

	updated, _ = updatedModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}, Alt: true})
	updatedModel = updated.(*editorModel)
	assert.Equal(t, 11, updatedModel.cursor.Col, "alt+f should move forward by one word in insert mode")

	updatedModel.cursor = newCursor(0, 12)

	updated, _ = updatedModel.Update(tea.KeyMsg{Type: tea.KeyCtrlLeft})
	updatedModel = updated.(*editorModel)
	assert.Equal(t, 6, updatedModel.cursor.Col, "ctrl+left should move to the previous word in insert mode")

	updated, _ = updatedModel.Update(tea.KeyMsg{Type: tea.KeyCtrlRight})
	updatedModel = updated.(*editorModel)
	assert.Equal(t, 11, updatedModel.cursor.Col, "ctrl+right should move forward by one word in insert mode")
}

func TestInsertModeEmacsKillAndYankCommands(t *testing.T) {
	editor := NewEditor(WithContent("hello brave world"))
	model := editor.(*editorModel)

	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}}
	updated, _ := model.Update(keyMsg)
	updatedModel := updated.(*editorModel)

	updatedModel.cursor = newCursor(0, 6)

	updated, _ = updatedModel.Update(tea.KeyMsg{Type: tea.KeyCtrlK})
	updatedModel = updated.(*editorModel)
	assert.Equal(t, "hello ", updatedModel.buffer.text(), "ctrl+k should kill to the end of the line")

	updated, _ = updatedModel.Update(tea.KeyMsg{Type: tea.KeyCtrlY})
	updatedModel = updated.(*editorModel)
	assert.Equal(t, "hello brave world", updatedModel.buffer.text(), "ctrl+y should yank the last killed text back into the buffer")

	updatedModel.cursor = newCursor(0, 12)
	updated, _ = updatedModel.Update(tea.KeyMsg{Type: tea.KeyCtrlU})
	updatedModel = updated.(*editorModel)
	assert.Equal(t, "world", updatedModel.buffer.text(), "ctrl+u should kill to the start of the line")
}

func TestInsertModeWordDeletionMacros(t *testing.T) {
	editor := NewEditor(WithContent("hello brave world"))
	model := editor.(*editorModel)

	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}}
	updated, _ := model.Update(keyMsg)
	updatedModel := updated.(*editorModel)

	updatedModel.cursor = newCursor(0, 12)
	updated, _ = updatedModel.Update(tea.KeyMsg{Type: tea.KeyCtrlW})
	updatedModel = updated.(*editorModel)
	assert.Equal(t, "hello world", updatedModel.buffer.text(), "ctrl+w should delete the previous word")

	editor = NewEditor(WithContent("hello brave world"))
	model = editor.(*editorModel)
	keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}}
	updated, _ = model.Update(keyMsg)
	updatedModel = updated.(*editorModel)

	updatedModel.cursor = newCursor(0, 6)
	updated, _ = updatedModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}, Alt: true})
	updatedModel = updated.(*editorModel)
	assert.Equal(t, "hello world", updatedModel.buffer.text(), "alt+d should delete the next word")

	editor = NewEditor(WithContent("hello brave world"))
	model = editor.(*editorModel)
	keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}}
	updated, _ = model.Update(keyMsg)
	updatedModel = updated.(*editorModel)

	updatedModel.cursor = newCursor(0, 6)
	updated, _ = updatedModel.Update(tea.KeyMsg{Type: tea.KeyDelete, Alt: true})
	updatedModel = updated.(*editorModel)
	assert.Equal(t, "hello world", updatedModel.buffer.text(), "alt+delete should delete the next word")
}

func TestInsertModeTransposeCharacters(t *testing.T) {
	editor := NewEditor(WithContent("teh"))
	model := editor.(*editorModel)

	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}}
	updated, _ := model.Update(keyMsg)
	updatedModel := updated.(*editorModel)

	updatedModel.cursor = newCursor(0, 2)
	updated, _ = updatedModel.Update(tea.KeyMsg{Type: tea.KeyCtrlT})
	updatedModel = updated.(*editorModel)

	assert.Equal(t, "the", updatedModel.buffer.text(), "ctrl+t should transpose adjacent characters")
}

func TestInnerParenCommands(t *testing.T) {
	editor := NewEditor(WithContent("prefix (hello world) suffix"))
	model := editor.(*editorModel)

	model.cursor = newCursor(0, 10)

	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}}
	updated, _ := model.Update(keyMsg)
	updatedModel := updated.(*editorModel)

	keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}}
	updated, _ = updatedModel.Update(keyMsg)
	updatedModel = updated.(*editorModel)

	keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}}
	updated, _ = updatedModel.Update(keyMsg)
	updatedModel = updated.(*editorModel)

	assert.Equal(t, "prefix () suffix", updatedModel.buffer.text(), "cib should delete the inner parenthesized contents")
	assert.Equal(t, ModeInsert, updatedModel.mode, "cib should switch to insert mode")
	assert.Equal(t, 8, updatedModel.cursor.Col, "cib should leave the cursor inside the parentheses")

	editor = NewEditor(WithContent("prefix (hello world) suffix"))
	model = editor.(*editorModel)
	model.cursor = newCursor(0, 0)

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	updatedModel = updated.(*editorModel)
	updated, _ = updatedModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	updatedModel = updated.(*editorModel)
	updated, _ = updatedModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
	updatedModel = updated.(*editorModel)

	assert.Equal(t, "prefix () suffix", updatedModel.buffer.text(), "cib should also target the nearest parentheses when outside them")
}

func TestInnerAndAroundQuoteCommands(t *testing.T) {
	editor := NewEditor(WithContent(`prefix "hello world" suffix`))
	model := editor.(*editorModel)

	model.cursor = newCursor(0, 10)

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	updatedModel := updated.(*editorModel)
	updated, _ = updatedModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	updatedModel = updated.(*editorModel)
	updated, _ = updatedModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'"'}})
	updatedModel = updated.(*editorModel)

	assert.Equal(t, `prefix "" suffix`, updatedModel.buffer.text(), `ci" should delete the contents inside double quotes`)
	assert.Equal(t, ModeInsert, updatedModel.mode, `ci" should switch to insert mode`)
	assert.Equal(t, 8, updatedModel.cursor.Col, `ci" should leave the cursor inside the quotes`)

	editor = NewEditor(WithContent(`prefix "hello world" suffix`))
	model = editor.(*editorModel)
	model.cursor = newCursor(0, 0)

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	updatedModel = updated.(*editorModel)
	updated, _ = updatedModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	updatedModel = updated.(*editorModel)
	updated, _ = updatedModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'"'}})
	updatedModel = updated.(*editorModel)

	assert.Equal(t, `prefix "" suffix`, updatedModel.buffer.text(), `ci" should also target the nearest quotes when outside them`)

	editor = NewEditor(WithContent(`prefix 'hello world' suffix`))
	model = editor.(*editorModel)
	model.cursor = newCursor(0, 10)

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	updatedModel = updated.(*editorModel)
	updated, _ = updatedModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	updatedModel = updated.(*editorModel)
	updated, _ = updatedModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'\''}})
	updatedModel = updated.(*editorModel)

	assert.Equal(t, `prefix  suffix`, updatedModel.buffer.text(), `da' should delete the contents and surrounding single quotes`)
}

func TestAdditionalVimMotions(t *testing.T) {
	editor := NewEditor(WithContent("alpha beta gamma\n\n(delta)\n{block}"))
	model := editor.(*editorModel)

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	updatedModel := updated.(*editorModel)
	assert.Equal(t, 4, updatedModel.cursor.Col, "e should move to the end of the current word")

	updated, _ = updatedModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	updatedModel = updated.(*editorModel)
	updated, _ = updatedModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	updatedModel = updated.(*editorModel)
	assert.Equal(t, 4, updatedModel.cursor.Col, "ge at the first word should stay on the current word end")

	updated, _ = updatedModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	updatedModel = updated.(*editorModel)
	updated, _ = updatedModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	updatedModel = updated.(*editorModel)
	assert.Equal(t, 13, updatedModel.cursor.Col, "fm should find the next matching character")

	updated, _ = updatedModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{';'}})
	updatedModel = updated.(*editorModel)
	assert.Equal(t, 14, updatedModel.cursor.Col, "; should repeat the last find motion")
}

func TestBracketMatchAndWordObjects(t *testing.T) {
	editor := NewEditor(WithContent("before [middle] after\n{block text}"))
	model := editor.(*editorModel)

	model.cursor = newCursor(0, 7)
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	updatedModel := updated.(*editorModel)
	updated, _ = updatedModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	updatedModel = updated.(*editorModel)
	updated, _ = updatedModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'['}})
	updatedModel = updated.(*editorModel)
	assert.Equal(t, "before [] after\n{block text}", updatedModel.buffer.text(), "di[ should delete inside square brackets")

	editor = NewEditor(WithContent("{alpha beta}"))
	model = editor.(*editorModel)
	model.cursor = newCursor(0, 1)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	updatedModel = updated.(*editorModel)
	updated, _ = updatedModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	updatedModel = updated.(*editorModel)
	updated, _ = updatedModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'B'}})
	updatedModel = updated.(*editorModel)
	assert.Equal(t, "", updatedModel.buffer.text(), "caB should delete braces and contents")
	assert.Equal(t, ModeInsert, updatedModel.mode, "caB should enter insert mode")

	editor = NewEditor(WithContent("(alpha)\nnext"))
	model = editor.(*editorModel)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'%'}})
	updatedModel = updated.(*editorModel)
	assert.Equal(t, 6, updatedModel.cursor.Col, "% should jump to the matching bracket")
}

func TestInsertModeYankPopAndTransposeWords(t *testing.T) {
	editor := NewEditor(WithContent("hello brave world"))
	model := editor.(*editorModel)

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	updatedModel := updated.(*editorModel)
	updatedModel.cursor = newCursor(0, 6)

	updated, _ = updatedModel.Update(tea.KeyMsg{Type: tea.KeyCtrlK})
	updatedModel = updated.(*editorModel)
	updatedModel.cursor = newCursor(0, 6)
	updated, _ = updatedModel.Update(tea.KeyMsg{Type: tea.KeyCtrlU})
	updatedModel = updated.(*editorModel)

	editor = NewEditor(WithContent(""))
	model = editor.(*editorModel)
	model.mode = ModeInsert
	model.killRing = append(model.killRing, "world", "hello brave ")
	model.yankBuffer = "world"
	model.cursor = newCursor(0, 0)

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyCtrlY})
	updatedModel = updated.(*editorModel)
	assert.Equal(t, "world", updatedModel.buffer.text(), "ctrl+y should insert the latest kill-ring entry")

	updated, _ = updatedModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}, Alt: true})
	updatedModel = updated.(*editorModel)
	assert.Equal(t, "hello brave ", updatedModel.buffer.text(), "alt+y should replace the previous yank with the next kill-ring entry")

	editor = NewEditor(WithContent("brave hello world"))
	model = editor.(*editorModel)
	model.mode = ModeInsert
	model.cursor = newCursor(0, 11)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}, Alt: true})
	updatedModel = updated.(*editorModel)
	assert.Equal(t, "brave world hello", updatedModel.buffer.text(), "alt+t should transpose the adjacent words around point")
}

func TestWrappedMovementCommands(t *testing.T) {
	editor := NewEditor(WithContent("Line 1\nLine 2\nLine 3"))
	model := editor.(*editorModel)

	// Test move to beginning of line (0)
	startBinding := model.registry.FindExact("0", ModeNormal)
	require.NotNil(t, startBinding, "Binding for '0' not found")

	model.cursor = newCursor(1, 3)
	startBinding.Command(model)

	assert.Equal(t, 0, model.cursor.Col, "0 command should set col to 0")

	// Test move to end of line ($)
	endBinding := model.registry.FindExact("$", ModeNormal)
	require.NotNil(t, endBinding, "Binding for '$' not found")

	endBinding.Command(model)

	assert.Equal(t, 5, model.cursor.Col, "$ command should move to end of line")

	// Test space (advance cursor)
	spaceBinding := model.registry.FindExact(" ", ModeNormal)
	require.NotNil(t, spaceBinding, "Binding for space not found")

	// Position cursor at second-to-last position of first line
	model.cursor = newCursor(0, 4)
	spaceBinding.Command(model)

	// Should move to last column
	assert.Equal(t, 5, model.cursor.Col, "Space should move right by 1")

	// One more space should wrap to next line
	spaceBinding.Command(model)

	assert.Equal(t, 1, model.cursor.Row, "Space at end of line should wrap to next line (row)")
	assert.Equal(t, 0, model.cursor.Col, "Space at end of line should wrap to col 0")
}

func TestJumpCommands(t *testing.T) {
	editor := NewEditor(WithContent("Line 1\nLine 2\nLine 3\nLine 4\nLine 5"))
	model := editor.(*editorModel)

	// Test move to first line (gg)
	startDocBinding := model.registry.FindExact("gg", ModeNormal)
	require.NotNil(t, startDocBinding, "Binding for 'gg' not found")

	model.cursor = newCursor(3, 0)
	startDocBinding.Command(model)

	assert.Equal(t, 0, model.cursor.Row, "gg command should set row to 0")

	// Test move to last line (G)
	endDocBinding := model.registry.FindExact("G", ModeNormal)
	require.NotNil(t, endDocBinding, "Binding for 'G' not found")

	endDocBinding.Command(model)

	assert.Equal(t, 4, model.cursor.Row, "G command should move to last line (4)")
}

func TestCommandLineCommands(t *testing.T) {
	editor := NewEditor()
	model := editor.(*editorModel)

	// Register test command
	cmdExecuted := false
	model.commands.Register("test", func(m *editorModel) tea.Cmd {
		cmdExecuted = true
		return nil
	})

	// Set up command mode
	model.mode = ModeCommand
	model.commandBuffer = "test"

	// Get the execute command binding
	execBinding := model.registry.FindExact("enter", ModeCommand)
	require.NotNil(t, execBinding, "Binding for 'enter' in command mode not found")

	cmd := execBinding.Command(model)
	model.Update(cmd())

	assert.True(t, cmdExecuted, "Command execution should run registered command")
	assert.Equal(t, ModeNormal, model.mode, "After command execution, mode should be Normal")
}
