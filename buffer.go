package gobuffer

import (
	"errors"
	"fmt"
)

// position holds the position in a two-dimensional space (like a Buffer) consisting of rows and columns.
// The first position (zero position) is row 0 and column 0.
type position struct {
	rowSize int
	Row     int
	Col     int
}

// Move will move the current position the specified number of steps. The moved position is returned. If steps
// is > 0 then the position is moved forward and if steps < 0 then the position is moved backward. Note that the
// lowest position is 0/0, and you may not move before that position. That is, if the current position is 0/3 and
// steps is -5 then the new position will be 0/0.
func (p position) Move(steps int) position {
	absPos := p.AbsolutePos() + steps
	if absPos < 0 {
		absPos = 0
	}
	p.Row = absPos / p.rowSize
	p.Col = absPos % p.rowSize
	return p
}

// AbsolutePos returns the absolute (one-dimensional) position. Note that position uses the standard array index
// convention where the first position is 0. Example of absolute positions (row size is assumed to be 10).
//
//	Line: 0, Idx: 0 => 0
//	Line: 0, Idx: 5 => 5
//	Line: 2, Idx: 8 => 28
func (p position) AbsolutePos() int {
	return p.Row*p.rowSize + p.Col
}

var IllegalStateError = errors.New("rollback position doesn't exist")
var ZeroStateError = errors.New("illegal non-initialized state")

// State holds a state for a Buffer. It could be used to roll back to a previously saved state.
type State struct {
	read  position
	write position
	init  bool
}

func (s State) zero() bool {
	return !s.init
}

func newState(read, write position) State {
	return State{
		read:  read,
		write: write,
		init:  true,
	}
}

// Buffer is a dynamic FIFO Buffer holding elements of the specified type. The Buffer grows with increments of
// the configured row size. The Buffer supports read, write, unread and rollback to a previously collected state.
//
// You may "unread" a buffer as many times as there has been writes to the buffer. To mitigate a Buffer to grow
// infinitely a Buffer may be committed to remove previously read elements. After a commit you may only unread
// elements that has been written after the last commit (technically you may unread elements to the beginning
// of the first row after the commit).
//
// Even if a buffer technically may be indefinitely big the implementation is by no means optimized for bigger
// buffers. Instead the buffer is developed to hold smaller number of elements at the same time (between commits).
type Buffer[T any] struct {
	rowSize  int
	startRow int // startRow holds the row number of the first row in the buffer.
	buffers  [][]T
	read     position // read points to the next element to read from the Buffer.
	write    position // write points to the position where the next element should be written.
}

// Read reads the next element from the Buffer. If there are no elements to read false is returned.
func (b *Buffer[T]) Read() (element T, ok bool) {
	if b.buffered() <= 0 {
		return
	}
	ok = true
	row, col := b.bufferPos(b.read)
	element = b.buffers[row][col]
	b.read = b.read.Move(1)
	return
}

// Unread will unread the last read element from the Buffer. The next element to read will be the previously
// read element.
func (b *Buffer[T]) Unread() {
	b.read = b.read.Move(-1)
}

// Write writes an element to the Buffer. If needed the Buffer is grown to hold the element.
func (b *Buffer[T]) Write(element T) {
	b.Grow(b.write.AbsolutePos() + 1)
	row, col := b.bufferPos(b.write)
	b.buffers[row][col] = element
	b.write = b.write.Move(1)
}

// State return a Buffer state. The state may be used to backtrack to the current state.
func (b *Buffer[T]) State() State {
	return newState(b.read, b.write)
}

// Rollback resets the Buffer read state to the provided state. The next element to read is the one that was the next
// element to read when the state was created. Note that writes are not reset. That is, any writes done after the
// state was collected will still be available in the Buffer.
//
// If the provided state is the "zero state" (has not been created by method State) then an ZeroStateError is returned.
//
// If rollback to a state that was created before the last commit then the rollback read position may not exist
// anymore. If such the case an IllegalStateError is returned. To mitigate such errors it is recommended only
// rollback to states created after the last commit.
func (b *Buffer[T]) Rollback(state State) error {
	if state.zero() {
		return ZeroStateError
	}
	// Check if state is still valid (not created before a call to commit)
	if state.read.Row < b.startRow {
		return IllegalStateError
	}
	b.read = state.read
	return nil
}

// Commit removes Buffer rows before the current read pointer. This will mitigate the Buffer to grow indefinitely.
func (b *Buffer[T]) Commit() {
	// Cleanup unreachable Buffer rows
	row, _ := b.bufferPos(b.read)
	b.buffers = b.buffers[row-b.startRow:]
	b.startRow += row
}

// Grow will grow the Buffer to be able to hold at least the specified number of elements.
func (b *Buffer[T]) Grow(size int) {
	rows := size / b.rowSize
	for i := len(b.buffers); i <= rows; i++ {
		b.buffers = append(b.buffers, make([]T, b.rowSize))
	}
}

// buffered returns the number of unread elements in the Buffer.
func (b *Buffer[T]) buffered() int {
	return b.write.AbsolutePos() - b.read.AbsolutePos()
}

func (b *Buffer[T]) bufferPos(pos position) (row, col int) {
	return pos.Row - b.startRow, pos.Col
}

// New creates a new Buffer holding objects of the specified type.
func New[T any]() (buf *Buffer[T]) {
	buf, _ = NewWithSize[T](10, 5)
	return
}

// NewWithSize creates a new Buffer with the specified row size. The Buffer is pre-allocated with the specified
// number of rows.
func NewWithSize[T any](rowSize, rows int) (buf *Buffer[T], err error) {
	if rowSize <= 0 {
		return nil, fmt.Errorf("illegal non-positive row size %d", rowSize)
	}
	buf = &Buffer[T]{
		rowSize: rowSize,
		buffers: make([][]T, 0, rows),
		read:    position{rowSize: rowSize},
		write:   position{rowSize: rowSize},
	}
	buf.Grow(rows * rowSize)
	return
}
