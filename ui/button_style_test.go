package ui

import "testing"

func TestButtonStyleAppliesOnlyToStandaloneButtons(t *testing.T) {
	var calls []bool
	var gotBorder [4]float32
	var gotRadius [4]float32
	theme := LightTheme()
	theme.Button = &ButtonStyle{
		Style: func(b Element, primary bool) {
			calls = append(calls, primary)
			b.Radius(3).Border(1, Hex("#003c74"))
		},
	}

	seg := 0
	tt := NewTester(func(c *Context) {
		c.SetTheme(theme)
		okEl := Button(c, "OK")
		if n := okEl.lookup(); n != nil {
			gotBorder = n.border
			gotRadius = n.radius
		}
		PrimaryButton(c, "Save")
		Toolbar(c, func() { Button(c, "Tool") })
		Segmented(c, &seg, "A", "B")
	}, 400, 300)

	if len(calls) != 2 || calls[0] || !calls[1] {
		t.Fatalf("ButtonStyle.Style calls = %v, want [false, true]", calls)
	}
	if gotBorder != [4]float32{1, 1, 1, 1} || gotRadius != [4]float32{3, 3, 3, 3} {
		t.Fatalf("standalone button border/radius = %v / %v, want 1 / 3", gotBorder, gotRadius)
	}
	if _, ok := tt.Find("OK"); !ok {
		t.Fatal("OK button not found")
	}
}

func TestSelectStyleOverridesTriggerAndChevron(t *testing.T) {
	triggerCalled := false
	chevronCalled := false
	theme := LightTheme()
	theme.Button = &ButtonStyle{
		Style: func(b Element, primary bool) {
			t.Error("ButtonStyle.Style should not run when SelectStyle.Trigger is set")
		},
	}
	theme.Select = &SelectStyle{
		Trigger: func(b Element) {
			triggerCalled = true
			b.Radius(0).Border(1, Hex("#7f9db9"))
		},
		Chevron: func(c *Context) {
			chevronCalled = true
			Text(c, "v")
		},
	}

	sel := "Untitled"
	tt := NewTester(func(c *Context) {
		c.SetTheme(theme)
		Select(c, &sel, []string{"Untitled", "Other"})
	}, 400, 300)

	if !triggerCalled || !chevronCalled {
		t.Fatalf("SelectStyle calls: trigger=%v chevron=%v", triggerCalled, chevronCalled)
	}
	if _, ok := tt.Find("v"); !ok {
		t.Fatal("custom SelectStyle.Chevron was not rendered")
	}
}

func TestSegmentedStyleCallsTrackAndSegment(t *testing.T) {
	trackCalled := false
	var segs []bool
	theme := LightTheme()
	theme.Segmented = &SegmentedStyle{
		Track: func(track Element) { trackCalled = true },
		Segment: func(seg Element, selected bool) {
			segs = append(segs, selected)
		},
	}

	sel := 1
	NewTester(func(c *Context) {
		c.SetTheme(theme)
		Segmented(c, &sel, "A", "B")
	}, 400, 300)

	if !trackCalled || len(segs) != 2 || segs[0] || !segs[1] {
		t.Fatalf("SegmentedStyle calls: track=%v segs=%v, want true [false, true]", trackCalled, segs)
	}
}
