package tui

import (
	"testing"
)

func TestKeyMap_ShortHelp(t *testing.T) {
	bindings := keys.ShortHelp()

	if len(bindings) != 4 {
		t.Errorf("ShortHelp() returned %d bindings, want 4", len(bindings))
	}
}

func TestKeyMap_FullHelp(t *testing.T) {
	groups := keys.FullHelp()

	if len(groups) != 3 {
		t.Errorf("FullHelp() returned %d groups, want 3", len(groups))
	}

	expectedGroupSizes := []int{4, 3, 2}
	for i, group := range groups {
		if len(group) != expectedGroupSizes[i] {
			t.Errorf("Group %d has %d bindings, want %d", i, len(group), expectedGroupSizes[i])
		}
	}
}

func TestKeyMap_KeyBindings(t *testing.T) {
	tests := []struct {
		name    string
		binding interface{}
	}{
		{"Up", keys.Up},
		{"Down", keys.Down},
		{"Left", keys.Left},
		{"Right", keys.Right},
		{"Enter", keys.Enter},
		{"Back", keys.Back},
		{"Tab", keys.Tab},
		{"Export", keys.Export},
		{"Quit", keys.Quit},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.binding == nil {
				t.Errorf("%s binding should not be nil", tt.name)
			}
		})
	}
}

func TestKeyMap_NavigationKeys(t *testing.T) {
	upKeys := keys.Up.Keys()
	if len(upKeys) != 2 {
		t.Errorf("Up binding should have 2 keys, got %d", len(upKeys))
	}

	downKeys := keys.Down.Keys()
	if len(downKeys) != 2 {
		t.Errorf("Down binding should have 2 keys, got %d", len(downKeys))
	}

	quitKeys := keys.Quit.Keys()
	if len(quitKeys) != 2 {
		t.Errorf("Quit binding should have 2 keys, got %d", len(quitKeys))
	}
}

func TestKeyMap_HelpText(t *testing.T) {
	upHelp := keys.Up.Help()
	if upHelp.Key == "" {
		t.Error("Up binding help key should not be empty")
	}
	if upHelp.Desc == "" {
		t.Error("Up binding help desc should not be empty")
	}

	quitHelp := keys.Quit.Help()
	if quitHelp.Key == "" {
		t.Error("Quit binding help key should not be empty")
	}
}

func BenchmarkKeyMap_ShortHelp(b *testing.B) {
	for i := 0; i < b.N; i++ {
		keys.ShortHelp()
	}
}

func BenchmarkKeyMap_FullHelp(b *testing.B) {
	for i := 0; i < b.N; i++ {
		keys.FullHelp()
	}
}
