package gorql

import (
	"errors"
	"github.com/iancoleman/strcase"
	"log"
	"reflect"
)

const (
	DefaultTagName  = "rql"
	DefaultFieldSep = "_"
	DefaultLimit    = 25
	DefaultMaxLimit = 100
)

// Config is the configuration for the parser.
type Config struct {
	// TagName is an optional tag name for configuration. t defaults to "rql".
	TagName string
	// Model is the resource definition. The parser is configured based on its definition.
	// For example, given the following struct definition:
	//
	//	type User struct {
	//		Age	 int	`rql:"filter,sort"`
	// 		Name string	`rql:"filter"`
	// 	}
	//
	// In order to create a parser for the given resource, you will do it like so:
	//
	//	var QueryParser = rql.NewParser(&rql.Config{
	// 		Model: User{},
	// 	})
	//
	Model interface{}
	// FieldSep is the separator for nested fields in a struct. For example, given the following struct:
	//
	//	type User struct {
	// 		Name 	string	`rql:"filter"`
	//		Address	struct {
	//			City string `rql:"filter"``
	//		}
	// 	}
	//
	// We assume the schema for this struct contains a column named "address_city". Therefore, the default
	// separator is underscore ("_"). But, you can change it to "." for convenience or readability reasons.
	// The parser will automatically convert it to underscore ("_"). If you want to control the name of
	// the column, use the "column" option in the struct definition. For example:
	//
	//	type User struct {
	// 		Name 	string	`rql:"filter,column=full_name"`
	// 	}
	//
	FieldSep string
	// ColumnFn is the function that translate the struct field string into a table column.
	// For example, given the following fields and their column names:
	//
	//	FullName => "full_name"
	// 	HTTPPort => "http_port"
	//
	// It is preferred that you will follow the same convention that your ORM or other DB helper use.
	// For example, If you are using `gorm` you want to se this option like this:
	//
	//	var QueryParser = rql.MustNewParser(
	// 		ColumnFn: gorm.ToDBName,
	// 	})
	//
	ColumnFn func(string) string
	// Log the logging function used to log debug information in the initialization of the parser.
	// It defaults `to log.Printf`.
	Log func(string, ...interface{})
	// DefaultLimit is the default value for the `Limit` field that returns when no limit supplied by the caller.
	// It defaults to 25.
	DefaultLimit int
	// LimitMaxValue is the upper boundary for the limit field. User will get an error if the given value is greater
	// than this value. It defaults to 100.
	LimitMaxValue int
}

// defaults sets the default configuration of Config.
func (c *Config) defaults() error {
	if c.Model == nil {
		return errors.New("rql: 'Model' is a required field")
	}
	if indirect(reflect.TypeOf(c.Model)).Kind() != reflect.Struct {
		return errors.New("rql: 'Model' must be a struct type")
	}
	if c.Log == nil {
		c.Log = log.Printf
	}
	if c.ColumnFn == nil {
		c.ColumnFn = Column
	}
	defaultString(&c.TagName, DefaultTagName)
	defaultString(&c.FieldSep, DefaultFieldSep)
	defaultInt(&c.DefaultLimit, DefaultLimit)
	defaultInt(&c.LimitMaxValue, DefaultMaxLimit)
	return nil
}

// Column is the default function that transform field name into column name.
// It used to convert the struct fields into lower camelcase. For example:
//
//	Username => username
//	FullName => fullName
//	HTTPCode => httpcode
func Column(s string) string {
	return strcase.ToLowerCamel(s)
}

func defaultString(s *string, v string) {
	if *s == "" {
		*s = v
	}
}

func defaultInt(i *int, v int) {
	if *i == 0 {
		*i = v
	}
}

// indirect returns the item at the end of indirection.
func indirect(t reflect.Type) reflect.Type {
	for ; t.Kind() == reflect.Ptr; t = t.Elem() {
	}
	return t
}
