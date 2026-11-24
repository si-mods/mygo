package shaderir

import (
	"go/constant"
	"math"
)

// maxFloatSteps is how many steps of a float counter LoopRuns takes, in
// float32 as the GPU and the CPU's interpreter add, to count a loop; a
// longer one is unbounded, as one whose counter stops growing in float32.
const maxFloatSteps = 1 << 16

// LoopRuns reports the most times the loop bodies of the fragment entry
// point run for one pixel, every loop's counted: a loop of n runs inside
// one of m adds m + m*n. A loop in a function counts at each call, times
// the loops around it; the larger branch of an if counts. bounded is
// false when a loop's start or end is not a constant, its body writes its
// counter, or its counter never reaches the end (wraps around, stops
// growing in float32, or takes more than maxFloatSteps float steps); runs
// saturate at math.MaxInt.
func (p *Program) LoopRuns() (runs int, bounded bool) {
	funcs := map[int]*Func{}
	for i := range p.Funcs {
		funcs[p.Funcs[i].Index] = &p.Funcs[i]
	}
	l := loopCounter{funcs: funcs, done: map[int]int{}, open: map[int]bool{}, bounded: true}
	runs = l.block(p.FragmentFunc.Block)
	return runs, l.bounded
}

type loopCounter struct {
	funcs   map[int]*Func
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
			if !ok || l.writes(s.Blocks[0], s.ForVarIndex) {
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

// writes reports whether b writes the local variable v, a loop's counter:
// assigns it, or passes it as a function's out-param.
func (l *loopCounter) writes(b *Block, v int) bool {
	if b == nil {
		return false
	}
	for i := range b.Stmts {
		s := &b.Stmts[i]
		if s.Type == Assign && len(s.Exprs) > 0 && rootLocal(&s.Exprs[0]) == v {
			return true
		}
		if s.Type == Init && s.InitIndex == v {
			return true
		}
		for j := range s.Exprs {
			if l.outArg(&s.Exprs[j], v) {
				return true
			}
		}
		for _, c := range s.Blocks {
			if l.writes(c, v) {
				return true
			}
		}
	}
	return false
}

// outArg reports whether e passes the local variable v to a call as an
// out-param.
func (l *loopCounter) outArg(e *Expr, v int) bool {
	if e.Type == Call && len(e.Exprs) > 0 && e.Exprs[0].Type == FunctionExpr {
		in := 0
		if fn := l.funcs[e.Exprs[0].Index]; fn != nil {
			in = len(fn.InParams)
		}
		for i := 1 + in; i < len(e.Exprs); i++ {
			if rootLocal(&e.Exprs[i]) == v {
				return true
			}
		}
	}
	for i := range e.Exprs {
		if l.outArg(&e.Exprs[i], v) {
			return true
		}
	}
	return false
}

// rootLocal returns the local variable e is, or is an element, a field or
// a swizzle of, or -1.
func rootLocal(e *Expr) int {
	for (e.Type == FieldSelector || e.Type == Index || e.Type == SwizzlingExpr) && len(e.Exprs) > 0 {
		e = &e.Exprs[0]
	}
	if e.Type == LocalVariable {
		return e.Index
	}
	return -1
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
	var n int
	if fn := l.funcs[f]; fn != nil {
		n = l.block(fn.Block)
	}
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
	return 0, false
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
