package sql

import (
	"fmt"
	"github.com/douglaslim/gorql"
	"github.com/douglaslim/gorql/pkg/driver"
	"net/url"
	"strconv"
	"strings"
)

type Translator struct {
	rootNode  *gorql.RqlRootNode
	sqlOpsDic map[string]driver.TranslatorOpFunc
}

func (st *Translator) SetOpFunc(op string, f driver.TranslatorOpFunc) {
	st.sqlOpsDic[strings.ToUpper(op)] = f
}

func (st *Translator) DeleteOpFunc(op string) {
	delete(st.sqlOpsDic, strings.ToUpper(op))
}

func (st *Translator) Where() (string, error) {
	if st.rootNode == nil {
		return "", nil
	}
	return st.where(st.rootNode.Node)
}

func (st *Translator) where(n *gorql.RqlNode) (string, error) {
	if n == nil {
		return ``, nil
	}
	f := st.sqlOpsDic[strings.ToUpper(n.Op)]
	if f == nil {
		return "", fmt.Errorf("no TranslatorOpFunc for op : '%s'", n.Op)
	}
	return f(n)
}

func (st *Translator) Limit() (sql string) {
	if st.rootNode == nil {
		return
	}
	limit := st.rootNode.Limit()
	if limit != "" && strings.ToUpper(limit) != "INFINITY" {
		sql = " LIMIT " + limit
	}
	return
}

func (st *Translator) Offset() (sql string) {
	if st.rootNode != nil && st.rootNode.Offset() != "" {
		sql = " OFFSET " + st.rootNode.Offset()
	}
	return
}

func (st *Translator) Sort() (sql string) {
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

func (st *Translator) Selects() (fields string) {
	if st.rootNode == nil {
		return
	}
	return strings.Join(st.rootNode.Selects(), ",")
}

func (st *Translator) Sql() (sql string, err error) {
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

func NewSqlTranslator(r *gorql.RqlRootNode) (st *Translator) {
	st = &Translator{r, map[string]driver.TranslatorOpFunc{}}

	starToPercentFunc := AlterStringFunc(func(s string) (string, error) {
		return strings.Replace(quote(s), `*`, `%`, -1), nil
	})

	st.SetOpFunc(driver.AndOp, st.GetAndOrTranslatorOpFunc(driver.AndOp))
	st.SetOpFunc(driver.OrOp, st.GetAndOrTranslatorOpFunc(driver.OrOp))

	st.SetOpFunc(driver.NeOp, st.GetEqualityTranslatorOpFunc("!=", "IS NOT"))
	st.SetOpFunc(driver.EqOp, st.GetEqualityTranslatorOpFunc("=", "IS"))

	st.SetOpFunc(driver.LikeOp, st.GetFieldValueTranslatorFunc(driver.LikeOp, starToPercentFunc))
	st.SetOpFunc(driver.MatchOp, st.GetFieldValueTranslatorFunc("ILIKE", starToPercentFunc))
	st.SetOpFunc(driver.GtOp, st.GetFieldValueTranslatorFunc(">", nil))
	st.SetOpFunc(driver.LtOp, st.GetFieldValueTranslatorFunc("<", nil))
	st.SetOpFunc(driver.GeOp, st.GetFieldValueTranslatorFunc(">=", nil))
	st.SetOpFunc(driver.LeOp, st.GetFieldValueTranslatorFunc("<=", nil))
	st.SetOpFunc(driver.NotOp, st.GetOpFirstTranslatorFunc(driver.NotOp, nil))

	return
}

func (st *Translator) GetEqualityTranslatorOpFunc(op, specialOp string) driver.TranslatorOpFunc {
	return func(n *gorql.RqlNode) (s string, err error) {
		value, err := url.QueryUnescape(n.Args[1].(string))
		if err != nil {
			return "", err
		}

		if value == `null` || value == `true` || value == `false` {
			field := n.Args[0].(string)
			if !gorql.IsValidField(field) {
				return ``, fmt.Errorf("invalid field name : %s", field)
			}

			return fmt.Sprintf("(%s %s %s)", field, specialOp, strings.ToUpper(value)), nil
		}

		return st.GetFieldValueTranslatorFunc(op, nil)(n)
	}
}

func (st *Translator) GetAndOrTranslatorOpFunc(op string) driver.TranslatorOpFunc {
	return func(n *gorql.RqlNode) (s string, err error) {
		sep := ""

		for _, a := range n.Args {
			s = s + sep
			switch v := a.(type) {
			case string:
				if !gorql.IsValidField(v) {
					return "", fmt.Errorf("invalid field name : %s", v)
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

func (st *Translator) GetFieldValueTranslatorFunc(op string, valueAlterFunc AlterStringFunc) driver.TranslatorOpFunc {
	return func(n *gorql.RqlNode) (s string, err error) {
		sep := ""

		for i, a := range n.Args {
			s += sep
			switch v := a.(type) {
			case string:
				var tempS string
				if i == 0 {
					if gorql.IsValidField(v) {
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
						tempS = quote(v)
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

func (st *Translator) GetOpFirstTranslatorFunc(op string, valueAlterFunc AlterStringFunc) driver.TranslatorOpFunc {
	return func(n *gorql.RqlNode) (s string, err error) {
		sep := ""

		for _, a := range n.Args {
			s += sep
			switch v := a.(type) {
			case string:
				var tempS string
				_, err := strconv.ParseInt(v, 10, 64)
				if err == nil || gorql.IsValidField(v) {
					tempS = v
				} else if valueAlterFunc != nil {
					tempS, err = valueAlterFunc(v)
					if err != nil {
						return "", err
					}
				} else {
					tempS = quote(v)
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

func quote(s string) string {
	return `'` + strings.Replace(s, `'`, `''`, -1) + `'`
}
