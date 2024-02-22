package gorql

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/url"
)

const (
	// Special tokens
	Illegal Token = iota
	Eof

	// Literals
	Ident // fields, function names

	// Reserved characters
	Space              //
	Ampersand          // &
	OpeningParenthesis // (
	ClosingParenthesis // )
	Comma              // ,
	EqualSign          // =
	Slash              // /
	SemiColon          // ;
	QuestionMark       // ?
	AtSymbol           // @
	Pipe               // |

	// Keywords
	And
	Or
	Equal
	Greater
	GreaterOrEqual
	Lower
	LowerOrEqual
	NotEqual
)

var (
	ReservedRunes = []rune{' ', '&', '(', ')', ',', '=', '/', ';', '?', '@', '|'}
	eof           = rune(0)
)

type Token int

type TokenString struct {
	t Token
	s string
}

func NewTokenString(t Token, s string) TokenString {
	unescapedString := ""
	if len(s) > 0 && string(s[0]) != "+" {
		unescapedString, _ = url.QueryUnescape(s)
	} else {
		//Golang`s "unescape" method replaces "+" with " ", however "+" literal is not possible in urlencoded string
		//this is a case of sorting argument specification: eg: sort(+name), thus in this case string left unmodified
		unescapedString = s
	}
	return TokenString{t: t, s: unescapedString}
}

type Scanner struct {
	r *bufio.Reader
}

func NewScanner() *Scanner {
	return &Scanner{}
}

// Scan returns the next token and literal value.
func (s *Scanner) Scan(r io.Reader) (out []TokenString, err error) {
	s.r = bufio.NewReader(r)

	for {
		tok, lit := s.ScanToken()
		if tok == Eof {
			break
		} else if tok == Illegal {
			return out, fmt.Errorf("illegal Token : %s", lit)
		} else {
			out = append(out, NewTokenString(tok, lit))
		}
	}

	return
}

func (s *Scanner) ScanToken() (tok Token, lit string) {
	ch := s.read()

	if isReservedRune(ch) {
		s.unread()
		return s.scanReservedRune()
	} else if isIdent(ch) {
		s.unread()
		return s.scanIdent()
	}

	if ch == eof {
		return Eof, ""
	}

	return Illegal, string(ch)
}

func (s *Scanner) read() rune {
	ch, _, err := s.r.ReadRune()
	if err != nil {
		return eof
	}
	return ch
}

// unread places the previously read rune back on the reader.
func (s *Scanner) unread() { _ = s.r.UnreadRune() }

func isReservedRune(ch rune) bool {
	for _, rr := range ReservedRunes {
		if ch == rr {
			return true
		}
	}
	return false
}

func (s *Scanner) scanReservedRune() (tok Token, lit string) {
	// Create a buffer and read the current character into it.
	var buf bytes.Buffer

	buf.WriteRune(s.read())
	lit = buf.String()

	// Read every subsequent whitespace character into the buffer.
	// Non-whitespace characters and EOF will cause the loop to exit.

	for _, rr := range ReservedRunes {
		if string(rr) == lit {
			switch rr {
			case '&':
				return Ampersand, lit
			case '(':
				return OpeningParenthesis, lit
			case ')':
				return ClosingParenthesis, lit
			case ',':
				return Comma, lit
			case '=':
				return EqualSign, lit
			case '/':
				return Slash, lit
			case ';':
				return SemiColon, lit
			case '?':
				return QuestionMark, lit
			case '@':
				return AtSymbol, lit
			case '|':
				return Pipe, lit
			case eof:
				return Eof, lit
			default:
				return Illegal, lit
			}
		}
	}
	return Illegal, lit
}

// isIdent returns true if the rune is an identifier
func isIdent(ch rune) bool {
	return IsLetter(ch) || IsDigit(ch) || isSpecialChar(ch)
}

// isSpecialChar returns true if the rune is a special character.
func isSpecialChar(ch rune) bool {
	return ch == '*' || ch == '_' || ch == '%' ||
		ch == '+' || ch == '-' || ch == '.'
}

// IsLetter returns true if the rune is a letter.
func IsLetter(ch rune) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z')
}

// IsDigit returns true if the rune is a digit.
func IsDigit(ch rune) bool { return ch >= '0' && ch <= '9' }

func (s *Scanner) scanIdent() (tok Token, lit string) {
	// Create a buffer and read the current character into it.
	var buf bytes.Buffer
	buf.WriteRune(s.read())

	// Read every subsequent ident character into the buffer.
	// Non-ident characters and EOF will cause the loop to exit.
	for {
		if ch := s.read(); ch == eof {
			break
		} else if !isIdent(ch) {
			s.unread()
			break
		} else {
			_, _ = buf.WriteRune(ch)
		}
	}

	return Ident, buf.String()
}
