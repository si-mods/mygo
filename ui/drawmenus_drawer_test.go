package ui

import "testing"

// TestDrawMenusDrawerOnlyWhenUsed checks that a window draws no popup
// menu until SetDrawMenus, and that turning it on links the drawn menus
// (SetMenuPresentation). Another window that never asks stays on the
// system's menus.
func TestDrawMenusDrawerOnlyWhenUsed(t *testing.T) {
	a := &drawApp{}
	tt := coreNewTester(a.view, 400, 300)
	tt.RightClick("Name")
	if tt.rt.drawsMenus() {
		t.Error("menus draw with the switch never on")
	}
	linked := menusInWindow.contextMenu != nil
	b := &drawApp{draw: true}
	tt = coreNewTester(b.view, 400, 300)
	if !tt.rt.drawsMenus() {
		t.Error("SetDrawMenus(true) did not draw menus")
	}
	if !linked && menusInWindow.contextMenu == nil {
		t.Error("SetDrawMenus(true) linked nothing")
	}
	c := &drawApp{}
	tt = coreNewTester(c.view, 400, 300)
	if tt.rt.drawsMenus() {
		t.Error("a window that never calls SetDrawMenus draws menus")
	}
}
