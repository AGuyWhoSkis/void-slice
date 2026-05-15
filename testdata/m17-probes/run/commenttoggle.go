package main

// Family: commenttoggle.
//
// V0 wraps the first line that contains an `=` (a curly-shape assignment)
// with a `// ` prefix. The line goes from `m_field = "foo";` to
// `// m_field = "foo";` — converting an assignment into a comment.
// Diagnostic delta should be at most "the type-check for that field
// disappears" — anything more is surprise.
//
// V1 unwraps the first single-line `// ...` line by stripping the `// `
// prefix. Logical opposite of V0 on a different starting state.

import (
	"bytes"
	"fmt"
	"os"
	"strings"
)

func init() { register("commenttoggle", runCommentToggle) }

func runCommentToggle(_ []string) {
	paths := walkCorpus("testdata/golden")
	fmt.Printf("# commenttoggle: %d files\n", len(paths))
	found := 0
	for _, p := range paths {
		src, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		base := lintCodes(p, src)

		// V0: prefix first assignment-bearing line with `// `.
		if start, end := firstAssignmentLine(src); start >= 0 {
			mut := make([]byte, 0, len(src)+3)
			mut = append(mut, src[:start]...)
			mut = append(mut, []byte("// ")...)
			mut = append(mut, src[start:]...)
			_ = end
			vari := lintCodes(p, mut)
			if !sameMultiset(base, vari) {
				emit(p, "commenttoggle", 0, start, start+3, diffMultisets(base, vari), "comment-out-assignment")
				found++
			}
		}

		// V1: strip leading `// ` from the first `// `-prefixed line.
		if start := firstLineCommentPrefix(src); start >= 0 {
			mut := make([]byte, 0, len(src))
			mut = append(mut, src[:start]...)
			mut = append(mut, src[start+3:]...)
			vari := lintCodes(p, mut)
			if !sameMultiset(base, vari) {
				emit(p, "commenttoggle", 1, start, start+3, diffMultisets(base, vari), "uncomment-line")
				found++
			}
		}
	}
	fmt.Printf("# commenttoggle: %d findings\n", found)
}

// firstAssignmentLine returns (lineStart, lineEnd) for the first line
// containing an `=` byte that is not already inside a comment or quoted
// literal. Returns (-1, -1) if no such line.
func firstAssignmentLine(src []byte) (int, int) {
	for off := 0; off < len(src); {
		end := bytes.IndexByte(src[off:], '\n')
		if end < 0 {
			end = len(src) - off
		}
		line := src[off : off+end]
		if i := bytes.IndexByte(line, '='); i >= 0 {
			// Quick filter: skip lines that look like block-comment bodies
			// (start with leading spaces then `*`) or inside `// ` prefix.
			trim := strings.TrimLeft(string(line), " \t")
			if !strings.HasPrefix(trim, "//") && !strings.HasPrefix(trim, "*") && !strings.HasPrefix(trim, "/*") {
				return off, off + end
			}
		}
		off += end + 1
	}
	return -1, -1
}

// firstLineCommentPrefix returns the offset of the `/` byte of the first
// `// ` line-comment prefix in the file (only counts the `// ` form so
// the V1 unwrap is well-defined). Returns -1 if no such occurrence.
func firstLineCommentPrefix(src []byte) int {
	for off := 0; off+2 < len(src); {
		end := bytes.IndexByte(src[off:], '\n')
		if end < 0 {
			end = len(src) - off
		}
		line := src[off : off+end]
		trim := bytes.TrimLeft(line, " \t")
		if bytes.HasPrefix(trim, []byte("// ")) {
			return off + (len(line) - len(trim))
		}
		off += end + 1
	}
	return -1
}
