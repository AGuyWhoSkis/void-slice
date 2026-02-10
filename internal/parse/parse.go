package parse

// Planned: build a token-driven parser that can *walk* .entities structure while:
//   - emitting structural diagnostics (missing braces/semicolon, unexpected token, etc.)
//   - enabling validate package to do semantic checks (array num/item parity, required keys)
//
// IMPORTANT DESIGN CONSTRAINTS:
//   - Option A punctuation: keep tokens.Kind==SYMBOL and inspect src[t.Span.Start].
//   - Huge files: parsing should be streaming-ish; do not build a full-file AST.
//     Parse one component at a time; allow validator to process and discard.
//
// -------------------------
// 1) Public API (parse package)
// -------------------------
//
// Choose ONE of these shapes (A or B). A is simplest and is the recommended start.
//
// A) “Walk” API (recommended for streaming validation):
//
//   type Handler interface {
//     OnVersion(versionTok scan.Token, versionValue int64)
//     OnComponentBegin(componentTok scan.Token, lbrace scan.Token)
//     OnComponentDecl(typeTok, nameTok scan.Token, lbrace scan.Token) // e.g. cpntFoo name {
//     OnComponentEnd(rbrace scan.Token)
//
//     OnObjectBegin(lbrace scan.Token) // for nested { ... }
//     OnObjectEnd(rbrace scan.Token)
//
//     OnAssignment(key Key, eqTok scan.Token, value Value, semiTok scan.Token)
//     // Optional: OnTypedBlock(typeTok, nameTok, lbrace) if a distinct hook is helpful.
//
//     OnDiag(diag scan.Diagnostic)
//   }
//
//   func WalkEntities(src []byte, toks []scan.Token, h Handler) (diags []scan.Diagnostic)
//
// Notes:
//   - WalkEntities may *also* return diags; Handler.OnDiag is optional redundancy.
//   - validate will implement Handler to track object frames and run checks at ObjectEnd.
//
// B) “Parse component” API (still streaming, slightly more AST):
//
//   type Component struct { /* minimal fields: type/name token spans, root object range */ }
//   func ParseEntities(src []byte, toks []scan.Token) (components []Component, diags []scan.Diagnostic)
//
// If approach B is used, validate iterates components, then re-walks tokens within that span.
// Approach A avoids re-walking and is simpler overall.
//
// -------------------------
// 2) Cursor / token stream helper
// -------------------------
//
// Implement TokenCursor (private to parse):
//
//   type cursor struct {
//     src   []byte
//     toks  []scan.Token
//     i     int
//     diags []scan.Diagnostic
//   }
//
//   func (c *cursor) eof() bool
//   func (c *cursor) peek() (scan.Token, bool)
//   func (c *cursor) next() (scan.Token, bool)
//   func (c *cursor) lexeme(t scan.Token) []byte  // slice: src[start:end]
//   func (c *cursor) sym(t scan.Token) byte       // src[start]
//   func (c *cursor) isIdent(t scan.Token, lit string) bool // no allocations
//
// Expect/match helpers (these should be the only places that create parse diags):
//   func (c *cursor) matchKind(k scan.Kind) (scan.Token, bool)
//   func (c *cursor) expectKind(k scan.Kind, code scan.DiagnosticCode, msg string) (scan.Token, bool)
//
//   func (c *cursor) matchSym(ch byte) (scan.Token, bool)
//   func (c *cursor) expectSym(ch byte, code scan.DiagnosticCode, msg string) (scan.Token, bool)
//
//   func (c *cursor) matchIdent(lit string) (scan.Token, bool)
//   func (c *cursor) expectIdent(lit string, code scan.DiagnosticCode, msg string) (scan.Token, bool)
//
// Recovery (critical for “keep going”):
//   func (c *cursor) syncTo(sym1 byte, sym2 byte /* optional */) // advance until one found
//   Recovery rules used most:
//     - Inside object: sync to ';' or '}'.
//     - At top-level: sync to 'component' or EOF.
//
// -------------------------
// 3) Minimal parse data types
// -------------------------
//
// Key + indexers (enough for “item[0]” and “item_add[\"id\"]”):
//
//   type Key struct {
//     BaseTok scan.Token       // IDENT token for base name (e.g. "item")
//     Indexers []Indexer       // zero or more
//   }
//   type IndexerKind int { IndexInt, IndexString, IndexIdent } // start with int+string only if preferred
//   type Indexer struct {
//     LBrackTok scan.Token
//     ValueTok  scan.Token     // NUMBER_LITERAL or QUOTE_LITERAL (or IDENT if needed)
//     RBrackTok scan.Token
//     Kind IndexerKind
//     IntValue int64           // if Kind==IndexInt
//   }
//
// Value representation (keep it light; validator mainly needs type + token spans):
//
//   type ValueKind int { ValNumber, ValString, ValIdent, ValObject }
//   type Value struct {
//     Kind ValueKind
//     Tok  scan.Token          // for number/string/ident
//     Obj  ObjectSpan          // for object value
//   }
//   type ObjectSpan struct { LBraceTok scan.Token; RBraceTok scan.Token }
//
// -------------------------
// 4) Grammar / walk functions (for .entities specifically)
// -------------------------
//
// Top-level walker:
//
//   func (c *cursor) walkEntities(h Handler) {
//     // Expect: Version <number>
//     // Then: zero or more component blocks until EOF
//   }
//
// Version:
//   - Expect IDENT "Version"
//   - Expect NUMBER_LITERAL
//   - Emit handler.OnVersion(...)
//
//
// Component block:
//
//   component {
//     <TypeIdent> <NameIdent> { <body> }
//   }
//
// Parse steps:
//   - expectIdent("component")
//   - expectSym('{')
//   - handler.OnComponentBegin(...)
//   - parse “component declaration” (typed block header):
//       expectKind(IDENTIFIER) -> typeTok
//       expectKind(IDENTIFIER) -> nameTok
//       expectSym('{') -> declLBrace
//       handler.OnComponentDecl(typeTok, nameTok, declLBrace)
//       walkObjectBody(h) until matching '}'
//   - expectSym('}') closing component outer
//   - handler.OnComponentEnd(...)
//
// Object body:
//
//   walkObjectBody assumes '{' has already been consumed and statements are parsed until '}'.
//
// Statement forms needed:
//
//   A) Assignment:
//       Key '=' Value ';'
//
//   B) Typed block header (common inside component outer):
//       Ident Ident '{' ObjectBody '}'   // no '=' and no ';'
//
//   C) (Optional) Ident '{' ObjectBody '}' // if it ever occurs
//
// Decision logic (peek-based, cheap):
//   - if next token is IDENT:
//       - parseKey() starting at that IDENT (consumes indexers too)
//       - if next is '=' => assignment
//       - else if next is IDENT (no indexers consumed after base) and next-next is '{' => typed block
//       - else if next is '{' => bare object block
//       - else => diag + recovery
//   - if next is '}' => end object
//   - else => unexpected token diag + sync
//
// parseKey():
//   - base = expectKind(IDENTIFIER)
//   - while next is '[':
//       - consume '['
//       - accept NUMBER_LITERAL or QUOTE_LITERAL (start with these two)
//       - consume ']'
//       - store Indexer (parse int value if number literal)
//   - return Key
//
// parseValue():
//   - if next is '{': parse object span by walking nested object body (and emitting OnObjectBegin/End)
//   - else if next is QUOTE_LITERAL: ValString
//   - else if next is NUMBER_LITERAL: ValNumber
//   - else if next is IDENTIFIER: ValIdent (true/false/etc)
//   - else: diag + recovery (sync to ';' or '}')
//
// Semicolon hookup:
//   - For assignment, require ';'.
//   - If missing, emit diag and sync to ';' or '}'.
//   - Pass the semicolon token to handler if present (useful span anchor).
//
// -------------------------
// 5) Diagnostics conventions (parse layer)
// -------------------------
//
// Create parse-specific diagnostic codes (in whatever package keeps codes):
//   - PARSE_UNEXPECTED_TOKEN
//   - PARSE_EXPECTED_SYMBOL
//   - PARSE_EXPECTED_IDENTIFIER
//   - PARSE_EXPECTED_SEMICOLON
//   - PARSE_UNTERMINATED_OBJECT
//
// Always point diagnostics at the smallest useful span:
//   - Unexpected token => that token span
//   - Missing symbol => zero-length span at “expected position” (use previous token end)
//   - Unterminated object => span from lbrace to EOF
//
// -------------------------
// 6) parse.go milestones (commit-sized)
// -------------------------
//
// Milestone 1: cursor + sym/lexeme + expect/match + basic diags.
// Milestone 2: walkEntities with Version + component skeleton (ignore body).
// Milestone 3: walkObjectBody + assignment parsing (Key '=' Value ';').
// Milestone 4: nested objects (ValueKind==ValObject) + OnObjectBegin/End.
// Milestone 5: typed block headers (Ident Ident '{' ... '}').
// Milestone 6: recovery (sync) tuned so parsing continues after an error.
//
//
// An extra note on package boundaries
// 	  Recommended flow in a main entrypoint later:
//  	 tokens, scanDiags := scan.Scan(src)
//  	 parse+validate diags := validate.ValidateEntities(src, tokens) // internally calls parse
//  	 allDiags := append(scanDiags, diags...)
//  	 out := report.Render(src, allDiags, RenderOptions{ContextLines: 1})
//
// 	  it will keep responsibilities clean:
//   	- scan: bytes -> tokens
//   	- parse: tokens -> structured walk (+ parse diags)
//   	- validate: semantic checks (+ validate diags)
//   	- report: nice output

// BONUS: Recovery rules used constantly
//
// 1) In assignment parsing, if '=' or value or ';' is missing:
//      - emit diag
//      - syncTo(';', '}')
//      - if ';' is reached, consume it (avoids looping)
//      - if '}' is reached, return to object loop (caller handles '}')
//
// 2) In object body loop, if an unexpected token appears:
//      - emit diag at that token
//      - advance one token
//      - (optional) if too many consecutive unexpected tokens, syncTo(';', '}')
//
// 3) Unterminated object:
//      - if EOF is reached while expecting '}', emit diag spanning from lbrace to EOF.
//
// Always ensure consumption of at least one token after emitting a diag; avoid infinite loops.
