package lua

import "fmt"

// Position identifies a location within a Lua source file. Line and Column
// are 1-based; Offset is the 0-based byte index into the source buffer.
type Position struct {
	Filename string
	Line     int
	Column   int
	Offset   int
}

// String renders the position in "file:line:col" form, or "line:col" if no
// filename is set.
func (p Position) String() string {
	if p.Filename == "" {
		return fmt.Sprintf("%d:%d", p.Line, p.Column)
	}
	return fmt.Sprintf("%s:%d:%d", p.Filename, p.Line, p.Column)
}

// TokenKind classifies a lexical token.
type TokenKind int

// Token kinds. The values are stable within a minor version but callers
// should compare against the named constants rather than the integers.
const (
	TokenEOF TokenKind = iota
	TokenError

	// Literals and identifiers.
	TokenIdent
	TokenNumber
	TokenString

	// Keywords (Lua 5.1 + `goto` from LuaJIT).
	TokenAnd
	TokenBreak
	TokenDo
	TokenElse
	TokenElseif
	TokenEnd
	TokenFalse
	TokenFor
	TokenFunction
	TokenGoto
	TokenIf
	TokenIn
	TokenLocal
	TokenNil
	TokenNot
	TokenOr
	TokenRepeat
	TokenReturn
	TokenThen
	TokenTrue
	TokenUntil
	TokenWhile

	// Operators and punctuation.
	TokenPlus        // +
	TokenMinus       // -
	TokenStar        // *
	TokenSlash       // /
	TokenDoubleSlash // //  (LuaJIT integer division)
	TokenPercent     // %
	TokenCaret       // ^
	TokenHash        // #
	TokenEq          // ==
	TokenNeq         // ~=
	TokenLeq         // <=
	TokenGeq         // >=
	TokenLt          // <
	TokenGt          // >
	TokenAssign      // =
	TokenLParen      // (
	TokenRParen      // )
	TokenLBrace      // {
	TokenRBrace      // }
	TokenLBracket    // [
	TokenRBracket    // ]
	TokenSemicolon   // ;
	TokenColon       // :
	TokenDoubleColon // ::
	TokenComma       // ,
	TokenDot         // .
	TokenConcat      // ..
	TokenEllipsis    // ...
)

// Token is a single lexical unit produced by the Lexer.
type Token struct {
	Kind  TokenKind
	Value string
	Pos   Position
}

var tokenKindNames = map[TokenKind]string{
	TokenEOF: "EOF", TokenError: "ERROR",
	TokenIdent: "IDENT", TokenNumber: "NUMBER", TokenString: "STRING",
	TokenAnd: "and", TokenBreak: "break", TokenDo: "do",
	TokenElse: "else", TokenElseif: "elseif", TokenEnd: "end",
	TokenFalse: "false", TokenFor: "for", TokenFunction: "function",
	TokenGoto: "goto", TokenIf: "if", TokenIn: "in",
	TokenLocal: "local", TokenNil: "nil", TokenNot: "not",
	TokenOr: "or", TokenRepeat: "repeat", TokenReturn: "return",
	TokenThen: "then", TokenTrue: "true", TokenUntil: "until",
	TokenWhile: "while",
	TokenPlus:  "+", TokenMinus: "-", TokenStar: "*",
	TokenSlash: "/", TokenDoubleSlash: "//", TokenPercent: "%",
	TokenCaret: "^", TokenHash: "#",
	TokenEq: "==", TokenNeq: "~=", TokenLeq: "<=", TokenGeq: ">=",
	TokenLt: "<", TokenGt: ">", TokenAssign: "=",
	TokenLParen: "(", TokenRParen: ")",
	TokenLBrace: "{", TokenRBrace: "}",
	TokenLBracket: "[", TokenRBracket: "]",
	TokenSemicolon: ";", TokenColon: ":", TokenDoubleColon: "::",
	TokenComma: ",", TokenDot: ".", TokenConcat: "..", TokenEllipsis: "...",
}

// String returns the canonical spelling of a TokenKind.
func (k TokenKind) String() string {
	if name, ok := tokenKindNames[k]; ok {
		return name
	}
	return fmt.Sprintf("TokenKind(%d)", int(k))
}

// keywords maps a source lexeme to its keyword TokenKind. Populated from
// tokenKindNames at init to keep the two in sync.
var keywords = map[string]TokenKind{
	"and": TokenAnd, "break": TokenBreak, "do": TokenDo,
	"else": TokenElse, "elseif": TokenElseif, "end": TokenEnd,
	"false": TokenFalse, "for": TokenFor, "function": TokenFunction,
	"goto": TokenGoto, "if": TokenIf, "in": TokenIn,
	"local": TokenLocal, "nil": TokenNil, "not": TokenNot,
	"or": TokenOr, "repeat": TokenRepeat, "return": TokenReturn,
	"then": TokenThen, "true": TokenTrue, "until": TokenUntil,
	"while": TokenWhile,
}
