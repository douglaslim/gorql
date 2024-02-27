package driver

import (
	"fmt"
	"gorql"
	"strings"
)

type MongoTranslator struct {
	rootNode *gorql.RqlRootNode
	OpsDic   map[string]TranslatorOpFunc
}

type AlterValueFunc func(interface{}) (interface{}, error)

func (mt *MongoTranslator) SetOpFunc(op string, f TranslatorOpFunc) {
	mt.OpsDic[strings.ToUpper(op)] = f
}

func (mt *MongoTranslator) DeleteOpFunc(op string) {
	delete(mt.OpsDic, strings.ToUpper(op))
}

func (mt *MongoTranslator) Where() (string, error) {
	if mt.rootNode == nil {
		return "", nil
	}
	return mt.where(mt.rootNode.Node)
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
	mt.SetOpFunc(OrOp, mt.GetAndOrTranslatorOpFunc(OrOp))
	//mt.SetOpFunc(NeOp, mt.GetEqualityTranslatorOpFunc("!=", "IS NOT"))
	mt.SetOpFunc(EqOp, mt.GetEqualityTranslatorOpFunc(strings.ToLower(EqOp)))
	//mt.SetOpFunc(LikeOp, mt.GetFieldValueTranslatorFunc("LIKE", starToPercentFunc))
	//mt.SetOpFunc(MatchOp, mt.GetFieldValueTranslatorFunc("ILIKE", starToPercentFunc))
	mt.SetOpFunc(GtOp, mt.GetFieldValueTranslatorFunc(strings.ToLower(GtOp)))
	//mt.SetOpFunc(LtOp, mt.GetFieldValueTranslatorFunc("<", nil))
	//mt.SetOpFunc(GeOp, mt.GetFieldValueTranslatorFunc(">=", nil))
	//mt.SetOpFunc(LeOp, mt.GetFieldValueTranslatorFunc("<=", nil))
	mt.SetOpFunc(NotOp, mt.GetFieldValueTranslatorFunc(strings.ToLower(NotOp)))
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

func (mt *MongoTranslator) GetEqualityTranslatorOpFunc(op string) TranslatorOpFunc {
	return func(n *gorql.RqlNode) (s string, err error) {
		return mt.GetFieldValueTranslatorFunc(op)(n)
	}
}

func (mt *MongoTranslator) GetFieldValueTranslatorFunc(op string) TranslatorOpFunc {
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
					convertedValue, err := convert(v)
					if err != nil {
						return "", err
					}
					s += fmt.Sprintf("%v", convertedValue)
				}
				s += tempS
			}
			sep = fmt.Sprintf(": ")
		}
		return fmt.Sprintf(`{'$%s': {%s}}`, op, s), nil
	}
}

//func (mt *MongoTranslator) GetOpFirstTranslatorFunc(op string) TranslatorOpFunc {
//	return func(n *gorql.RqlNode) (s string, err error) {
//		sep := ""
//		for _, a := range n.Args {
//			s += sep
//			switch v := a.(type) {
//			case string:
//				var tempS string
//				_, err := strconv.ParseInt(v, 10, 64)
//				if err == nil || IsValidField(v) {
//					tempS = v
//				} else if valueAlterFunc != nil {
//					tempS, err = valueAlterFunc(v)
//					if err != nil {
//						return "", err
//					}
//				} else {
//					tempS = Quote(v)
//				}
//				s += tempS
//			case *gorql.RqlNode:
//				var tempS string
//				tempS, err = mt.where(v)
//				if err != nil {
//					return "", err
//				}
//				s = s + tempS
//			}
//
//			sep = ", "
//		}
//
//		return fmt.Sprintf("{'$%s': %s}", op, strings.Join(ops, ", ")), nil
//	}
//}

func convert(value interface{}) (interface{}, error) {
	switch v := value.(type) {
	case string:
		return Quote(v), nil
	}
	return value, nil
}
