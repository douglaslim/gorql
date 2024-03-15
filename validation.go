package gorql

import (
	"errors"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"time"
)

type ValidationFunc func(*RqlNode) error

func errorType(v interface{}, expected string) error {
	actual := "nil"
	if v != nil {
		actual = reflect.TypeOf(v).Kind().String()
	}
	return fmt.Errorf("expect <%s>, got <%s>", expected, actual)
}

// validate that the underlined element of given interface is a boolean.
func validateBool(v interface{}) error {
	if _, ok := v.(bool); !ok {
		return errorType(v, "bool")
	}
	return nil
}

// validate that the underlined element of given interface is a string.
func validateString(v interface{}) error {
	if _, ok := v.(string); !ok {
		return errorType(v, "string")
	}
	return nil
}

// validate that the underlined element of given interface is a float.
func validateFloat(v interface{}) error {
	if _, ok := v.(float64); !ok {
		return errorType(v, "float64")
	}
	return nil
}

// validate that the underlined element of given interface is an int.
func validateInt(v interface{}) error {
	n, ok := v.(float64)
	if !ok {
		return errorType(v, "int")
	}
	if math.Trunc(n) != n {
		return errors.New("not an integer")
	}
	return nil
}

// validate that the underlined element of given interface is an int and greater than 0.
func validateUInt(v interface{}) error {
	if err := validateInt(v); err != nil {
		return err
	}
	if v.(float64) < 0 {
		return errors.New("not an unsigned integer")
	}
	return nil
}

// validate that the underlined element of this interface is a "datetime" string.
func validateTime(layout string) func(interface{}) error {
	return func(v interface{}) error {
		s, ok := v.(string)
		if !ok {
			return errorType(v, "string")
		}
		_, err := time.Parse(layout, s)
		return err
	}
}

func (p *Parser) validateFields(n *RqlNode) error {
	if n == nil {
		return nil
	}
	fn := p.GetFieldValidationFunc()
	if fn == nil {
		return fmt.Errorf("no field validation op '%s'", n.Op)
	}
	return fn(n)
}

func (p *Parser) GetFieldValidationFunc() ValidationFunc {
	return func(n *RqlNode) (err error) {
		var field *field
		for i, a := range n.Args {
			switch v := a.(type) {
			case string:
				if i == 0 {
					f, ok := p.fields[v]
					if !ok || !f.Filterable {
						return fmt.Errorf("field name (arg: %s) is not filterable", v)
					}
					field = f
					if field.ReplaceWith != "" {
						n.Args[i] = field.ReplaceWith
					}
				} else {
					if field == nil {
						return fmt.Errorf("no field is found for node value %s", v)
					}
					newVal, err := field.CovertFn(v)
					if err != nil {
						return fmt.Errorf("encounter field error: %s", err)
					}
					n.Args[i] = newVal
				}
			case *RqlNode:
				if v.Op == "group" {
					continue
				}
				err = p.validateFields(v)
				if err != nil {
					return err
				}
			}
		}
		return nil
	}
}

func (p *Parser) validateSpecialOps(r *RqlRootNode) error {
	if r.Limit() != "" {
		err := p.validateLimit(r.Limit())
		if err != nil {
			return err
		}
	}
	if r.Offset() != "" {
		err := p.validateOffset(r.Offset())
		if err != nil {
			return err
		}
	}
	if p.c != nil && len(r.Sort()) > 0 {
		err := p.validateSort(r.Sort())
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *Parser) validateSort(sortItems []Sort) error {
	for _, s := range sortItems {
		f, ok := p.fields[s.By]
		if !ok || !f.Sortable {
			return fmt.Errorf("field %s is not sortable", s.By)
		}
	}
	return nil
}

func (p *Parser) validateOffset(o string) error {
	offset, err := strconv.Atoi(o)
	if err != nil {
		return fmt.Errorf("invalid format for offset: %s", err)
	}
	if offset < 0 {
		return fmt.Errorf("offset is less than zero")
	}
	return nil
}

func (p *Parser) validateLimit(l string) error {
	limit, err := strconv.Atoi(l)
	if err != nil {
		return fmt.Errorf("invalid format for limit: %s", err)
	}
	if limit < 0 {
		return fmt.Errorf("specified limit is less than zero")
	}
	if p.c != nil && limit > p.c.LimitMaxValue {
		return fmt.Errorf("specified limit is more than the max limit %d allowed", p.c.DefaultLimit)
	}
	return nil
}

func IsValidField(s string) bool {
	for _, ch := range s {
		if !IsLetter(ch) && !IsDigit(ch) && ch != '_' && ch != '-' && ch != '.' {
			return false
		}
	}
	return true
}
