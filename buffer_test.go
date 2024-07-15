package gobuffer

import (
	"errors"
	"testing"
)

func TestBufferRollback_ZeroStateError(t *testing.T) {
	buf := New[rune]()
	buf.Write('a')
	_, _ = buf.Read()
	buf.Write('b')
	err := buf.Rollback(State{})
	if err == nil || err.Error() != ZeroStateError.Error() {
		t.Errorf("unexpected error rollback zero state: %s", err)
	}
	r, _ := buf.Read()
	if r != 'b' {
		t.Errorf("expected read 'b' (got %c)", r)
	}
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
			"single write and read", []any{
				opWrite[rune]{Elem: 'a'},
				opRead[rune]{Exp: 'a'},
				opEOF{},
			},
		},
		{
			"multiple write and read", []any{
				opWrite[rune]{Elem: 'a'},
				opWrite[rune]{Elem: 'b'},
				opWrite[rune]{Elem: 'c'},
				opRead[rune]{Exp: 'a'},
				opRead[rune]{Exp: 'b'},
				opRead[rune]{Exp: 'c'},
				opEOF{},
			},
		},
		{
			"read no unread", []any{
				opWrite[rune]{Elem: 'a'},
				opRead[rune]{Exp: 'a'},
				opReadNotOk{},
				opEOF{},
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
				opRead[rune]{Exp: 'a'},
				opRead[rune]{Exp: 'b'},
				opRead[rune]{Exp: 'c'},
				opRead[rune]{Exp: 'd'},
				opRead[rune]{Exp: 'e'},
				opRead[rune]{Exp: 'f'},
				opRead[rune]{Exp: 'g'},
				opEOF{},
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
				opRead[rune]{Exp: '1'},
				opRead[rune]{Exp: '2'},
				opRead[rune]{Exp: '3'},
				opRead[rune]{Exp: '4'},
				opRead[rune]{Exp: '5'},
				opRead[rune]{Exp: '6'},
				opRead[rune]{Exp: '7'},
				opCommit{},
				opRead[rune]{Exp: '8'},
				opWrite[rune]{Elem: 'a'},
				opRead[rune]{Exp: '9'},
				opRead[rune]{Exp: '0'},
				opRead[rune]{Exp: '1'},
				opRead[rune]{Exp: '2'},
				opRead[rune]{Exp: 'a'},
				opEOF{},
			},
		},
		{
			"single unread", []any{
				opWrite[rune]{Elem: 'a'},
				opWrite[rune]{Elem: 'b'},
				opRead[rune]{Exp: 'a'},
				opUnread{},
				opRead[rune]{Exp: 'a'},
				opRead[rune]{Exp: 'b'},
				opEOF{},
			},
		},
		{
			"multiple unread", []any{
				opWrite[rune]{Elem: 'a'},
				opWrite[rune]{Elem: 'b'},
				opRead[rune]{Exp: 'a'},
				opWrite[rune]{Elem: 'c'},
				opUnread{},
				opUnread{},
				opRead[rune]{Exp: 'a'},
				opRead[rune]{Exp: 'b'},
				opRead[rune]{Exp: 'c'},
				opEOF{},
			},
		},
		{
			"state and rollback", []any{
				opWrite[rune]{Elem: 'a'},
				opWrite[rune]{Elem: 'b'},
				opWrite[rune]{Elem: 'c'},
				opCommit{},
				opRead[rune]{Exp: 'a'},
				opState{},
				opRead[rune]{Exp: 'b'},
				opRead[rune]{Exp: 'c'},
				opWrite[rune]{Elem: 'd'},
				opRollback{},
				opWrite[rune]{Elem: 'e'},
				opRead[rune]{Exp: 'b'},
				opRead[rune]{Exp: 'c'},
				opRead[rune]{Exp: 'd'},
				opRead[rune]{Exp: 'e'},
				opEOF{},
			},
		},
		{
			"rollback to non-existing position", []any{
				opWrites[rune]{Elem: 'a', Num: 20},
				opState{},
				opReads{Num: 15},
				opCommit{},
				opRead[rune]{Exp: 'a'},
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
				opRead[rune]{Exp: '1'},
				opRead[rune]{Exp: '2'},
				opRead[rune]{Exp: '3'},
				opRead[rune]{Exp: '4'},
				opRead[rune]{Exp: '5'},
				opRead[rune]{Exp: '6'},
				opRead[rune]{Exp: '7'},
				opCommit{},
				opState{},
				opRead[rune]{Exp: '8'},
				opRead[rune]{Exp: '9'},
				opRollback{},
				opRead[rune]{Exp: '8'},
				opRead[rune]{Exp: '9'},
				opRead[rune]{Exp: '0'},
				opRead[rune]{Exp: '1'},
				opEOF{},
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
				opRead[rune]{Exp: 'a'},
				opRead[rune]{Exp: 'b'},
				opRead[rune]{Exp: 'c'},
				opBuffered{Exp: 4},
				opRead[rune]{Exp: 'd'},
				opRead[rune]{Exp: 'e'},
				opRead[rune]{Exp: 'f'},
				opRead[rune]{Exp: 'g'},
				opEOF{},
			},
		},
		{
			"buffered with rollback", []any{
				opWrite[rune]{Elem: 'a'},
				opWrite[rune]{Elem: 'b'},
				opWrite[rune]{Elem: 'c'},
				opWrite[rune]{Elem: 'd'},
				opRead[rune]{Exp: 'a'},
				opBuffered{Exp: 3},
				opState{},
				opRead[rune]{Exp: 'b'},
				opRead[rune]{Exp: 'c'},
				opBuffered{Exp: 1},
				opRollback{},
				opBuffered{Exp: 3},
				opRead[rune]{Exp: 'b'},
				opRead[rune]{Exp: 'c'},
				opRead[rune]{Exp: 'd'},
				opEOF{},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var state State
			buf, err := NewWithSize[rune](rowSize, rows)
			if err != nil {
				t.Errorf("unexpected error creating Buffer: %s", err)
			}
			for i, o := range test.ops {
				switch op := o.(type) {
				case opRead[rune]:
					r, ok := buf.Read()
					if !ok {
						t.Errorf("[%d] unexpected read not ok", i)
					}
					if r != op.Exp {
						t.Errorf("[%d] unexpected rune read:\nexp=%c\ngot=%c", i, op.Exp, r)
					}
				case opReadNotOk:
					_, ok := buf.Read()
					if ok {
						t.Errorf("[%d] unexpected read ok", i)
					}
				case opWrite[rune]:
					buf.Write(op.Elem)
				case opWrites[rune]:
					for j := 0; j < op.Num; j++ {
						buf.Write(op.Elem)
					}
				case opReads:
					for j := 0; j < op.Num; j++ {
						_, ok := buf.Read()
						if !ok {
							t.Errorf("[%d] unexpected read not ok", i)
						}
					}
				case opUnread:
					buf.Unread()
				case opState:
					state = buf.State()
				case opRollback:
					err = buf.Rollback(state)
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
				case opEOF:
					// EOF means no elements to read in Buffer
					if buf.Buffered() != 0 {
						t.Errorf("[%d] expected empty Buffer (%d)", i, buf.Buffered())
					}
				}
			}
		})
	}
}

type opRead[T any] struct {
	Exp T
	Ok  bool
	Err error
}

type opReadNotOk struct{}

type opWrite[T any] struct {
	Elem T
}

type opWrites[T any] struct {
	Elem T
	Num  int
}

type opReads struct {
	Num int
}

type opUnread struct{}

type opState struct{}

type opRollback struct {
	Err error
}

type opCommit struct{}

type opBuffered struct {
	Exp int
}

type opEOF struct{}
