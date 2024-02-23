package gorql

import (
	"errors"
	"fmt"
	"io"
	"strings"
)

var (
	reservedOps = []string{OffsetOp, LimitOp, SelectOp, SortOp}
)

const (
	OffsetOp = "offset"
	LimitOp  = "limit"
	SelectOp = "select"
	SortOp   = "sort"
)

type RqlNode struct {
	Op   string
	Args []interface{}
}

type Sort struct {
	By   string
	Desc bool
}

type RqlRootNode struct {
	Node   *RqlNode
	limit  string
	offset string
	fields []string
	sorts  []Sort
}

func (r *RqlRootNode) Limit() string {
	return r.limit
}

func (r *RqlRootNode) Offset() string {
	return r.offset
}

func (r *RqlRootNode) Sort() []Sort {
	return r.sorts
}

var ErrValueError = fmt.Errorf("bloc is a value")

type TokenBloc []TokenString

// String print the TokenBloc value for test purpose only
func (tb TokenBloc) String() (s string) {
	for _, t := range tb {
		s = s + fmt.Sprintf("'%s' ", t.s)
	}
	return
}

func (r *RqlRootNode) ParseSpecialOps() {
	if parseLimit(r.Node, r) || parseSort(r.Node, r) || parseOffset(r.Node, r) || parseFields(r.Node, r) {
		r.Node = nil
	} else if r.Node != nil {
		if strings.ToUpper(r.Node.Op) == "AND" {
			tmpNodeArgs := r.Node.Args[:0]
			for _, c := range r.Node.Args {
				switch n := c.(type) {
				case *RqlNode:
					isSpecialOps := parseLimit(n, r) || parseSort(n, r) || parseOffset(n, r) || parseFields(n, r)
					if !isSpecialOps {
						tmpNodeArgs = append(tmpNodeArgs, n)
					}
				}
			}
			r.Node.Args = tmpNodeArgs
			if len(r.Node.Args) == 0 {
				r.Node = nil
			}
		}
	}
}

func parseLimit(n *RqlNode, root *RqlRootNode) (isLimitOp bool) {
	if n == nil {
		return false
	}
	if n.Op == LimitOp {
		root.limit = n.Args[1].(string)
		isLimitOp = true
	}
	return
}

func parseOffset(n *RqlNode, root *RqlRootNode) (isOffsetOp bool) {
	if n == nil {
		return false
	}
	if n.Op == OffsetOp {
		root.offset = n.Args[1].(string)
		isOffsetOp = true
	}
	return
}

func parseSort(n *RqlNode, root *RqlRootNode) (isSortOp bool) {
	if n == nil {
		return false
	}
	if n.Op == SortOp {
		for _, s := range n.Args {
			property := s.(string)
			desc := false

			if property[0] == '+' {
				property = property[1:]
			} else if property[0] == '-' {
				desc = true
				property = property[1:]
			}
			root.sorts = append(root.sorts, Sort{By: property, Desc: desc})
		}

		isSortOp = true
	}
	return
}

func parseFields(n *RqlNode, root *RqlRootNode) (isFieldsOp bool) {
	if n == nil {
		return false
	}
	if n.Op == SelectOp {
		for _, s := range n.Args {
			property := s.(string)
			root.fields = append(root.fields, property)
		}
		isFieldsOp = true
	}
	return
}

type Parser struct {
	s *Scanner
}

func NewParser() *Parser {
	return &Parser{s: NewScanner()}
}

// Parse constructs an AST for code transformation
func (p *Parser) Parse(r io.Reader) (root *RqlRootNode, err error) {
	var tokenStrings []TokenString
	if tokenStrings, err = p.s.Scan(r); err != nil {
		return nil, err
	}
	root = &RqlRootNode{}
	root.Node, err = parse(tokenStrings)
	if err != nil {
		return nil, err
	}
	root.ParseSpecialOps()
	return
}

// parse recursively return the children node from the tokens
func parse(ts []TokenString) (node *RqlNode, err error) {
	var childNode *RqlNode

	var childTs [][]TokenString
	node = &RqlNode{}
	if len(ts) == 0 {
		return nil, nil
	}
	if isParenthesisBloc(ts) && findClosingIndex(ts[1:]) == len(ts)-2 {
		ts = ts[1 : len(ts)-1]
	}

	node.Op, childTs = splitByBasisOp(ts)
	if node.Op == "" || len(childTs) == 1 {
		return getBlocNode(ts)
	}

	for _, c := range childTs {
		childNode, err = parse(c)
		if err != nil {
			if errors.Is(err, ErrValueError) {
				node.Args = append(node.Args, c[0].s)
			} else {
				return nil, err
			}
		} else {
			node.Args = append(node.Args, childNode)
		}
	}

	return
}

// isParenthesisBloc returns true if the token strings is a parenthesis block
func isParenthesisBloc(tb []TokenString) bool {
	return tb[0].t == OpeningParenthesis
}

// findClosingIndex returns the index of the closing parenthesis from the token strings
func findClosingIndex(tb []TokenString) int {
	i := findTokenIndex(tb, ClosingParenthesis)
	return i
}

// findTokenIndex returns the token index for the search token
func findTokenIndex(tb []TokenString, token Token) int {
	depth := 0
	for i, ts := range tb {
		if ts.t == OpeningParenthesis {
			depth++
		} else if ts.t == ClosingParenthesis {
			if depth == 0 && token == ClosingParenthesis {
				return i
			}
			depth--
		} else if token == ts.t && depth == 0 {
			return i
		}
	}
	return -1
}

func splitByBasisOp(tb []TokenString) (op string, tbs [][]TokenString) {
	matchingToken := Illegal

	depth := 0
	lastIndex := 0

	basisTokenGroups := [][]Token{
		{Ampersand, Comma},
		{Pipe, SemiColon},
	}
	for _, bt := range basisTokenGroups {
		btExtended := append(bt, Illegal)
		for i, ts := range tb {
			if ts.t == OpeningParenthesis {
				depth++
			} else if ts.t == ClosingParenthesis && depth > 0 {
				depth--
			} else if depth == 0 {
				if isTokenInSlice(bt, ts.t) && isTokenInSlice(btExtended, matchingToken) {
					matchingToken = ts.t
					tbs = append(tbs, tb[lastIndex:i])
					lastIndex = i + 1
				}
			}
		}
		if lastIndex != 0 {
			break
		}
	}

	tbs = append(tbs, tb[lastIndex:])
	op = getTokenOp(matchingToken)
	return
}

func isTokenInSlice(tokens []Token, tok Token) bool {
	for _, t := range tokens {
		if t == tok {
			return true
		}
	}
	return false
}

func getTokenOp(t Token) string {
	switch t {
	case Ampersand, Comma:
		return "AND"
	case Pipe, SemiColon:
		return "OR"
	}
	return ""
}

func getBlocNode(tb []TokenString) (*RqlNode, error) {
	n := &RqlNode{}

	if isValue(tb) {
		return nil, ErrValueError
	} else if isFuncStyleBloc(tb) {
		var err error
		n.Op = tb[0].s
		tb = tb[2:]
		ci := findClosingIndex(tb)
		if len(tb) > ci+1 && tb[ci+1].t != ClosingParenthesis && tb[ci+1].t != Comma {
			return nil, fmt.Errorf("unrecognized func style bloc (missing comma?)")
		}
		n.Args, err = parseFuncArgs(tb[:ci])
		if err != nil {
			return nil, err
		}
	} else if isReservedBloc(tb) {
		n.Op = tb[0].s
		n.Args = []interface{}{tb[0].s, tb[2].s}
	} else if isSimpleEqualBloc(tb) {
		n.Op = "eq"
		n.Args = []interface{}{tb[0].s, tb[2].s}
	} else if isDoubleEqualBloc(tb) {
		n.Op = tb[2].s
		n.Args = []interface{}{tb[0].s}
		tbLen := len(tb)
		if tbLen == 4 {
			n.Args = append(n.Args, ``)
		} else if isParenthesisBloc(tb[4:]) && findClosingIndex(tb[5:]) == tbLen-6 {
			args, err := parseFuncArgs(tb[5 : tbLen-1])
			if err != nil {
				return nil, err
			}
			n.Args = append(n.Args, args...)
		} else {
			arg := ``
			for _, a := range tb[4:] {
				arg = arg + a.s
			}
			n.Args = append(n.Args, arg)
		}

	} else {
		return nil, fmt.Errorf("Unrecognized bloc : " + TokenBloc(tb).String())
	}

	return n, nil
}

func isValue(tb []TokenString) bool {
	return len(tb) == 1 && tb[0].t == Ident
}

func isFuncStyleBloc(tb []TokenString) bool {
	return (tb[0].t == Ident) && (tb[1].t == OpeningParenthesis)
}

func parseFuncArgs(tb []TokenString) (args []interface{}, err error) {
	var argTokens [][]TokenString

	indexes := findAllTokenIndexes(tb, Comma)

	if len(indexes) == 0 {
		argTokens = append(argTokens, tb)
	} else {
		lastIndex := 0
		for _, i := range indexes {
			argTokens = append(argTokens, tb[lastIndex:i])
			lastIndex = i + 1
		}
		argTokens = append(argTokens, tb[lastIndex:])
	}

	for _, ts := range argTokens {
		n, err := parse(ts)
		if err != nil {
			if errors.Is(err, ErrValueError) {
				args = append(args, ts[0].s)
			} else {
				return args, err
			}
		} else {
			args = append(args, n)
		}
	}

	return
}

func findAllTokenIndexes(tb []TokenString, token Token) (indexes []int) {
	depth := 0
	for i, ts := range tb {
		if ts.t == OpeningParenthesis {
			depth++
		} else if ts.t == ClosingParenthesis {
			if depth == 0 && token == ClosingParenthesis {
				indexes = append(indexes, i)
			}
			depth--
		} else if token == ts.t && depth == 0 {
			indexes = append(indexes, i)
		}
	}
	return
}

func isSimpleEqualBloc(tb []TokenString) bool {
	isSimple := tb[0].t == Ident && tb[1].t == EqualSign
	if len(tb) > 3 {
		isSimple = isSimple && tb[3].t != EqualSign
	}

	return isSimple
}

func isReservedBloc(tb []TokenString) bool {
	matchReserved := false
	for _, r := range reservedOps {
		if tb[0].s == r {
			matchReserved = true
		}
	}
	return tb[0].t == Ident && tb[1].t == EqualSign && matchReserved
}

func isDoubleEqualBloc(tb []TokenString) bool {
	return tb[0].t == Ident && tb[1].t == EqualSign && tb[2].t == Ident && tb[3].t == EqualSign
}
