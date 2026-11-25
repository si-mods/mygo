package editor

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// DefaultTheme picks the syntax palette by the background's lightness,
// not the Dark flag, so a theme that paints light surfaces under a dark
// appearance still reads (and a dark surface under a light one).
func TestDefaultThemeBackground(t *testing.T) {
	light := DefaultTheme(&ui.Theme{Dark: true, Background: ui.Hex("#ffffff"), Text: ui.Hex("#173b61")})
	if got := light.Tokens[TokenString].Color; got != ui.Hex("#0a3069") {
		t.Fatalf("light background under Dark: string color %v, want the light palette", got)
	}
	dark := DefaultTheme(&ui.Theme{Dark: false, Background: ui.Hex("#0a1428"), Text: ui.Hex("#dcebff")})
	if got := dark.Tokens[TokenString].Color; got != ui.Hex("#a5d6ff") {
		t.Fatalf("dark background under light: string color %v, want the dark palette", got)
	}
}
