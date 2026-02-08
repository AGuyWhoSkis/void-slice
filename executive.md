# Void Slice — Scanner + Tokenizer Scaffold

Current Goal: build a **reusable scanner pass** that produces precise diagnostics, and then add a **tokenizer** that turns bytes into a token stream with spans.

This doc is a scaffold of **interfaces, invariants, and checkpoints**

---

## 0) Deliverables for this stage

By the end of this stage these should be working:

1) A *single-pass* that:
- walks the input once
- tracks byte offsets + line/col
- records bracket/quote issues as `Diagnostic`s with `Span`

2) A **tokenizer** that:
- consumes input as `[]byte`and produces `[]Token`
- attaches a `Span` to every token
- *does not* crash on bad input (it emits diagnostics + best-effort tokens)

3) An **idTech linter** that can:
- print diagnostics and tokens in a readable way
- validate common patterns in .decl files
    - ex: check for `edit = {}` at start of file
    - ex: validate enum values, ie. `enumItem[ARK_CUE_ODOR]`
    - ex: warn on hash key modification, ie. `item_add["efIm-1"] = { .. }`
    - ex: validate array length `num = 3;` (self-declared) against actual number of items
- (goal) show the source slice for a span

---

## 1) Core data models

These are stable, boring, and extremely effective:

### 1.1 Span and position (v1)

Design decision:
- store byte offsets as integers, and use a line index to compute line/col for display

Minimal shape for building Scanner/Lexer/Debugger

```
type Span struct {
    Start int // byte offset, inclusive
    End   int // byte offset, exclusive
}

type Pos struct {
    Line int // 1-based
    Col  int // 1-based (byte offset)
}

type SpanPos struct {
    Span  Span
    Start Pos
    End   Pos
}
```

`Col` is measured in **bytes** not runes, since actual files are ASCII.

### 1.2 Diagnostic

Diagnostics must be consistent and easy to sort/deduplicate.

```
type Severity int // define some constants

type Diagnostic struct {
    Code     string   // define naming conventions
    Message  string
    Severity Severity
    Span     Span     // always include a span; if unknown, use zero-length at a best guess location
}
```

---

## 2) Line index utility

"offset -> line/col" will be needed *constantly* for diagnostics and token dumps.

### 2.1 Invariants

- Maintain a slice of newline byte offsets, e.g. indices of `\n`.
- To compute line/col:
  - line = number of newlines before offset + 1
  - col  = offset - (last newline offset)   (plus 1 if you want 1-based col)

### 2.2 Proposed interface

```
type LineIndex struct {
    // e.g. Newlines []int  // offsets of '\n'
}

func BuildLineIndex(src []byte) LineIndex
func (li LineIndex) PosAt(offset int) Pos
func (li LineIndex) SpanPos(sp Span) SpanPos
```

Checkpoint: Write a tiny unit test that feeds `"a\nbc\n"` and verifies positions at offsets 0..len.

---

## 3) Scanner pass (structural validation)

Since matching braces/quotes is not enough, a *library pass* is needed

### 3.1 Scope: what the scanner is responsible for

Scanner responsibilities:
- scan the file byte-by-byte into labelled Tokens
    - Token type: comment
    - Token type: quote literal, "e.g"
    - Token type: number literal, e.g. -123.321f
    - Token type: symbol/operator, e.g. brackets {}() or =;
    - Token type: code statement (ie. it's not a comment/quote/number/symbol)
- Scanner validates only these structures
    - missing semicolon
    - invalid char within number literal
    - unclosed "double quote or /* block comment

Scanner does *not* decide token kinds beyond what is needed for structural integrity.

### 3.2 Output

Scanner returns:
- `[]Diagnostic`
- (maybe) a small `ScanSummary` useful for the tokenizer:
  - e.g. "these spans are inside strings" or "unclosed quote at end"
  - Tokenizer can repeat string handling, but probably worth avoiding
    - option 1: Tokenizer owns string handling, Scanner focuses on bracket balancing
    - option 2: Scanner owns string state, Tokenizer queries it

A minimal signature:

```
type ScanResult struct {
    Diagnostics []Diagnostic
    Unclosed []UnclosedThing // Optional
}

func ScanStructure(src []byte) ScanResult
```

### 3.3 Error strategy

- On unexpected closer (e.g. `}` with empty stack), emit a diagnostic at that char’s span and continue.
- On EOF with unclosed openers, emit diagnostic per unclosed opener, spanning from opener to opener+1 (or opener..EOF).

Checkpoint: Feed obviously broken inputs and ensure *multiple* errors are produced (best effort)

---

## 4) Tokenizer (lexer) — the next step

Tokenizer turns bytes into tokens + diagnostics.

### 4.1 Key design choices (pick now)

1) **Input type**
- Recommend `[]byte` as the canonical internal representation
- Maybe accept `string` at the API boundary and convert once?

2) **Token stores raw slice or copies**
- Store `Span` and derive lexeme by slicing `src[Start:End]` when needed.
- This avoids allocations and is ideal for huge files.

3) **Whitespace handling**
- Most parsers skip whitespace tokens but still advance location.
- If lossless rewrite is later prioritized, whitespace/comments can be recorded as "trivia".
- For now skip whitespace, but keep span tracking correct.

### 4.2 Minimal token type

```
type Token struct {
    Kind TokenKind // you choose: int enum, string, iota constants, etc.
    Span Span
}
```

Token types
- punctuation kinds (braces, brackets, parens, equals, semicolon, comma)
- identifier
- string literal
- number
- maybe "unknown/bad" token (for recovery)

### 4.3 Tokenizer result

```
type LexResult struct {
    Tokens      []Token
    Diagnostics []Diagnostic
}

func Lex(src []byte) LexResult
```

---

## 5) Tokenizer algorithm scaffold

### 5.1 High-level loop

Pseudo-structure (not code):

1. `i := 0`
2. while `i < len(src)`:
   - if whitespace: consume run, `i = j`, continue
   - switch on `src[i]`:
     - if punctuation: emit token, `i++`
     - if quote: scan string literal
     - if digit or minus + digit: scan number
     - if identifier start: scan identifier
     - else: emit diagnostic + either:
       - emit Unknown token for that char; `i++`
       - or skip it; `i++`

### 5.2 String literal scanning (important)

Decide rules:
- Are newlines allowed inside strings? (probably not)
- Escape sequences: is `\"` an escaped quote? (likely yes)
- If EOF occurs before closing quote: emit diagnostic spanning from opening quote to EOF; emit a token anyway (span = opener..EOF) so parser can continue.

### 5.3 Number scanning (good enough first pass)

Decide:
- floats with decimal point
- valid type suffixes (0.2f, 0m, etc.)
- optional exponent (e.g. `1e-3`) — maybe not needed but cheap to support
- negative numbers: is `-` always the start of a number?

### 5.4 Identifier scanning

Decide:
- allow `_` (yes)
- allow digits after first char (yes)
- allow `.`? (depends on syntax) (maybe? TODO)

---

## 6) Diagnostics + spans for errors

Early diagnostics
- invalid character
- unterminated string
- invalid escape sequence (optional)
- malformed number (optional)

Every diagnostic should have:
- `Span` pointing to the smallest relevant range (often 1 char, or the whole bad literal)

---

## 7) Helpers

### 7.1 Small helper predicates

- `isSpace(b byte) bool`
- `isIdentStart(b byte) bool`
- `isIdentCont(b byte) bool`
- `isDigit(b byte) bool`

### 7.2 Tiny cursor type (optional)

A cursor can hide boundary checks and give you `peek()` / `advance()`.

```
type Cursor struct {
    Src []byte
    I   int
}

func (c *Cursor) AtEnd() bool
func (c *Cursor) Peek() byte          // undefined if AtEnd
func (c *Cursor) Advance() byte       // returns consumed byte
func (c *Cursor) SpanFrom(start int) Span
```

Not strictly necessary, but might help tidy the main loop

### 7.3 A "readWhile" helper

Implement a helper that consumes while predicate true:

```
func readWhile(src []byte, i int, pred func(byte) bool) (j int)
```

This turns identifier/number/whitespace into one-liners.
---
