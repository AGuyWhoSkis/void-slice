High level architecture for Void Slice

Scanner / Tokenizer
    knows about bytes, comments, strings, numbers, and symbols
    outputs Tokens (as type+Span) + diagnostic errors
    does not attempt to enforce balanced braces beyond emitting brace tokens

Parser / Structural
    knows the grammar shape: assignments, objects, bracketed indices, statement terminators
    outputs structure (events) + diagnostic errors
    does not know about “num counts items” beyond parsing it as a normal assignment

Validators
    operate on parsed structure
    can be layered:
        generic “container/num/item” validator
        file-type validators (decl, entities)
        schema/engine validators (pluggable)

Result type for the whole pipeline:
    []Token (optional to return at top-level; useful for tooling/debug)
    []Diagnostic from all stages
    error only for “could not run pipeline” (IO, internal invariant broken, etc.)