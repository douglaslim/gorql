package driver

import (
	"github.com/douglaslim/gorql"
)

const (
	AndOp   = "AND"
	OrOp    = "OR"
	NeOp    = "NE"
	EqOp    = "EQ"
	LikeOp  = "LIKE"
	MatchOp = "MATCH"
	GtOp    = "GT"
	LtOp    = "LT"
	GeOp    = "GE"
	LeOp    = "LE"
	NotOp   = "NOT"
	InOp    = "IN"
)

type TranslatorOpFunc func(*gorql.RqlNode) (string, error)
