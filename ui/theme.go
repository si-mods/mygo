package ui

import (
	"runtime"
	"time"
)

// Theme holds the colors and metrics widgets use. Change a copy of
// LightTheme or DarkTheme and set it with Context.SetTheme.
type Theme struct {
	Dark bool
	// DisableAutoDimming keeps disabled elements at full opacity. Applications
	// opting in must provide their own disabled colors; input, focus and
	// accessibility still use the element's disabled state.
	DisableAutoDimming bool
	// Background fills the window.
	Background Color
	// Surface is the face of buttons, inputs and other controls;
	// SurfaceHover and SurfacePressed while hovered or pressed.
	Surface        Color
	SurfaceHover   Color
	SurfacePressed Color
	// Border outlines controls.
	Border Color
	// Text and TextMuted color text; TextMuted is for secondary text and
	// placeholders.
	Text      Color
	TextMuted Color
	// Accent colors primary buttons, checked controls and focus rings;
	// AccentText is text on it.
	Accent        Color
	AccentHover   Color
	AccentPressed Color
	AccentText    Color
	// InactiveAccent is what the accent fades to in a window that is not
	// drawn as active (Context.WindowActive), as AppKit fades an inactive
	// window's: the fill of selected rows, of checked check boxes, radio
	// buttons and switches, of sliders and progress bars, the line of the
	// selected tab and the icons of a sidebar, with AccentText on it as on
	// the accent; primary buttons show as the others there. Left zero, it
	// is a gray AccentText stands out on; set it to Accent to keep the
	// accent in every window.
	InactiveAccent Color
	Danger         Color
	// Warning and Success color what calls for attention and what went
	// well, as a Meter's levels.
	Warning Color
	Success Color
	// Selection highlights selected text.
	Selection Color
	// Focus is the ring around the control with the keyboard focus,
	// FocusRingWidth DIPs wide (2 by default; 0 is 2) and FocusRingOffset
	// DIPs outside its edge (1 in LightTheme and DarkTheme): 0 puts the
	// ring on the edge, and a negative offset inside it.
	// Element.FocusRingOffset and Element.FocusRingColor change them for
	// one element.
	Focus           Color
	FocusRingWidth  float32
	FocusRingOffset float32
	// Inverse fills tooltips and toasts, the theme turned over so that
	// they stand out from what they are over: dark in a light theme, light
	// in a dark one. InverseText is text on it. Left zero, they are Text
	// and Background.
	Inverse     Color
	InverseText Color
	// Scrollbar colors scroll bar thumbs, ScrollbarWidth DIPs wide (6 by
	// default; 0 is 6): the defaults of ScrollbarStyle, which overrides
	// them where it sets a field. It styles their tracks, hover and drag
	// colors, the width of their bars and thumbs, their corners and
	// shortest thumb; a theme that sets neither draws as it always has.
	Scrollbar      Color
	ScrollbarWidth float32
	ScrollbarStyle ScrollbarStyle
	// Radius rounds the corners of controls.
	Radius float32
	// Spacing is the unit of the room widgets leave inside and between
	// their parts: their paddings and gaps, and the sizes of check boxes,
	// switches, sliders and the rows of tables and trees, are multiples
	// of it. 4 by default: 3 makes widgets compact, 5 roomy; 0 is 4.
	Spacing float32
	// FontSize is the size of text, Font its family ("" is the system's).
	FontSize float32
	Font     string
	// TooltipDelay is how long the pointer rests on an element before its
	// tooltip shows (Tooltip, TooltipBase): 0 is 0.5 seconds, and a
	// negative delay shows it at once.
	TooltipDelay time.Duration
	// TooltipWarm is how soon after a tip goes another shows at once, as
	// the pointer moves along a toolbar: 0 is 0.4 seconds, and a negative
	// value turns the warm window off so every tip waits TooltipDelay.
	// Ignored when TooltipDelay is negative.
	TooltipWarm time.Duration
	// Menu styles the menus MyGo draws in the window, those of PopoverMenu,
	// PopoverMenuButton and of every context menu once the window draws
	// them (Context.SetDrawMenus). nil, and any part of it left at zero,
	// keeps MyGo's look. A pointer, so that a Theme stays comparable with
	// ==: a MenuStyle holds a function.
	Menu *MenuStyle
	// Button styles the standalone buttons MyGo draws in the window
	// (Button, PrimaryButton, MenuButton, PopoverMenuButton, SplitButton).
	// nil, and any part of it left at zero, keeps MyGo's look. A pointer,
	// so that a Theme stays comparable with ==: a ButtonStyle holds a
	// function.
	Button *ButtonStyle
	// Select styles the Select drop-down controls MyGo draws in the window.
	// nil, and any part of it left at zero, keeps MyGo's look. A pointer,
	// so that a Theme stays comparable with ==.
	Select *SelectStyle
	// Segmented styles Segmented controls MyGo draws in the window.
	// nil, and any part of it left at zero, keeps MyGo's look. A pointer,
	// so that a Theme stays comparable with ==.
	Segmented *SegmentedStyle
	// Checkbox styles Checkbox, CheckboxState and CheckboxGroup boxes MyGo
	// draws in the window. nil, and any part of it left at zero, keeps
	// MyGo's look. A pointer, so that a Theme stays comparable with ==.
	Checkbox *CheckboxStyle
	// Radio styles Radio buttons MyGo draws in the window. nil, and any
	// part of it left at zero, keeps MyGo's look. A pointer, so that a
	// Theme stays comparable with ==.
	Radio *RadioStyle

	// followed is what WithPreferences changed, nil where it has not.
	followed *followedTheme
}

// ButtonStyle styles standalone buttons MyGo draws in the window
// (Theme.Button). An app sets the parts it wants, and MyGo keeps its own
// look for those left at zero.
type ButtonStyle struct {
	// Style styles a standalone button after MyGo sets its layout and
	// default colors: its face (for example with Element.Material), border,
	// text color, radius, padding and shadows. primary is true for
	// PrimaryButton. Not called for toolbar or segmented buttons.
	Style func(b Element, primary bool)
}

// SelectStyle styles Select drop-down controls MyGo draws in the window
// (Theme.Select). An app sets the parts it wants, and MyGo keeps its own
// look for those left at zero.
type SelectStyle struct {
	// Trigger styles a Select's trigger button after MyGo sets its layout
	// and default colors. When nil, Theme.Button (if set) or MyGo's
	// default button look applies.
	Trigger func(b Element)
	// Chevron draws the drop-down indicator inside the trigger button;
	// nil keeps MyGo's chevron.
	Chevron func(c *Context)
}

// MenuStyle styles the menus MyGo draws in the window (Theme.Menu). An app
// sets the parts it wants, and MyGo keeps its own look for those left at
// zero. It changes how a menu looks, not how it works.
type MenuStyle struct {
	// Panel styles each menu's panel after MyGo set its layout, where it
	// attaches and its margin: its face, border, radius, shadow and
	// padding, and what it changes of the layout stays. level is 0 for the menu and 1,
	// 2 for the submenus inside it. nil keeps MyGo's.
	Panel func(panel Element, level int)
	// RowHeight is the least height of an item's row, RowRadius the radius
	// of its highlight and RowPadding the room before and after its
	// content; 0 keeps MyGo's.
	RowHeight, RowRadius, RowPadding float32
	// Highlight colors the row of the item the pointer or the keys are
	// on, HighlightText its text; zero keeps the accent and its text.
	Highlight, HighlightText Color
	// Text and TextMuted color menu item labels and secondary text
	// (accelerators, subtitles, disabled items); zero keeps the theme's
	// Text and TextMuted.
	Text, TextMuted Color
	// Separator colors the line between groups of items; zero keeps the
	// theme's Border.
	Separator Color
	// TextSize is the size of an item's text; 0 keeps the theme's.
	TextSize float32
}

// SegmentedStyle styles Segmented controls MyGo draws in the window
// (Theme.Segmented). An app sets the parts it wants, and MyGo keeps its
// own look for those left at zero.
type SegmentedStyle struct {
	// Track styles a Segmented control's outer track after MyGo sets its
	// layout and default colors. nil keeps MyGo's look.
	Track func(track Element)
	// Segment styles each segment after MyGo sets its layout and default
	// colors. selected is true for the chosen segment. nil keeps MyGo's
	// look.
	Segment func(seg Element, selected bool)
}

// CheckboxStyle styles Checkbox, CheckboxState and CheckboxGroup boxes
// MyGo draws in the window (Theme.Checkbox). An app sets the parts it
// wants, and MyGo keeps its own look for those left at zero.
type CheckboxStyle struct {
	// Box styles the box element after MyGo sets its layout and default
	// colors: its face (for example with Element.Material), border,
	// radius and size. row is the clickable row element holding hover,
	// pressed and disabled states. nil keeps MyGo's look.
	Box func(box, row Element, state CheckState)
	// Mark draws the check or dash inside box; nil keeps MyGo's mark.
	Mark func(p *Painter, r Rect, state CheckState)
}

// RadioStyle styles Radio buttons MyGo draws in the window (Theme.Radio).
// An app sets the parts it wants, and MyGo keeps its own look for those
// left at zero.
type RadioStyle struct {
	// Dot styles the outer circle element after MyGo sets its layout and
	// default colors: its face (for example with Element.Material),
	// border, radius and size. row is the clickable row element holding
	// hover, pressed and disabled states. nil keeps MyGo's look.
	Dot func(dot, row Element, selected bool)
	// Mark draws the inner dot when selected; nil keeps MyGo's dot.
	Mark func(p *Painter, r Rect, selected bool)
}

// followedTheme is a theme before and after following the desktop's
// preferences.
type followedTheme struct {
	_        [0]func() // used by pointer, so there is no value equality to link
	from, to Theme
}

// Space returns n units of the theme's Spacing, to size elements of your
// own in step with the widgets: ui.Column(c).Gap(t.Space(2)).
func (t *Theme) Space(n float32) float32 {
	s := t.Spacing
	if s <= 0 {
		s = 4
	}
	return n * s
}

// inactiveAccent returns what the accent fades to in a window that is not
// drawn as active (InactiveAccent).
func (t *Theme) inactiveAccent() Color {
	if t.InactiveAccent != Transparent {
		return t.InactiveAccent
	}
	// As far from AccentText as text on it needs, 4.5 times its contrast.
	far := Color{A: 255}
	if t.AccentText.gray().R <= 127 {
		far = Color{R: 255, G: 255, B: 255, A: 255}
	}
	return t.AccentText.Mix(far, 0.55)
}

// inverse returns the fill of tooltips and toasts, and the color of text
// on it.
func (t *Theme) inverse() (fill, text Color) {
	fill, text = t.Inverse, t.InverseText
	if fill == Transparent {
		fill = t.Text
	}
	if text == Transparent {
		text = t.Background
	}
	return fill, text
}

// focusRingWidth returns how wide the focus ring is (FocusRingWidth).
func (t *Theme) focusRingWidth() float32 {
	if t.FocusRingWidth <= 0 {
		return 2
	}
	return t.FocusRingWidth
}

func (t *Theme) scrollbarWidth() float32 {
	if t.ScrollbarWidth <= 0 {
		return 6
	}
	return t.ScrollbarWidth
}

// tooltipDelay returns how long the pointer rests on an element before its
// tip shows (TooltipDelay), and how soon after a tip goes the next one
// shows at once (TooltipWarm).
func (t *Theme) tooltipDelay() (delay, warm time.Duration) {
	if t.TooltipDelay < 0 {
		return 0, 0
	}
	delay = t.TooltipDelay
	if delay == 0 {
		delay = tooltipDelay
	}
	switch {
	case t.TooltipWarm < 0:
		warm = 0
	case t.TooltipWarm > 0:
		warm = t.TooltipWarm
	default:
		warm = tooltipWarm
	}
	return delay, warm
}

// Rem returns n times the theme's FontSize, as CSS's rem, to size elements
// with the text.
func (t *Theme) Rem(n float32) float32 {
	if t.FontSize <= 0 {
		return n * defaultFontSize()
	}
	return n * t.FontSize
}

func defaultFontSize() float32 {
	switch runtime.GOOS {
	case "darwin":
		return 13
	case "ios":
		return 17 // UIFont's body
	case "android":
		return 16 // Material's body large
	}
	return 14
}

// LightTheme returns the theme for a light appearance.
func LightTheme() *Theme {
	return &Theme{
		Background:      Hex("#ffffff"),
		Surface:         Hex("#f4f4f5"),
		SurfaceHover:    Hex("#e9e9ec"),
		SurfacePressed:  Hex("#dddde1"),
		Border:          Hex("#d9d9de"),
		Text:            Hex("#18181b"),
		TextMuted:       Hex("#71717a"),
		Accent:          Hex("#2563eb"),
		AccentHover:     Hex("#1d4ed8"),
		AccentPressed:   Hex("#1e40af"),
		AccentText:      Hex("#ffffff"),
		Danger:          Hex("#dc2626"),
		Warning:         Hex("#d97706"),
		Success:         Hex("#16a34a"),
		Selection:       RGBA(37, 99, 235, 0.25),
		Focus:           RGBA(37, 99, 235, 0.55),
		FocusRingWidth:  2,
		FocusRingOffset: 1,
		Scrollbar:       RGBA(0, 0, 0, 0.32),
		Radius:          6,
		Spacing:         4,
		FontSize:        defaultFontSize(),
	}
}

// DarkTheme returns the theme for a dark appearance.
func DarkTheme() *Theme {
	return &Theme{
		Dark:            true,
		Background:      Hex("#18181b"),
		Surface:         Hex("#27272a"),
		SurfaceHover:    Hex("#323236"),
		SurfacePressed:  Hex("#3c3c41"),
		Border:          Hex("#3f3f46"),
		Text:            Hex("#f4f4f5"),
		TextMuted:       Hex("#a1a1aa"),
		Accent:          Hex("#3b82f6"),
		AccentHover:     Hex("#60a5fa"),
		AccentPressed:   Hex("#2563eb"),
		AccentText:      Hex("#ffffff"),
		Danger:          Hex("#ef4444"),
		Warning:         Hex("#f59e0b"),
		Success:         Hex("#22c55e"),
		Selection:       RGBA(59, 130, 246, 0.4),
		Focus:           RGBA(96, 165, 250, 0.6),
		FocusRingWidth:  2,
		FocusRingOffset: 1,
		Scrollbar:       RGBA(255, 255, 255, 0.35),
		Radius:          6,
		Spacing:         4,
		FontSize:        defaultFontSize(),
	}
}
