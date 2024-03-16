package mongo

import (
	"fmt"
	"github.com/douglaslim/gorql"
	"github.com/douglaslim/gorql/pkg/driver"
	"strconv"
	"strings"
	"time"
)

type Translator struct {
	rootNode *gorql.RqlRootNode
	OpsDic   map[string]driver.TranslatorOpFunc
}

type AlterValueFunc func(interface{}) (interface{}, error)

var starToRegexPatternFunc = AlterValueFunc(func(value interface{}) (interface{}, error) {
	v, ok := value.(string)
	if !ok {
		return nil, fmt.Errorf("unable to convert %v to string", value)
	}
	newValue := v
	if len(v) >= 3 && strings.HasPrefix(v, "*") && strings.HasSuffix(v, "*") {
		newValue = v[1 : len(v)-1]
	} else if len(v) >= 2 && strings.HasPrefix(v, "*") {
		newValue = v[1:] + "$"
	} else if len(v) >= 2 && strings.HasSuffix(v, "*") {
		newValue = "^" + v[0:len(v)-1]
	}
	return convert(newValue)
})

var ilikePatternFunc = AlterValueFunc(func(value interface{}) (interface{}, error) {
	v, ok := value.(string)
	if !ok {
		return nil, fmt.Errorf("unable to convert %v to string", value)
	}
	newVal, err := starToRegexPatternFunc(v)
	if err != nil {
		return nil, err
	}
	return fmt.Sprintf(`%v, "$options": "i"`, newVal), nil
})

var convert = AlterValueFunc(func(value interface{}) (interface{}, error) {
	switch v := value.(type) {
	case string:
		return quote(v), nil
	case time.Time:
		return newDateTimeFromTime(v), nil
	}
	return value, nil
})

func (mt *Translator) SetOpFunc(op string, f driver.TranslatorOpFunc) {
	mt.OpsDic[strings.ToUpper(op)] = f
}

func (mt *Translator) DeleteOpFunc(op string) {
	delete(mt.OpsDic, strings.ToUpper(op))
}

func (mt *Translator) Mongo() (mongo string, err error) {
	var where string
	where, err = mt.Where()
	if err != nil {
		return
	}
	if len(where) > 0 {
		mongo = where
	}
	sort := mt.Sort()
	if len(sort) > 0 {
		mongo += ", " + sort
	}
	limit := mt.Limit()
	if len(limit) > 0 {
		mongo += ", " + limit
	}
	offset := mt.Offset()
	if len(offset) > 0 {
		mongo += ", " + offset
	}
	return mongo, nil
}

func (mt *Translator) Where() (string, error) {
	if mt.rootNode == nil {
		return "", nil
	}
	return mt.where(mt.rootNode.Node)
}

func (mt *Translator) Sort() (sort string) {
	if mt.rootNode == nil {
		return
	}
	sep := ""
	for _, item := range mt.rootNode.Sort() {
		sort += sep
		direction := 1
		if item.Desc {
			direction = -1
		}
		sort += fmt.Sprintf(`"%s": %d`, item.By, direction)
		sep = ", "
	}
	if len(sort) > 0 {
		return fmt.Sprintf(`{"$sort": {%s}}`, sort)
	}
	return
}

func (mt *Translator) Limit() (limit string) {
	if mt.rootNode == nil {
		return
	}
	l := mt.rootNode.Limit()
	if l != "" && strings.ToUpper(l) != "INFINITY" {
		v, _ := strconv.Atoi(l)
		limit = fmt.Sprintf(`{"$limit": %d}`, v)
	}
	return
}

func (mt *Translator) Offset() (offset string) {
	if mt.rootNode != nil && mt.rootNode.Offset() != "" {
		v, _ := strconv.Atoi(mt.rootNode.Offset())
		offset = fmt.Sprintf(`{"$skip": %d}`, v)
	}
	return
}

func (mt *Translator) where(n *gorql.RqlNode) (string, error) {
	if n == nil {
		return ``, nil
	}
	f := mt.OpsDic[strings.ToUpper(n.Op)]
	if f == nil {
		return "", fmt.Errorf("no TranslatorOpFunc for op : '%s'", n.Op)
	}
	return f(n)
}

func NewMongoTranslator(r *gorql.RqlRootNode) (mt *Translator) {
	mt = &Translator{r, map[string]driver.TranslatorOpFunc{}}

	mt.SetOpFunc(driver.AndOp, mt.GetJoinTranslatorOpFunc(strings.ToLower(driver.AndOp)))
	mt.SetOpFunc(driver.OrOp, mt.GetJoinTranslatorOpFunc(strings.ToLower(driver.OrOp)))
	mt.SetOpFunc(driver.NeOp, mt.GetFieldValueTranslatorFunc(strings.ToLower(driver.NeOp), convert))
	mt.SetOpFunc(driver.EqOp, mt.GetFieldValueTranslatorFunc(strings.ToLower(driver.EqOp), convert))
	mt.SetOpFunc(driver.LikeOp, mt.GetFieldValueTranslatorFunc("regex", starToRegexPatternFunc))
	mt.SetOpFunc(driver.MatchOp, mt.GetFieldValueTranslatorFunc("regex", ilikePatternFunc))
	mt.SetOpFunc(driver.GtOp, mt.GetFieldValueTranslatorFunc(strings.ToLower(driver.GtOp), convert))
	mt.SetOpFunc(driver.LtOp, mt.GetFieldValueTranslatorFunc(strings.ToLower(driver.LtOp), convert))
	mt.SetOpFunc(driver.GeOp, mt.GetFieldValueTranslatorFunc("gte", convert))
	mt.SetOpFunc(driver.LeOp, mt.GetFieldValueTranslatorFunc("lte", convert))
	mt.SetOpFunc(driver.NotOp, mt.GetJoinTranslatorOpFunc("nor"))
	mt.SetOpFunc(driver.InOp, mt.GetSliceTranslatorFunc(strings.ToLower(driver.InOp), convert))
	return
}

func (mt *Translator) GetJoinTranslatorOpFunc(op string) driver.TranslatorOpFunc {
	return func(n *gorql.RqlNode) (s string, err error) {
		var ops []string
		for _, a := range n.Args {
			switch v := a.(type) {
			case *gorql.RqlNode:
				var tempS string
				tempS, err = mt.where(v)
				if err != nil {
					return "", err
				}
				ops = append(ops, tempS)
			default:
				return "", fmt.Errorf("%s operation need query as arguments", op)
			}
		}
		return fmt.Sprintf(`{"$%s": [%s]}`, op, strings.Join(ops, ", ")), nil
	}
}

func (mt *Translator) GetFieldValueTranslatorFunc(op string, alterValueFunc AlterValueFunc) driver.TranslatorOpFunc {
	return func(n *gorql.RqlNode) (s string, err error) {
		sep := ""
		for i, a := range n.Args {
			s += sep
			var tempS string
			if i == 0 {
				if gorql.IsValidField(a.(string)) {
					tempS = quote(a.(string))
				} else {
					return "", fmt.Errorf("first argument must be a valid field name (arg: %v)", a)
				}
			} else {
				convertedValue, err := alterValueFunc(a)
				if err != nil {
					return "", err
				}
				s += fmt.Sprintf(`{"$%s": %v}`, op, convertedValue)
			}
			s += tempS
			sep = fmt.Sprintf(`: `)
		}
		return fmt.Sprintf(`{%s}`, s), nil
	}
}

func (mt *Translator) GetSliceTranslatorFunc(op string, alterValueFunc AlterValueFunc) driver.TranslatorOpFunc {
	return func(n *gorql.RqlNode) (s string, err error) {
		var values []string
		var field string
		if len(n.Args) > 0 {
			a := n.Args[0]
			if gorql.IsValidField(a.(string)) {
				field = quote(a.(string))
			} else {
				return "", fmt.Errorf("first argument must be a valid field name (arg: %s)", a)
			}
		}
		subArgs := n.Args[1:]
		if len(subArgs) > 1 {
			return "", fmt.Errorf("expect enclosed arrays with square brackets argument")
		}
		groupNode, ok := subArgs[0].(*gorql.RqlNode)
		if !ok {
			return "", fmt.Errorf("expected group node but got %v", subArgs[0])
		}
		if len(groupNode.Args) < 2 {
			return "", fmt.Errorf("array of values not found")
		}
		for _, a := range groupNode.Args[1:] {
			convertedValue, err := alterValueFunc(a)
			if err != nil {
				return "", err
			}
			values = append(values, fmt.Sprintf("%v", convertedValue))
		}
		s += fmt.Sprintf(`%s: {"$%s": [%v]}`, field, op, strings.Join(values, ", "))
		return fmt.Sprintf(`{%s}`, s), nil
	}
}

func quote(s string) string {
	return `"` + strings.Replace(s, `"`, `""`, -1) + `"`
}

func newDateTimeFromTime(t time.Time) int64 {
	return t.Unix()*1e3 + int64(t.Nanosecond())/1e6
}
