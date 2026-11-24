package shader_test

import (
	"testing"

	"github.com/egoist/mygo/ui/shader"
)

// LoopRuns counts the runs of the loops' bodies for a pixel from the
// shader's source, and finds the loops whose bounds are not constants.
func TestLoopRuns(t *testing.T) {
	for _, c := range []struct {
		name, body string
		runs       int
		bounded    bool
	}{
		{"none", `func Fragment(dst vec4, pos vec2) vec4 {
	return vec4(pos.x)
}`, 0, true},
		{"constant", `func Fragment(dst vec4, pos vec2) vec4 {
	s := 0.0
	for i := 0; i < 10; i++ {
		s += float(i)
	}
	return vec4(s)
}`, 10, true},
		{"step and down", `func Fragment(dst vec4, pos vec2) vec4 {
	s := 0.0
	for i := 10; i > 0; i -= 3 {
		s += float(i)
	}
	for i := 0; i <= 4; i += 2 {
		s += float(i)
	}
	for i := range 5 {
		s += float(i)
	}
	return vec4(s)
}`, 4 + 3 + 5, true},
		{"float counter", `func Fragment(dst vec4, pos vec2) vec4 {
	s := 0.0
	for x := 0.0; x < 1.0; x += 0.25 {
		s += x
	}
	return vec4(s)
}`, 4, true},
		{"nested", `func Fragment(dst vec4, pos vec2) vec4 {
	s := 0.0
	for i := 0; i < 4; i++ {
		for j := 0; j < 8; j++ {
			s += float(i * j)
		}
	}
	return vec4(s)
}`, 4 + 4*8, true},
		{"called twice", `func sum(n float) float {
	s := 0.0
	for i := 0; i < 5; i++ {
		s += n
	}
	return s
}

func Fragment(dst vec4, pos vec2) vec4 {
	s := sum(pos.x) + sum(pos.y)
	for i := 0; i < 3; i++ {
		s += sum(float(i))
	}
	return vec4(s)
}`, 5 + 5 + 3*(1+5), true},
		{"larger branch", `func Fragment(dst vec4, pos vec2) vec4 {
	s := 0.0
	if pos.x < 1 {
		for i := 0; i < 2; i++ {
			s += 1
		}
	} else {
		for i := 0; i < 7; i++ {
			s += 1
		}
	}
	return vec4(s)
}`, 7, true},
		{"uniform bound", `var N int

func Fragment(dst vec4, pos vec2) vec4 {
	s := 0.0
	for i := 0; i < N; i++ {
		s += 1
	}
	return vec4(s)
}`, 0, false},
		{"uniform range", `var N int

func Fragment(dst vec4, pos vec2) vec4 {
	s := 0.0
	for range N {
		s += 1
	}
	return vec4(s)
}`, 0, false},
		{"never ends", `func Fragment(dst vec4, pos vec2) vec4 {
	s := 0.0
	for i := 0; i != 5; i += 2 {
		s += 1
	}
	return vec4(s)
}`, 0, false},
		{"uint wraps", `func Fragment(dst vec4, pos vec2) vec4 {
	s := 0.0
	for i := uint(3); i >= 0; i-- {
		s += 1
	}
	return vec4(s)
}`, 0, false},
		{"effect", `func ColorEffect(pos vec2, color vec4) vec4 {
	for i := 0; i < 6; i++ {
		color *= 0.9
	}
	return color
}`, 6, true},
	} {
		s, err := shader.Compile("package main\n\n" + c.body + "\n")
		if err != nil {
			t.Errorf("%s: %v", c.name, err)
			continue
		}
		runs, bounded := s.LoopRuns()
		if bounded != c.bounded || (bounded && runs != c.runs) {
			t.Errorf("%s: LoopRuns() = %d, %v, want %d, %v", c.name, runs, bounded, c.runs, c.bounded)
		}
	}
}
