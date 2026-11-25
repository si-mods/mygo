package ui

import (
	"bytes"
	"testing"
	"time"
)

// countedMaterial paints a red that moves with the frame's time, counting
// its paints, and asks for the next frame when ask is set.
type countedMaterial struct {
	paints *int
	ask    bool
}

func (m countedMaterial) PaintMaterial(p *Painter, box Rect, radii [4]float32) {
	*m.paints++
	if m.ask {
		p.AnimationFrame()
	}
	p.FillRadii(box, radii, RGB(uint8(p.Now().UnixMilli()%200), 0, 0))
}

// materialView has a material that asks for frames or not, a drawing
// that does not move, and a child over the material.
func materialView(ask bool, materials, statics *int) func(c *context) {
	return func(c *context) {
		coreColumn(c).Background(RGB(250, 250, 250)).Padding(4).Children(func() {
			coreBox(c).Size(120, 24).Radius(6).Material(countedMaterial{materials, ask}).Children(func() {
				coreBox(c).Size(20, 10).Background(RGB(0, 0, 255))
			})
			coreBox(c).Size(120, 20).Draw(func(p *Painter, r Rect) {
				*statics++
				p.Fill(r, RGB(0, 200, 0), 0)
			})
		})
	}
}

// TestRepaintPartsMaterial: a material that asks for frames is painted
// again alone, as a Draw callback's part, in frames that only paint
// again: the window is not painted whole, the rest is not painted again,
// and the pixels are those of the window painted whole.
func TestRepaintPartsMaterial(t *testing.T) {
	var materials, statics, wholeMaterials, wholeStatics int
	parts := newRepaintTester(materialView(true, &materials, &statics))
	whole := newRepaintTester(materialView(true, &wholeMaterials, &wholeStatics))
	whole.rt.noParts = true
	m, s := materials, statics
	for i := range 6 {
		d := time.Duration(7+i*3) * time.Millisecond
		pb, _ := parts.frame(d)
		wb, _ := whole.frame(d)
		if pb || wb {
			t.Fatalf("frame %d built the view: %v %v", i, pb, wb)
		}
		if !bytes.Equal(parts.Image().Pix, whole.Image().Pix) {
			t.Fatalf("frame %d: painted in parts differs from painted whole", i)
		}
	}
	if parts.rt.partPaints != 6 || parts.rt.untracked {
		t.Errorf("%d of 6 frames painted in parts (untracked %v)", parts.rt.partPaints, parts.rt.untracked)
	}
	if materials-m != 6 || statics != s {
		t.Errorf("the material painted %d times and the still drawing %d times again, want 6 and 0", materials-m, statics-s)
	}
}

// TestRepaintPartsMaterialStill: a material that asks for nothing is not a
// part: a drawing that moves beside it repaints alone, and the material
// is not painted again.
func TestRepaintPartsMaterialStill(t *testing.T) {
	var materials, statics int
	tt := newRepaintTester(func(c *context) {
		materialView(false, &materials, &statics)(c)
		coreBox(c).Size(20, 20).Draw(func(p *Painter, r Rect) { p.AnimationFrame() })
	})
	if len(tt.rt.drawn) != 1 {
		t.Fatalf("%d parts, want the moving drawing's alone", len(tt.rt.drawn))
	}
	m := materials
	for range 3 {
		tt.frame(8 * time.Millisecond)
	}
	if tt.rt.partPaints != 3 || materials != m {
		t.Errorf("%d frames in parts, the material painted %d times again; want 3 and 0", tt.rt.partPaints, materials-m)
	}
}
