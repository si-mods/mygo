package ui

import "testing"

// TestDrawMenusDrawerOnlyWhenUsed checks that an app that neither calls
// SetDrawMenus nor uses a PopoverMenu holds none of the drawn menus' code.
func TestDrawMenusDrawerOnlyWhenUsed(t *testing.T) {
	a := &drawApp{}
	tt := coreNewTester(a.view, 400, 300)
	tt.RightClick("Name")
	if tt.rt.drawer != nil {
		t.Error("the drawn menus' code is registered with the switch never on")
	}
	b := &drawApp{draw: true}
	tt = coreNewTester(b.view, 400, 300)
	if tt.rt.drawer == nil {
		t.Error("SetDrawMenus(true) registered nothing")
	}
}
