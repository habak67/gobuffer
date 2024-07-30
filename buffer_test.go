package gobuffer

import (
	"errors"
	"testing"
)

func TestBufferRollback_ZeroStateError(t *testing.T) {
	buf := New[rune]()
	buf.Write('a')
	buf.Consume()
	buf.Write('b')
	err := buf.Rollback(State{})
	if err == nil || err.Error() != ZeroStateError.Error() {
		t.Errorf("unexpected error rollback zero state: %s", err)
	}
	r, _ := buf.Next()
	if r != 'b' {
		t.Errorf("expected next to be 'b' (got %c)", r)
	}
}

func TestBufferRollback_IllegalStateError(t *testing.T) {
	buf := NewWithSize[rune](5, 4)
	for i := 0; i < 15; i++ {
		buf.Write('a')
	}
	state := buf.State()
	for i := 0; i < 10; i++ {
		buf.Consume()
	}
	buf.Commit()
	err := buf.Rollback(state)
	if err == nil || err.Error() != IllegalStateError.Error() {
		t.Errorf("unexpected error rollback illegal state: %v", err)
	}
}

func TestNewWithSize_ZeroRowSizePanic(t *testing.T) {
	defer func() { _ = recover() }()

	_ = NewWithSize[rune](0, 2)
	// Never reaches here if `OtherFunctionThatPanics` panics.
	t.Errorf("expected NewWithSize to panic")
}

func TestNewWithSize_NegativeRowSizePanic(t *testing.T) {
	defer func() { _ = recover() }()

	_ = NewWithSize[rune](-5, 2)
	// Never reaches here if `OtherFunctionThatPanics` panics.
	t.Errorf("expected NewWithSize to panic")
}

func TestNewWithSize_ZeroRowsPanic(t *testing.T) {
	defer func() { _ = recover() }()

	_ = NewWithSize[rune](10, 0)
	// Never reaches here if `OtherFunctionThatPanics` panics.
	t.Errorf("expected NewWithSize to panic")
}

func TestNewWithSize_NegativeRowsPanic(t *testing.T) {
	defer func() { _ = recover() }()

	_ = NewWithSize[rune](10, -1)
	// Never reaches here if `OtherFunctionThatPanics` panics.
	t.Errorf("expected NewWithSize to panic")
}

func TestBufferPos(t *testing.T) {
	rowSize := 10
	tests := []struct {
		name  string
		moves []int
		row   int
		col   int
		abs   int
	}{
		{"forward single one line", []int{5}, 0, 5, 5},
		{"forward full line", []int{10}, 1, 0, 10},
		{"forward multiple one line", []int{5, 2}, 0, 7, 7},
		{"forward multiple full line", []int{5, 5}, 1, 0, 10},
		{"forward wrap line", []int{5, 8}, 1, 3, 13},
		{"forward multiple wrap line", []int{5, 28}, 3, 3, 33},
		{"forward backward", []int{5, -3}, 0, 2, 2},
		{"forward backward wrap lines", []int{12, -3}, 0, 9, 9},
		{"forward backward multiple wrap lines", []int{32, -25}, 0, 7, 7},
		{"forward backward before start", []int{5, -8}, 0, 0, 0},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			b := position{rowSize: rowSize}
			for _, i := range test.moves {
				b = b.Move(i)
			}
			if b.Row != test.row {
				t.Errorf("unexpected row:\nexp=%d\ngot=%d", test.row, b.Row)
			}
			if b.Col != test.col {
				t.Errorf("unexpected index:\nexp=%d\ngot=%d", test.col, b.Col)
			}
			if b.AbsolutePos() != test.abs {
				t.Errorf("unexpected size:\nexp=%d\ngot=%d", test.abs, b.AbsolutePos())
			}
		})
	}
}

func TestBuffer(t *testing.T) {
	rowSize := 5
	rows := 1
	tests := []struct {
		name string
		ops  []any
	}{
		{
			"single write and next", []any{
				opWrite[rune]{Elem: 'a'},
				opNext[rune]{Exp: 'a'},
				opConsume{},
				opNextNotOk{},
			},
		},
		{
			"multiple not ok at end", []any{
				opWrite[rune]{Elem: 'a'},
				opNextAndConsume[rune]{Exp: 'a'},
				opNextNotOk{},
				opConsume{},
				opNextNotOk{},
				opConsume{},
				opNextNotOk{},
			},
		},
		{
			"no ok at empty buffer", []any{
				opConsume{},
				opNextNotOk{},
				opConsume{},
				opNextNotOk{},
				opWrite[rune]{Elem: 'a'},
				opNextAndConsume[rune]{Exp: 'a'},
				opNextNotOk{},
			},
		},
		{
			"consume when no next", []any{
				opWrite[rune]{Elem: 'a'},
				opNext[rune]{Exp: 'a'},
				opConsume{},
				opNextNotOk{},
				opConsume{},
				opNextNotOk{},
				opNextNotOk{},
				opConsume{},
				opNextNotOk{},
			},
		},
		{
			"multiple write and multiple next", []any{
				opWrite[rune]{Elem: 'a'},
				opWrite[rune]{Elem: 'b'},
				opNext[rune]{Exp: 'a'},
				opConsume{},
				opNext[rune]{Exp: 'b'},
				opNext[rune]{Exp: 'b'},
				opNext[rune]{Exp: 'b'},
				opConsume{},
				opNextNotOk{},
			},
		},
		{
			"next no unread elements", []any{
				opWrite[rune]{Elem: 'a'},
				opNextAndConsume[rune]{Exp: 'a'},
				opNextNotOk{},
				opNextNotOk{},
			},
		},
		{
			"grow buffer size on write", []any{
				opWrite[rune]{Elem: 'a'},
				opWrite[rune]{Elem: 'b'},
				opWrite[rune]{Elem: 'c'},
				opWrite[rune]{Elem: 'd'},
				opWrite[rune]{Elem: 'e'},
				opWrite[rune]{Elem: 'f'},
				opWrite[rune]{Elem: 'g'},
				opNextAndConsume[rune]{Exp: 'a'},
				opNextAndConsume[rune]{Exp: 'b'},
				opNextAndConsume[rune]{Exp: 'c'},
				opNextAndConsume[rune]{Exp: 'd'},
				opNextAndConsume[rune]{Exp: 'e'},
				opNextAndConsume[rune]{Exp: 'f'},
				opNextAndConsume[rune]{Exp: 'g'},
				opNextNotOk{},
			},
		},
		{
			"read after commit remove rows", []any{
				opWrite[rune]{Elem: '1'},
				opWrite[rune]{Elem: '2'},
				opWrite[rune]{Elem: '3'},
				opWrite[rune]{Elem: '4'},
				opWrite[rune]{Elem: '5'},
				opWrite[rune]{Elem: '6'},
				opWrite[rune]{Elem: '7'},
				opWrite[rune]{Elem: '8'},
				opWrite[rune]{Elem: '9'},
				opWrite[rune]{Elem: '0'},
				opWrite[rune]{Elem: '1'},
				opWrite[rune]{Elem: '2'},
				opNextAndConsume[rune]{Exp: '1'},
				opNextAndConsume[rune]{Exp: '2'},
				opNextAndConsume[rune]{Exp: '3'},
				opNextAndConsume[rune]{Exp: '4'},
				opNextAndConsume[rune]{Exp: '5'},
				opNextAndConsume[rune]{Exp: '6'},
				opNextAndConsume[rune]{Exp: '7'},
				opCommit{},
				opNextAndConsume[rune]{Exp: '8'},
				opWrite[rune]{Elem: 'a'},
				opNextAndConsume[rune]{Exp: '9'},
				opNextAndConsume[rune]{Exp: '0'},
				opNextAndConsume[rune]{Exp: '1'},
				opNextAndConsume[rune]{Exp: '2'},
				opNextAndConsume[rune]{Exp: 'a'},
				opNextNotOk{},
			},
		},
		{
			"state and rollback", []any{
				opWrite[rune]{Elem: 'a'},
				opWrite[rune]{Elem: 'b'},
				opWrite[rune]{Elem: 'c'},
				opCommit{},
				opNextAndConsume[rune]{Exp: 'a'},
				opState{},
				opNextAndConsume[rune]{Exp: 'b'},
				opNextAndConsume[rune]{Exp: 'c'},
				opWrite[rune]{Elem: 'd'},
				opRollback{},
				opWrite[rune]{Elem: 'e'},
				opNextAndConsume[rune]{Exp: 'b'},
				opNextAndConsume[rune]{Exp: 'c'},
				opNextAndConsume[rune]{Exp: 'd'},
				opNextAndConsume[rune]{Exp: 'e'},
				opNextNotOk{},
			},
		},
		{
			"rollback to non-existing position", []any{
				opWrites[rune]{Elem: 'a', Num: 20},
				opState{},
				opConsumes{Num: 15},
				opCommit{},
				opNext[rune]{Exp: 'a'},
				opRollback{Err: IllegalStateError},
			},
		},
		{
			"rollback after commit remove rows", []any{
				opWrite[rune]{Elem: '1'},
				opWrite[rune]{Elem: '2'},
				opWrite[rune]{Elem: '3'},
				opWrite[rune]{Elem: '4'},
				opWrite[rune]{Elem: '5'},
				opWrite[rune]{Elem: '6'},
				opWrite[rune]{Elem: '7'},
				opWrite[rune]{Elem: '8'},
				opWrite[rune]{Elem: '9'},
				opWrite[rune]{Elem: '0'},
				opWrite[rune]{Elem: '1'},
				opNextAndConsume[rune]{Exp: '1'},
				opNextAndConsume[rune]{Exp: '2'},
				opNextAndConsume[rune]{Exp: '3'},
				opNextAndConsume[rune]{Exp: '4'},
				opNextAndConsume[rune]{Exp: '5'},
				opNextAndConsume[rune]{Exp: '6'},
				opNextAndConsume[rune]{Exp: '7'},
				opCommit{},
				opState{},
				opNextAndConsume[rune]{Exp: '8'},
				opNextAndConsume[rune]{Exp: '9'},
				opRollback{},
				opNextAndConsume[rune]{Exp: '8'},
				opNextAndConsume[rune]{Exp: '9'},
				opNextAndConsume[rune]{Exp: '0'},
				opNextAndConsume[rune]{Exp: '1'},
				opNextNotOk{},
			},
		},
		{
			"buffered", []any{
				opWrite[rune]{Elem: 'a'},
				opWrite[rune]{Elem: 'b'},
				opBuffered{Exp: 2},
				opWrite[rune]{Elem: 'c'},
				opWrite[rune]{Elem: 'd'},
				opWrite[rune]{Elem: 'e'},
				opWrite[rune]{Elem: 'f'},
				opWrite[rune]{Elem: 'g'},
				opBuffered{Exp: 7},
				opNextAndConsume[rune]{Exp: 'a'},
				opNextAndConsume[rune]{Exp: 'b'},
				opNextAndConsume[rune]{Exp: 'c'},
				opBuffered{Exp: 4},
				opNextAndConsume[rune]{Exp: 'd'},
				opNextAndConsume[rune]{Exp: 'e'},
				opNextAndConsume[rune]{Exp: 'f'},
				opNextAndConsume[rune]{Exp: 'g'},
				opBuffered{Exp: 0},
				opNextNotOk{},
			},
		},
		{
			"buffered with only next", []any{
				opWrite[rune]{Elem: 'a'},
				opWrite[rune]{Elem: 'b'},
				opBuffered{Exp: 2},
				opNext[rune]{Exp: 'a'},
				opNext[rune]{Exp: 'a'},
				opNext[rune]{Exp: 'a'},
				opBuffered{Exp: 2},
				opNextAndConsume[rune]{Exp: 'a'},
				opNextAndConsume[rune]{Exp: 'b'},
				opBuffered{Exp: 0},
				opNextNotOk{},
			},
		},
		{
			"buffered with rollback", []any{
				opWrite[rune]{Elem: 'a'},
				opWrite[rune]{Elem: 'b'},
				opWrite[rune]{Elem: 'c'},
				opWrite[rune]{Elem: 'd'},
				opNextAndConsume[rune]{Exp: 'a'},
				opBuffered{Exp: 3},
				opState{},
				opNextAndConsume[rune]{Exp: 'b'},
				opNextAndConsume[rune]{Exp: 'c'},
				opBuffered{Exp: 1},
				opRollback{},
				opBuffered{Exp: 3},
				opNextAndConsume[rune]{Exp: 'b'},
				opNextAndConsume[rune]{Exp: 'c'},
				opNextAndConsume[rune]{Exp: 'd'},
				opNextNotOk{},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var state State
			buf := NewWithSize[rune](rowSize, rows)
			for i, o := range test.ops {
				switch op := o.(type) {
				case opNext[rune]:
					r, ok := buf.Next()
					if !ok {
						t.Errorf("[%d] unexpected read not ok", i)
					}
					if r != op.Exp {
						t.Errorf("[%d] unexpected rune read:\nexp=%c\ngot=%c", i, op.Exp, r)
					}
				case opNextNotOk:
					_, ok := buf.Next()
					if ok {
						t.Errorf("[%d] unexpected read ok", i)
					}
				case opNextAndConsume[rune]:
					r, ok := buf.Next()
					if !ok {
						t.Errorf("[%d] unexpected read not ok", i)
					}
					if r != op.Exp {
						t.Errorf("[%d] unexpected rune read:\nexp=%c\ngot=%c", i, op.Exp, r)
					}
					buf.Consume()
				case opWrite[rune]:
					buf.Write(op.Elem)
				case opWrites[rune]:
					for j := 0; j < op.Num; j++ {
						buf.Write(op.Elem)
					}
				case opConsume:
					buf.Consume()
				case opConsumes:
					for j := 0; j < op.Num; j++ {
						buf.Consume()
					}
				case opState:
					state = buf.State()
				case opRollback:
					err := buf.Rollback(state)
					if !errors.Is(err, op.Err) {
						t.Errorf("[%d] unexpected error:\nexp=%v\ngot=%v", i, op.Err, err)
					}
				case opCommit:
					buf.Commit()
				case opBuffered:
					n := buf.Buffered()
					if n != op.Exp {
						t.Errorf("[%d] unexpected buffered:\nexp=%d\ngot=%d", i, op.Exp, n)
					}
				}
			}
		})
	}
}

type opNext[T any] struct {
	Exp T
}

type opNextAndConsume[T any] struct {
	Exp T
}

type opNextNotOk struct{}

type opWrite[T any] struct {
	Elem T
}

type opWrites[T any] struct {
	Elem T
	Num  int
}

type opConsume struct{}

type opConsumes struct {
	Num int
}

type opState struct{}

type opRollback struct {
	Err error
}

type opCommit struct{}

type opBuffered struct {
	Exp int
}
