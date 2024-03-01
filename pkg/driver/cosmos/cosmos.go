package cosmos

import (
	"fmt"
	"github.com/douglaslim/gorql"
	"github.com/douglaslim/gorql/pkg/driver"
	"net/url"
	"strconv"
	"strings"
)

type Translator struct {
	rootNode *gorql.RqlRootNode
	opsDic   map[string]driver.TranslatorOpFunc
	args     []interface{}
}

type Param struct {
	Name  string      `json:"name"` // should contain a @ character
	Value interface{} `json:"value"`
}

func (ct *Translator) SetOpFunc(op string, f driver.TranslatorOpFunc) {
	ct.opsDic[strings.ToUpper(op)] = f
}

func (ct *Translator) DeleteOpFunc(op string) {
	delete(ct.opsDic, strings.ToUpper(op))
}

func (ct *Translator) Where() (string, error) {
	if ct.rootNode == nil {
		return "", nil
	}
	ct.args = make([]interface{}, 0)
	return ct.where(ct.rootNode.Node)
}

func (ct *Translator) where(n *gorql.RqlNode) (string, error) {
	if n == nil {
		return ``, nil
	}
	f := ct.opsDic[strings.ToUpper(n.Op)]
	if f == nil {
		return "", fmt.Errorf("no TranslatorOpFunc for op : '%s'", n.Op)
	}
	return f(n)
}

func (ct *Translator) Limit() (limit string) {
	if ct.rootNode == nil {
		return
	}
	return ct.rootNode.Limit()
}

func (ct *Translator) Offset() (sql string) {
	if ct.rootNode != nil {
		return
	}
	return ct.rootNode.Offset()
}

func (ct *Translator) Sort() (sql string) {
	if ct.rootNode == nil {
		return
	}
	sorts := ct.rootNode.Sort()
	if len(sorts) > 0 {
		sql = " ORDER BY "
		sep := ""
		for _, sort := range sorts {
			sql = sql + sep + fmt.Sprintf("c.%s", sort.By)
			if sort.Desc {
				sql = sql + " DESC"
			}
			sep = ", "
		}
	}

	return
}

func (ct *Translator) Sql() (sql string, err error) {
	var where string

	where, err = ct.Where()
	if err != nil {
		return
	}

	if len(where) > 0 {
		sql = `WHERE ` + where
	}

	sort := ct.Sort()
	if len(sort) > 0 {
		sql += sort
	}
	return sql, nil
}

var convert = AlterValueFunc(func(value interface{}) (interface{}, error) {
	return value, nil
})

var starToPercentFunc = AlterValueFunc(func(value interface{}) (interface{}, error) {
	return strings.Replace(value.(string), `*`, `%`, -1), nil
})

func NewCosmosTranslator(r *gorql.RqlRootNode) (st *Translator) {
	st = &Translator{rootNode: r, opsDic: map[string]driver.TranslatorOpFunc{}}

	st.SetOpFunc(driver.AndOp, st.GetAndOrTranslatorOpFunc(driver.AndOp))
	st.SetOpFunc(driver.OrOp, st.GetAndOrTranslatorOpFunc(driver.OrOp))

	st.SetOpFunc(driver.NeOp, st.GetEqualityTranslatorOpFunc("!=", "IS NOT"))
	st.SetOpFunc(driver.EqOp, st.GetEqualityTranslatorOpFunc("=", "IS"))

	st.SetOpFunc(driver.LikeOp, st.GetFieldValueTranslatorFunc(driver.LikeOp, starToPercentFunc))
	st.SetOpFunc(driver.MatchOp, st.GetFieldValueTranslatorFunc("ILIKE", starToPercentFunc))
	st.SetOpFunc(driver.GtOp, st.GetFieldValueTranslatorFunc(">", convert))
	st.SetOpFunc(driver.LtOp, st.GetFieldValueTranslatorFunc("<", convert))
	st.SetOpFunc(driver.GeOp, st.GetFieldValueTranslatorFunc(">=", convert))
	st.SetOpFunc(driver.LeOp, st.GetFieldValueTranslatorFunc("<=", convert))
	st.SetOpFunc(driver.NotOp, st.GetOpFirstTranslatorFunc(driver.NotOp, convert))

	return
}

func (ct *Translator) GetEqualityTranslatorOpFunc(op, specialOp string) driver.TranslatorOpFunc {
	return func(n *gorql.RqlNode) (s string, err error) {
		value, ok := n.Args[1].(string)
		if ok {
			escVal, err := url.QueryUnescape(value)
			if err != nil {
				return "", err
			}
			value = escVal
			if value == `null` || value == `true` || value == `false` {
				field := n.Args[0].(string)
				if !gorql.IsValidField(field) {
					return ``, fmt.Errorf("invalid field name : %s", field)
				}

				return fmt.Sprintf("(%s %s %s)", field, specialOp, strings.ToUpper(value)), nil
			}
		}
		return ct.GetFieldValueTranslatorFunc(op, nil)(n)
	}
}

func (ct *Translator) GetAndOrTranslatorOpFunc(op string) driver.TranslatorOpFunc {
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
				tempS, err = ct.where(v)
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

type AlterValueFunc func(interface{}) (interface{}, error)

func (ct *Translator) GetFieldValueTranslatorFunc(op string, valueAlterFunc AlterValueFunc) driver.TranslatorOpFunc {
	return func(n *gorql.RqlNode) (s string, err error) {
		sep := ""

		for i, a := range n.Args {
			s += sep
			switch v := a.(type) {
			case *gorql.RqlNode:
				var tempS string
				tempS, err = ct.where(v)
				if err != nil {
					return "", err
				}
				s = s + tempS
			default:
				var tempS string
				if i == 0 {
					field := v.(string)
					if gorql.IsValidField(field) {
						tempS = fmt.Sprintf("c.%s", field)
					} else {
						return "", fmt.Errorf("first argument must be a valid field name (arg: %s)", v)
					}
				} else {
					placholder := fmt.Sprintf("@%s", strconv.Itoa(len(ct.args)+1))
					var value = v
					if valueAlterFunc != nil {
						value, err = valueAlterFunc(v)
						if err != nil {
							return "", err
						}
					}
					if value != "" {
						ct.args = append(ct.args, Param{
							Name:  placholder,
							Value: value,
						})
						tempS = placholder
					} else {
						tempS = quote(value.(string))
					}

				}
				s += tempS
			}

			sep = " " + op + " "
		}

		return "(" + s + ")", nil
	}
}

func (ct *Translator) GetOpFirstTranslatorFunc(op string, valueAlterFunc AlterValueFunc) driver.TranslatorOpFunc {
	return func(n *gorql.RqlNode) (s string, err error) {
		sep := ""

		for _, a := range n.Args {
			s += sep
			switch v := a.(type) {
			case string:
				placholder := fmt.Sprintf("@%s", strconv.Itoa(len(ct.args)+1))
				var value interface{} = v
				if valueAlterFunc != nil {
					value, err = valueAlterFunc(v)
					if err != nil {
						return "", err
					}
				}
				ct.args = append(ct.args, Param{
					Name:  placholder,
					Value: value,
				})
				s += placholder
			case *gorql.RqlNode:
				var tempS string
				tempS, err = ct.where(v)
				if err != nil {
					return "", err
				}
				s = s + tempS
			}

			sep = " AND "
		}

		return op + "(" + s + ")", nil
	}
}

// Args returns slice of arguments for WHERE statement
func (ct *Translator) Args() []interface{} {
	return ct.args
}

func quote(s string) string {
	return `'` + strings.Replace(s, `'`, `''`, -1) + `'`
}
