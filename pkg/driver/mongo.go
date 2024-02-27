package driver

import (
	"fmt"
	"gorql"
	"strconv"
	"strings"
)

type MongoTranslator struct {
	rootNode *gorql.RqlRootNode
	OpsDic   map[string]TranslatorOpFunc
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
	return fmt.Sprintf(`%v, '$options': 'i'`, newVal), nil
})

var convert = AlterValueFunc(func(value interface{}) (interface{}, error) {
	switch v := value.(type) {
	case string:
		return Quote(v), nil
	}
	return value, nil
})

func (mt *MongoTranslator) SetOpFunc(op string, f TranslatorOpFunc) {
	mt.OpsDic[strings.ToUpper(op)] = f
}

func (mt *MongoTranslator) DeleteOpFunc(op string) {
	delete(mt.OpsDic, strings.ToUpper(op))
}

func (mt *MongoTranslator) Mongo() (mongo string, err error) {
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

func (mt *MongoTranslator) Where() (string, error) {
	if mt.rootNode == nil {
		return "", nil
	}
	return mt.where(mt.rootNode.Node)
}

func (mt *MongoTranslator) Sort() (sort string) {
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
		sort += fmt.Sprintf(`'%s': %d`, item.By, direction)
		sep = ", "
	}
	if len(sort) > 0 {
		return fmt.Sprintf("{'$sort': {%s}}", sort)
	}
	return
}

func (mt *MongoTranslator) Limit() (limit string) {
	if mt.rootNode == nil {
		return
	}
	l := mt.rootNode.Limit()
	if l != "" && strings.ToUpper(l) != "INFINITY" {
		v, _ := strconv.Atoi(l)
		limit = fmt.Sprintf("{'$limit': %d}", v)
	}
	return
}

func (mt *MongoTranslator) Offset() (offset string) {
	if mt.rootNode != nil && mt.rootNode.Offset() != "" {
		v, _ := strconv.Atoi(mt.rootNode.Offset())
		offset = fmt.Sprintf("{'$skip': %d}", v)
	}
	return
}

func (mt *MongoTranslator) where(n *gorql.RqlNode) (string, error) {
	if n == nil {
		return ``, nil
	}
	f := mt.OpsDic[strings.ToUpper(n.Op)]
	if f == nil {
		return "", fmt.Errorf("no TranslatorOpFunc for op : '%s'", n.Op)
	}
	return f(n)
}

func NewMongoTranslator(r *gorql.RqlRootNode) (mt *MongoTranslator) {
	mt = &MongoTranslator{r, map[string]TranslatorOpFunc{}}

	mt.SetOpFunc(AndOp, mt.GetAndOrTranslatorOpFunc(strings.ToLower(AndOp)))
	mt.SetOpFunc(OrOp, mt.GetAndOrTranslatorOpFunc(strings.ToLower(OrOp)))
	mt.SetOpFunc(NeOp, mt.GetFieldValueTranslatorFunc(strings.ToLower(NeOp), convert))
	mt.SetOpFunc(EqOp, mt.GetFieldValueTranslatorFunc(strings.ToLower(EqOp), convert))
	mt.SetOpFunc(LikeOp, mt.GetFieldValueTranslatorFunc("regex", starToRegexPatternFunc))
	mt.SetOpFunc(MatchOp, mt.GetFieldValueTranslatorFunc("regex", ilikePatternFunc))
	mt.SetOpFunc(GtOp, mt.GetFieldValueTranslatorFunc(strings.ToLower(GtOp), convert))
	mt.SetOpFunc(LtOp, mt.GetFieldValueTranslatorFunc(strings.ToLower(LtOp), convert))
	mt.SetOpFunc(GeOp, mt.GetFieldValueTranslatorFunc("gte", convert))
	mt.SetOpFunc(LeOp, mt.GetFieldValueTranslatorFunc("lte", convert))
	mt.SetOpFunc(NotOp, mt.GetOpFirstTranslatorFunc(strings.ToLower(NotOp)))
	return
}

func (mt *MongoTranslator) GetAndOrTranslatorOpFunc(op string) TranslatorOpFunc {
	return func(n *gorql.RqlNode) (s string, err error) {
		var ops []string
		for _, a := range n.Args {
			switch v := a.(type) {
			case string:
				if !IsValidField(v) {
					return "", fmt.Errorf("invalid field name : %s", v)
				}
				s = s + v
			case *gorql.RqlNode:
				var tempS string
				tempS, err = mt.where(v)
				if err != nil {
					return "", err
				}
				ops = append(ops, tempS)
			}
		}
		return fmt.Sprintf("{'$%s': [%s]}", op, strings.Join(ops, ", ")), nil
	}
}

func (mt *MongoTranslator) GetFieldValueTranslatorFunc(op string, alterValueFunc AlterValueFunc) TranslatorOpFunc {
	return func(n *gorql.RqlNode) (s string, err error) {
		sep := ""
		for i, a := range n.Args {
			s += sep
			switch v := a.(type) {
			case *gorql.RqlNode:
				var tempS string
				tempS, err = mt.where(v)
				if err != nil {
					return "", err
				}
				s = s + tempS
			default:
				var tempS string
				if i == 0 {
					if IsValidField(v.(string)) {
						tempS = Quote(v.(string))
					} else {
						return "", fmt.Errorf("first argument must be a valid field name (arg: %s)", v)
					}
				} else {
					convertedValue, err := alterValueFunc(v)
					if err != nil {
						return "", err
					}
					s += fmt.Sprintf("{'$%s': %v}", op, convertedValue)
				}
				s += tempS
			}
			sep = fmt.Sprintf(`: `)
		}
		return fmt.Sprintf(`{%s}`, s), nil
	}
}

func (mt *MongoTranslator) GetOpFirstTranslatorFunc(op string) TranslatorOpFunc {
	return func(n *gorql.RqlNode) (s string, err error) {
		sep := ""
		for _, a := range n.Args {
			s += sep
			switch v := a.(type) {
			case *gorql.RqlNode:
				var tempS string
				tempS, err = mt.where(v)
				if err != nil {
					return "", err
				}
				s = s + tempS
			default:
				convertedValue, err := convert(v)
				if err != nil {
					return "", err
				}
				s += fmt.Sprintf("%v", convertedValue)
			}
			sep = ", "
		}

		return fmt.Sprintf("{'$%s': %s}", op, s), nil
	}
}
