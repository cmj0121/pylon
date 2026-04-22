package pylon

import "testing"

func TestNewSourceEmpty(t *testing.T) {
	s := NewSource("")
	p := s.PositionAt(0)
	if p != (Position{Offset: 0, Line: 0, Column: 0}) {
		t.Fatalf("empty PositionAt(0)=%+v", p)
	}
}

func TestPositionAtSingleLine(t *testing.T) {
	s := NewSource("hello")
	cases := []struct {
		at   int
		want Position
	}{
		{0, Position{0, 0, 0}},
		{3, Position{3, 0, 3}},
		{5, Position{5, 0, 5}},
	}
	for _, c := range cases {
		got := s.PositionAt(c.at)
		if got != c.want {
			t.Errorf("PositionAt(%d)=%+v want %+v", c.at, got, c.want)
		}
	}
}

func TestPositionAtMultiLine(t *testing.T) {
	// "abc\ndef\nghi"  -> bytes: 0:a 1:b 2:c 3:\n 4:d 5:e 6:f 7:\n 8:g 9:h 10:i
	s := NewSource("abc\ndef\nghi")
	cases := []struct {
		at   int
		want Position
	}{
		{0, Position{0, 0, 0}},    // 'a'
		{3, Position{3, 0, 3}},    // '\n' still line 0, column 3
		{4, Position{4, 1, 0}},    // 'd' first byte of line 1
		{7, Position{7, 1, 3}},    // second '\n' — end of line 1
		{8, Position{8, 2, 0}},    // 'g'
		{10, Position{10, 2, 2}},  // 'i'
		{11, Position{11, 2, 3}},  // one-past-last
		{999, Position{11, 2, 3}}, // clamped
	}
	for _, c := range cases {
		got := s.PositionAt(c.at)
		if got != c.want {
			t.Errorf("PositionAt(%d)=%+v want %+v", c.at, got, c.want)
		}
	}
}

func TestSubInheritsPositions(t *testing.T) {
	parent := NewSource("abc\ndef\nghi")
	sub := parent.Sub(4, 11) // "def\nghi"
	// Sub's local 0 is parent's absolute 4.
	if got, want := sub.PositionAt(0), parent.PositionAt(4); got != want {
		t.Errorf("sub.PositionAt(0)=%+v parent.PositionAt(4)=%+v", got, want)
	}
	// Sub's local 4 ('g') is parent's absolute 8.
	if got, want := sub.PositionAt(4), parent.PositionAt(8); got != want {
		t.Errorf("sub.PositionAt(4)=%+v parent.PositionAt(8)=%+v", got, want)
	}
	// View() on the sub is the narrowed slice.
	if got, want := sub.View(), "def\nghi"; got != want {
		t.Errorf("sub.View()=%q want %q", got, want)
	}
}

func TestUTF16ColumnCJK(t *testing.T) {
	// "[ 你好世界 ]"
	//   [  sp 你 好 世 界 sp ]
	//   idx(bytes): 0  1  2     5     8     11    14 15
	//   idx(utf16): 0  1  2     3     4     5     6  7
	// each CJK rune is 3 bytes UTF-8, 1 UTF-16 unit.
	src := "[ 你好世界 ]"
	s := NewSource(src)
	// Position of the closing ']' (last byte in this ASCII-tail case).
	closeBytePos := len(src) - 1
	got := s.PositionAt(closeBytePos)
	if got.Column != 7 {
		t.Errorf("CJK ']' Column=%d want 7 (bytes=%d)", got.Column, closeBytePos)
	}
	if got.Line != 0 {
		t.Errorf("CJK ']' Line=%d want 0", got.Line)
	}
}

func TestUTF16ColumnSupplementaryPlane(t *testing.T) {
	// "[ 😀 ]"  —  😀 is U+1F600, 4 bytes UTF-8, 2 UTF-16 units (surrogate pair).
	// Columns:  [=0  sp=1  😀 spans columns 2..3 (2 units)  sp=4  ]=5
	src := "[ 😀 ]"
	s := NewSource(src)
	closeBytePos := len(src) - 1
	got := s.PositionAt(closeBytePos)
	if got.Column != 5 {
		t.Errorf("emoji ']' Column=%d want 5", got.Column)
	}
}

func TestPositionAtCRLF(t *testing.T) {
	// "abc\r\ndef\r\n" — linePrefix must index ORIGINAL bytes (CR included).
	// Bytes:  0:a 1:b 2:c 3:\r 4:\n 5:d 6:e 7:f 8:\r 9:\n
	// Lines:  line 0 spans [0, 5); line 1 spans [5, 10); line 2 spans [10, ...).
	s := NewSource("abc\r\ndef\r\n")
	if got, want := s.PositionAt(5), (Position{5, 1, 0}); got != want {
		t.Errorf("PositionAt('d')=%+v want %+v", got, want)
	}
	// The '\r' byte at offset 3 is still on line 0, column 3.
	if got, want := s.PositionAt(3), (Position{3, 0, 3}); got != want {
		t.Errorf("PositionAt('\\r')=%+v want %+v", got, want)
	}
}

func TestSpanOf(t *testing.T) {
	s := NewSource("abc\ndef")
	got := s.SpanOf(1, 5) // "bc\nd"
	want := Span{Start: Position{1, 0, 1}, End: Position{5, 1, 1}}
	if got != want {
		t.Errorf("SpanOf(1,5)=%+v want %+v", got, want)
	}
}
