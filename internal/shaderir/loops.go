package shaderir

import (
	"go/constant"
	"math"
)

// maxFloatSteps is how many steps of a float counter LoopRuns takes, in
// float32 as the GPU does, to count a loop exactly; a longer one counts
// in float64.
const maxFloatSteps = 1 << 16

// LoopRuns reports the most times the loop bodies of the fragment entry
// point run for one pixel, every loop's counted: a loop of n runs inside
// one of m adds m + m*n. A loop in a function counts at each call, times
// the loops around it; the larger branch of an if counts. bounded is
// false when a loop's start or end is not a constant, or its counter
// never reaches the end (or wraps around); runs saturate at math.MaxInt.
func (p *Program) LoopRuns() (runs int, bounded bool) {
	funcs := map[int]*Block{}
	for _, f := range p.Funcs {
		funcs[f.Index] = f.Block
	}
	l := loopCounter{funcs: funcs, done: map[int]int{}, open: map[int]bool{}, bounded: true}
	runs = l.block(p.FragmentFunc.Block)
	return runs, l.bounded
}

type loopCounter struct {
	funcs   map[int]*Block
	done    map[int]int // the runs of a function, counted
	open    map[int]bool
	bounded bool
}

func (l *loopCounter) block(b *Block) int {
	if b == nil {
		return 0
	}
	n := 0
	for i := range b.Stmts {
		s := &b.Stmts[i]
		for j := range s.Exprs {
			n = addRuns(n, l.expr(&s.Exprs[j]))
		}
		switch s.Type {
		case For:
			trips, ok := forTrips(s)
			if !ok {
				l.bounded = false
				trips = math.MaxInt
			}
			n = addRuns(n, mulRuns(trips, addRuns(1, l.block(s.Blocks[0]))))
		case If:
			most := 0
			for _, c := range s.Blocks {
				most = max(most, l.block(c))
			}
			n = addRuns(n, most)
		default:
			for _, c := range s.Blocks {
				n = addRuns(n, l.block(c))
			}
		}
	}
	return n
}

// expr counts the loops of the functions e calls.
func (l *loopCounter) expr(e *Expr) int {
	n := 0
	if e.Type == FunctionExpr {
		n = l.call(e.Index)
	}
	for i := range e.Exprs {
		n = addRuns(n, l.expr(&e.Exprs[i]))
	}
	return n
}

func (l *loopCounter) call(f int) int {
	if n, ok := l.done[f]; ok {
		return n
	}
	if l.open[f] { // recursion, which the compiler refuses
		l.bounded = false
		return math.MaxInt
	}
	l.open[f] = true
	n := l.block(l.funcs[f])
	delete(l.open, f)
	l.done[f] = n
	return n
}

// forTrips returns how many times a for loop's body runs, if its bounds
// are constants and its counter reaches the end within its type.
func forTrips(s *Stmt) (int, bool) {
	if s.ForInit == nil || s.ForEnd == nil || s.ForDelta == nil {
		return 0, false
	}
	if s.ForVarType.Main == Float {
		return floatTrips(s)
	}
	init, ok1 := constant.Int64Val(constant.ToInt(s.ForInit))
	end, ok2 := constant.Int64Val(constant.ToInt(s.ForEnd))
	delta, ok3 := constant.Int64Val(constant.ToInt(s.ForDelta))
	if !ok1 || !ok2 || !ok3 {
		return 0, false
	}
	lo, hi := int64(math.MinInt32), int64(math.MaxInt32)
	if s.ForVarType.Main == Uint {
		lo, hi = 0, math.MaxUint32
	}
	if !holds(s.ForOp, float64(init), float64(end)) {
		return 0, true
	}
	var n int64
	d := end - init
	switch {
	case delta == 0:
		return 0, false
	case s.ForOp == EqualOp:
		n = 1
	case s.ForOp == NotEqualOp:
		if d%delta != 0 || d/delta < 0 {
			return 0, false
		}
		n = d / delta
	case (s.ForOp == LessThanOp || s.ForOp == LessThanEqualOp) != (delta > 0):
		return 0, false
	case s.ForOp == LessThanOp || s.ForOp == GreaterThanOp:
		n = (d + delta - sign(delta)) / delta // rounded up
	default:
		n = d/delta + 1
	}
	// The counter's last value, which ends the loop, must fit its type,
	// or it wraps around and the loop runs on.
	if last := init + n*delta; last < lo || last > hi {
		return 0, false
	}
	return int(n), true
}

func floatTrips(s *Stmt) (int, bool) {
	init, _ := constant.Float64Val(constant.ToFloat(s.ForInit))
	end, _ := constant.Float64Val(constant.ToFloat(s.ForEnd))
	delta, _ := constant.Float64Val(constant.ToFloat(s.ForDelta))
	x, e, d := float32(init), float32(end), float32(delta)
	for n := 0; n <= maxFloatSteps; n++ {
		if !holds(s.ForOp, float64(x), float64(e)) {
			return n, true
		}
		if x+d == x {
			return 0, false
		}
		x += d
	}
	if s.ForOp == EqualOp || s.ForOp == NotEqualOp || (s.ForOp == LessThanOp || s.ForOp == LessThanEqualOp) != (d > 0) {
		return 0, false
	}
	return int(math.Min(math.Ceil((end-init)/delta)+1, math.MaxInt)), true
}

func holds(op Op, a, b float64) bool {
	switch op {
	case LessThanOp:
		return a < b
	case LessThanEqualOp:
		return a <= b
	case GreaterThanOp:
		return a > b
	case GreaterThanEqualOp:
		return a >= b
	case EqualOp:
		return a == b
	case NotEqualOp:
		return a != b
	}
	return true
}

func sign(x int64) int64 {
	if x < 0 {
		return -1
	}
	return 1
}

func addRuns(a, b int) int {
	if a > math.MaxInt-b {
		return math.MaxInt
	}
	return a + b
}

func mulRuns(a, b int) int {
	if a != 0 && b > math.MaxInt/a {
		return math.MaxInt
	}
	return a * b
}
