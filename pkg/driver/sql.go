package driver

import (
	"fmt"
	"gorql"
	"net/url"
	"strconv"
	"strings"
)

type TranslatorOpFunc func(*gorql.RqlNode) (string, error)

type SqlTranslator struct {
	rootNode  *gorql.RqlRootNode
	sqlOpsDic map[string]TranslatorOpFunc
}

func (st *SqlTranslator) SetOpFunc(op string, f TranslatorOpFunc) {
	st.sqlOpsDic[strings.ToUpper(op)] = f
}

func (st *SqlTranslator) DeleteOpFunc(op string) {
	delete(st.sqlOpsDic, strings.ToUpper(op))
}

func (st *SqlTranslator) Where() (string, error) {
	if st.rootNode == nil {
		return "", nil
	}
	return st.where(st.rootNode.Node)
}

func (st *SqlTranslator) where(n *gorql.RqlNode) (string, error) {
	if n == nil {
		return ``, nil
	}
	f := st.sqlOpsDic[strings.ToUpper(n.Op)]
	if f == nil {
		return "", fmt.Errorf("no TranslatorOpFunc for op : '%s'", n.Op)
	}
	return f(n)
}

func (st *SqlTranslator) Limit() (sql string) {
	if st.rootNode == nil {
		return
	}
	limit := st.rootNode.Limit()
	if limit != "" && strings.ToUpper(limit) != "INFINITY" {
		sql = " LIMIT " + limit
	}
	return
}

func (st *SqlTranslator) Offset() (sql string) {
	if st.rootNode != nil && st.rootNode.Offset() != "" {
		sql = " OFFSET " + st.rootNode.Offset()
	}
	return
}

func (st *SqlTranslator) Sort() (sql string) {
	if st.rootNode == nil {
		return
	}
	sorts := st.rootNode.Sort()
	if len(sorts) > 0 {
		sql = " ORDER BY "
		sep := ""
		for _, sort := range sorts {
			sql = sql + sep + sort.By
			if sort.Desc {
				sql = sql + " DESC"
			}
			sep = ", "
		}
	}

	return
}

func (st *SqlTranslator) Sql() (sql string, err error) {
	var where string

	where, err = st.Where()
	if err != nil {
		return
	}

	if len(where) > 0 {
		sql = `WHERE ` + where
	}

	sort := st.Sort()
	if len(sort) > 0 {
		sql += sort
	}

	limit := st.Limit()
	if len(limit) > 0 {
		sql += limit
	}

	offset := st.Offset()
	if len(offset) > 0 {
		sql += offset
	}

	return sql, nil
}

func NewSqlTranslator(r *gorql.RqlRootNode) (st *SqlTranslator) {
	st = &SqlTranslator{r, map[string]TranslatorOpFunc{}}

	starToPercentFunc := AlterStringFunc(func(s string) (string, error) {
		return strings.Replace(Quote(s), `*`, `%`, -1), nil
	})

	st.SetOpFunc("AND", st.GetAndOrTranslatorOpFunc("AND"))
	st.SetOpFunc("OR", st.GetAndOrTranslatorOpFunc("OR"))

	st.SetOpFunc("NE", st.GetEqualityTranslatorOpFunc("!=", "IS NOT"))
	st.SetOpFunc("EQ", st.GetEqualityTranslatorOpFunc("=", "IS"))

	st.SetOpFunc("LIKE", st.GetFieldValueTranslatorFunc("LIKE", starToPercentFunc))
	st.SetOpFunc("MATCH", st.GetFieldValueTranslatorFunc("ILIKE", starToPercentFunc))
	st.SetOpFunc("GT", st.GetFieldValueTranslatorFunc(">", nil))
	st.SetOpFunc("LT", st.GetFieldValueTranslatorFunc("<", nil))
	st.SetOpFunc("GE", st.GetFieldValueTranslatorFunc(">=", nil))
	st.SetOpFunc("LE", st.GetFieldValueTranslatorFunc("<=", nil))
	st.SetOpFunc("NOT", st.GetOpFirstTranslatorFunc("NOT", nil))

	return
}

func (st *SqlTranslator) GetEqualityTranslatorOpFunc(op, specialOp string) TranslatorOpFunc {
	return func(n *gorql.RqlNode) (s string, err error) {
		value, err := url.QueryUnescape(n.Args[1].(string))
		if err != nil {
			return "", err
		}

		if value == `null` || value == `true` || value == `false` {
			field := n.Args[0].(string)
			if !IsValidField(field) {
				return ``, fmt.Errorf("invalid field name : %s", field)
			}

			return fmt.Sprintf("(%s %s %s)", field, specialOp, strings.ToUpper(value)), nil
		}

		return st.GetFieldValueTranslatorFunc(op, nil)(n)
	}
}

func (st *SqlTranslator) GetAndOrTranslatorOpFunc(op string) TranslatorOpFunc {
	return func(n *gorql.RqlNode) (s string, err error) {
		sep := ""

		for _, a := range n.Args {
			s = s + sep
			switch v := a.(type) {
			case string:
				if !IsValidField(v) {
					return "", fmt.Errorf("Invalid field name : %s", v)
				}
				s = s + v
			case *gorql.RqlNode:
				var tempS string
				tempS, err = st.where(v)
				if err != nil {
					return "", err
				}
				s = s + tempS
			}

			sep = " " + op + " "
		}

		return "(" + s + ")", nil
	}
}

type AlterStringFunc func(string) (string, error)

func (st *SqlTranslator) GetFieldValueTranslatorFunc(op string, valueAlterFunc AlterStringFunc) TranslatorOpFunc {
	return func(n *gorql.RqlNode) (s string, err error) {
		sep := ""

		for i, a := range n.Args {
			s += sep
			switch v := a.(type) {
			case string:
				var tempS string
				if i == 0 {
					if IsValidField(v) {
						tempS = v
					} else {
						return "", fmt.Errorf("first argument must be a valid field name (arg: %s)", v)
					}
				} else {
					_, err := strconv.ParseInt(v, 10, 64)
					if err == nil {
						tempS = v
					} else if valueAlterFunc != nil {
						tempS, err = valueAlterFunc(v)
						if err != nil {
							return "", err
						}
					} else {
						tempS = Quote(v)
					}
				}

				s += tempS
			case *gorql.RqlNode:
				var tempS string
				tempS, err = st.where(v)
				if err != nil {
					return "", err
				}
				s = s + tempS
			}

			sep = " " + op + " "
		}

		return "(" + s + ")", nil
	}
}

func (st *SqlTranslator) GetOpFirstTranslatorFunc(op string, valueAlterFunc AlterStringFunc) TranslatorOpFunc {
	return func(n *gorql.RqlNode) (s string, err error) {
		sep := ""

		for _, a := range n.Args {
			s += sep
			switch v := a.(type) {
			case string:
				var tempS string
				_, err := strconv.ParseInt(v, 10, 64)
				if err == nil || IsValidField(v) {
					tempS = v
				} else if valueAlterFunc != nil {
					tempS, err = valueAlterFunc(v)
					if err != nil {
						return "", err
					}
				} else {
					tempS = Quote(v)
				}

				s += tempS
			case *gorql.RqlNode:
				var tempS string
				tempS, err = st.where(v)
				if err != nil {
					return "", err
				}
				s = s + tempS
			}

			sep = ", "
		}

		return op + "(" + s + ")", nil
	}
}

func IsValidField(s string) bool {
	for _, ch := range s {
		if !gorql.IsLetter(ch) && !gorql.IsDigit(ch) && ch != '_' && ch != '-' && ch != '.' {
			return false
		}
	}

	return true
}

func Quote(s string) string {
	return `'` + strings.Replace(s, `'`, `''`, -1) + `'`
}
