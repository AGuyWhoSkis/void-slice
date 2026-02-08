package scan

func IsWhitespace(b byte) bool {
	// TODO: assess if \v \r \f are worth checking
	return b == ' ' || b == '\t' || b == '\r' || b == '\v' || b == '\f'
}

func IsDigit(b byte) bool {
	return b >= '0' && b <= '9'
}

func IsAlpha(b byte) bool {
	return (b >= 'a' && b <= 'z') ||
		(b >= 'A' && b <= 'Z')
}

func IsIdentStart(b byte) bool {
	return IsAlpha(b) || b == '_'
}

func IsIdentCont(b byte) bool {
	return IsIdentStart(b) || IsDigit(b)
}

func IsASCIIPunct(b byte) bool {
	return (b >= '!' && b <= '/') ||
		(b >= ':' && b <= '@') ||
		(b >= '[' && b <= '`') ||
		(b >= '{' && b <= '~')
}


func (li LineIndex) SpanPos(sp Span) SpanPos {
	return SpanPos{
		Span:  sp,
		Start: li.PosAt(sp.Start),
		End:   li.PosAt(sp.End),
	}
}

// Creates a slice of newline byte offsets, e.g. "\nABC\n" ==> []int{3, 4}
func BuildLineIndex(src []byte) LineIndex {
	li := LineIndex{Newlines: []int{}}
	for i, b := range src {
		if b == '\n' {
			li.Newlines = append(li.Newlines, i)
		}
	}
	return li
}

func (li LineIndex) PosAt(offset int) Pos {
	// Assumes li.Newlines contains byte offsets where src[i] == '\n', in ascending order.
	prevNLOffset := -1
	line := 1

	// To compute line/col:
	// line = number of newlines before offset + 1
	// col = offset - (last newline offset) (plus 1 for 1-based col)

	for _, nl := range li.Newlines {
		if nl >= offset {
			break
		}
		prevNLOffset = nl
		line++
	}

	if prevNLOffset == -1 {
		return Pos{Line: 1, Col: offset + 1}
	}
	// col is 1-based: the byte right after '\n' is col 1
	return Pos{Line: line, Col: offset - prevNLOffset}
}
