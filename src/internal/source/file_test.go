package source

import "testing"

func TestMapLineColumnUsesOneBasedPositions(t *testing.T) {
	sourceMap := NewMap("first\nsecond\nthird")

	tests := []struct {
		name       string
		offset     int
		wantLine   int
		wantColumn int
	}{
		{
			name:       "start of file",
			offset:     0,
			wantLine:   1,
			wantColumn: 1,
		},
		{
			name:       "middle of first line",
			offset:     2,
			wantLine:   1,
			wantColumn: 3,
		},
		{
			name:       "start of second line",
			offset:     6,
			wantLine:   2,
			wantColumn: 1,
		},
		{
			name:       "end of file",
			offset:     len("first\nsecond\nthird"),
			wantLine:   3,
			wantColumn: 6,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			line, column, ok := sourceMap.LineColumn(test.offset)
			if !ok {
				t.Fatal("expected offset to resolve")
			}
			if line != test.wantLine {
				t.Fatalf("expected line %d, got %d", test.wantLine, line)
			}
			if column != test.wantColumn {
				t.Fatalf("expected column %d, got %d", test.wantColumn, column)
			}
		})
	}
}

func TestMapLineColumnRejectsOutOfBoundsOffsets(t *testing.T) {
	sourceMap := NewMap("abc")

	for _, offset := range []int{-1, 4} {
		t.Run("offset", func(t *testing.T) {
			line, column, ok := sourceMap.LineColumn(offset)
			if ok {
				t.Fatalf("expected offset %d to be rejected, got %d:%d", offset, line, column)
			}
		})
	}
}

func TestNewFileStoresSourceMap(t *testing.T) {
	file := NewFile("project/src/main.puff", "main.puff", "on load\nend")

	if file.Path != "project/src/main.puff" {
		t.Fatalf("expected path %q, got %q", "project/src/main.puff", file.Path)
	}
	if file.RelPath != "main.puff" {
		t.Fatalf("expected rel path %q, got %q", "main.puff", file.RelPath)
	}
	if file.Text != "on load\nend" {
		t.Fatalf("expected text %q, got %q", "on load\nend", file.Text)
	}

	line, column, ok := file.Map.LineColumn(8)
	if !ok {
		t.Fatal("expected file source map to resolve offset")
	}
	if line != 2 || column != 1 {
		t.Fatalf("expected offset 8 to resolve to 2:1, got %d:%d", line, column)
	}
}
