package source

import "sort"

type File struct {
	Path    string
	RelPath string
	Text    string
	Map     Map
}

func NewFile(path string, relPath string, text string) File {
	return File{
		Path:    path,
		RelPath: relPath,
		Text:    text,
		Map:     NewMap(text),
	}
}

type Map struct {
	lineStarts []int
	size       int
}

func NewMap(text string) Map {
	lineStarts := []int{0}

	for offset, char := range text {
		if char == '\n' {
			lineStarts = append(lineStarts, offset+1)
		}
	}

	return Map{
		lineStarts: lineStarts,
		size:       len(text),
	}
}

func (sourceMap Map) LineColumn(offset int) (int, int, bool) {
	if offset < 0 || offset > sourceMap.size {
		return 0, 0, false
	}

	lineIndex := sort.Search(len(sourceMap.lineStarts), func(i int) bool {
		return sourceMap.lineStarts[i] > offset
	}) - 1
	if lineIndex < 0 {
		return 0, 0, false
	}

	line := lineIndex + 1
	column := offset - sourceMap.lineStarts[lineIndex] + 1

	return line, column, true
}
