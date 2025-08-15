package tui

import kb "github.com/charmbracelet/bubbles/key"

// KeyMap defines global keybindings used across the TUI.
type KeyMap struct {
	Quit        kb.Binding
	Help        kb.Binding
	SwitchLeft  kb.Binding
	SwitchRight kb.Binding
	PopupOpen   kb.Binding
	PopupClose  kb.Binding
}
