package gorql

import (
	"database/sql"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"
)

func FuzzParse(f *testing.F) {
	f.Fuzz(func(t *testing.T, a string) {
		p, err := NewParser(nil)
		if err != nil {
			t.Fatalf("New parser error :%s", err)
		}
		_, _ = p.Parse(strings.NewReader(a))
	})
}

type ParseURLTest struct {
	Name           string      // Name of the test
	URL            string      // Input URL
	WantParseError bool        // Test should raise an error when parsing the RQL query
	Model          interface{} // Input Model for query
}

type ParseFieldTest struct {
	Name        string
	StructField reflect.StructField
	WantError   bool
}

var parseURLTests = []ParseURLTest{
	{
		Name:           `Basic URL parse`,
		URL:            `http://localhost:8000?and(eq(foo,42),gt(price,10),not(disabled=false))`,
		WantParseError: false,
		Model: new(struct {
			Foo      string  `rql:"filter"`
			Price    float64 `rql:"filter"`
			Disabled bool    `rql:"filter"`
		}),
	},
	{
		Name:           `Basic URL parse with space encoding`,
		URL:            `http://localhost:8000?eq(foo,john%20wick%20is%20back)`,
		WantParseError: false,
		Model: new(struct {
			Foo string `rql:"filter"`
		}),
	},
	{
		Name:           `URL parse with no space encoding`,
		URL:            `http://localhost:8000?eq(foo,john wick is back)`,
		WantParseError: false,
		Model: new(struct {
			Foo string `rql:"filter"`
		}),
	},
	{
		Name:           `URL parse with invalid RQL query (missing closing bracket)`,
		URL:            `http://localhost:8000?and(eq(foo,john wick is back)`,
		WantParseError: true,
		Model: new(struct {
			Foo string `rql:"filter"`
		}),
	},
}

func TestParseURL(t *testing.T) {
	for _, test := range parseURLTests {
		test.Run(t)
	}
}

func TestParseField(t *testing.T) {
	for _, test := range parseFieldTests {
		test.Run(t)
	}
}

var parseFieldTests = []ParseFieldTest{
	{
		Name: "Basic sortable field",
		StructField: reflect.StructField{
			Name: "Foo",
			Tag:  `rql:"sort"`,
			Type: reflect.TypeOf(""),
		},
		WantError: false,
	},
	{
		Name: "Basic filterable field",
		StructField: reflect.StructField{
			Name: "Price",
			Tag:  `rql:"filter"`,
			Type: reflect.TypeOf(0.0),
		},
		WantError: false,
	},
	{
		Name: "Field with replacewith option",
		StructField: reflect.StructField{
			Name: "Bar",
			Tag:  `rql:"replacewith=bar_column"`,
			Type: reflect.TypeOf(""),
		},
		WantError: false,
	},
	{
		Name: "Field with invalid layout",
		StructField: reflect.StructField{
			Name: "CreatedAt",
			Tag:  `rql:"layout=invalid_layout"`,
			Type: reflect.TypeOf(time.Time{}),
		},
		WantError: true,
	},
	{
		Name: "Field with valid layout",
		StructField: reflect.StructField{
			Name: "CreatedAt",
			Tag:  `rql:"layout=2006-01-02T15:04:05Z07:00"`,
			Type: reflect.TypeOf(time.Time{}),
		},
		WantError: false,
	},
	{
		Name: "Unsupported field type",
		StructField: reflect.StructField{
			Name: "Unsupported",
			Tag:  `rql:"filter"`,
			Type: reflect.TypeOf([]int{}),
		},
		WantError: true,
	},
	{
		Name: "Field with column option",
		StructField: reflect.StructField{
			Name: "Baz",
			Tag:  `rql:"column=baz_column"`,
			Type: reflect.TypeOf(""),
		},
		WantError: false,
	},
	{
		Name: "Bool field",
		StructField: reflect.StructField{
			Name: "BoolField",
			Tag:  `rql:"filter"`,
			Type: reflect.TypeOf(true),
		},
		WantError: false,
	},
	{
		Name: "String field",
		StructField: reflect.StructField{
			Name: "StringField",
			Tag:  `rql:"filter"`,
			Type: reflect.TypeOf(""),
		},
		WantError: false,
	},
	{
		Name: "Int field",
		StructField: reflect.StructField{
			Name: "IntField",
			Tag:  `rql:"filter"`,
			Type: reflect.TypeOf(int(0)),
		},
		WantError: false,
	},
	{
		Name: "Int8 field",
		StructField: reflect.StructField{
			Name: "Int8Field",
			Tag:  `rql:"filter"`,
			Type: reflect.TypeOf(int8(0)),
		},
		WantError: false,
	},
	{
		Name: "Int16 field",
		StructField: reflect.StructField{
			Name: "Int16Field",
			Tag:  `rql:"filter"`,
			Type: reflect.TypeOf(int16(0)),
		},
		WantError: false,
	},
	{
		Name: "Int32 field",
		StructField: reflect.StructField{
			Name: "Int32Field",
			Tag:  `rql:"filter"`,
			Type: reflect.TypeOf(int32(0)),
		},
		WantError: false,
	},
	{
		Name: "Int64 field",
		StructField: reflect.StructField{
			Name: "Int64Field",
			Tag:  `rql:"filter"`,
			Type: reflect.TypeOf(int64(0)),
		},
		WantError: false,
	},
	{
		Name: "Uint field",
		StructField: reflect.StructField{
			Name: "UintField",
			Tag:  `rql:"filter"`,
			Type: reflect.TypeOf(uint(0)),
		},
		WantError: false,
	},
	{
		Name: "Uint8 field",
		StructField: reflect.StructField{
			Name: "Uint8Field",
			Tag:  `rql:"filter"`,
			Type: reflect.TypeOf(uint8(0)),
		},
		WantError: false,
	},
	{
		Name: "Uint16 field",
		StructField: reflect.StructField{
			Name: "Uint16Field",
			Tag:  `rql:"filter"`,
			Type: reflect.TypeOf(uint16(0)),
		},
		WantError: false,
	},
	{
		Name: "Uint32 field",
		StructField: reflect.StructField{
			Name: "Uint32Field",
			Tag:  `rql:"filter"`,
			Type: reflect.TypeOf(uint32(0)),
		},
		WantError: false,
	},
	{
		Name: "Uint64 field",
		StructField: reflect.StructField{
			Name: "Uint64Field",
			Tag:  `rql:"filter"`,
			Type: reflect.TypeOf(uint64(0)),
		},
		WantError: false,
	},
	{
		Name: "Uintptr field",
		StructField: reflect.StructField{
			Name: "UintptrField",
			Tag:  `rql:"filter"`,
			Type: reflect.TypeOf(uintptr(0)),
		},
		WantError: false,
	},
	{
		Name: "Float32 field",
		StructField: reflect.StructField{
			Name: "Float32Field",
			Tag:  `rql:"filter"`,
			Type: reflect.TypeOf(float32(0)),
		},
		WantError: false,
	},
	{
		Name: "Float64 field",
		StructField: reflect.StructField{
			Name: "Float64Field",
			Tag:  `rql:"filter"`,
			Type: reflect.TypeOf(float64(0)),
		},
		WantError: false,
	},
	{
		Name: "Slice of strings field",
		StructField: reflect.StructField{
			Name: "StringSliceField",
			Tag:  `rql:"filter"`,
			Type: reflect.TypeOf([]string{}),
		},
		WantError: false,
	},
	{
		Name: "Unsupported slice element type",
		StructField: reflect.StructField{
			Name: "UnsupportedSliceField",
			Tag:  `rql:"filter"`,
			Type: reflect.TypeOf([]int{}),
		},
		WantError: true,
	},
	{
		Name: "sql.NullBool field",
		StructField: reflect.StructField{
			Name: "NullBoolField",
			Tag:  `rql:"filter"`,
			Type: reflect.TypeOf(sql.NullBool{}),
		},
		WantError: false,
	},
	{
		Name: "sql.NullString field",
		StructField: reflect.StructField{
			Name: "NullStringField",
			Tag:  `rql:"filter"`,
			Type: reflect.TypeOf(sql.NullString{}),
		},
		WantError: false,
	},
	{
		Name: "sql.NullInt64 field",
		StructField: reflect.StructField{
			Name: "NullInt64Field",
			Tag:  `rql:"filter"`,
			Type: reflect.TypeOf(sql.NullInt64{}),
		},
		WantError: false,
	},
	{
		Name: "sql.NullFloat64 field",
		StructField: reflect.StructField{
			Name: "NullFloat64Field",
			Tag:  `rql:"filter"`,
			Type: reflect.TypeOf(sql.NullFloat64{}),
		},
		WantError: false,
	},
	{
		Name: "Convertible to time.Time field",
		StructField: reflect.StructField{
			Name: "ConvertibleTimeField",
			Tag:  `rql:"filter"`,
			Type: reflect.TypeOf(time.Time{}),
		},
		WantError: false,
	},
}

func (p ParseURLTest) Run(t *testing.T) {
	parser, err := NewParser(&Config{Model: p.Model})
	if err != nil {
		t.Fatalf("(%s) New parser error :%v\n", p.Name, err)
	}
	u, _ := url.Parse(p.URL)
	_, err = parser.ParseURL(u.Query())
	if p.WantParseError != (err != nil) {
		t.Fatalf("(%s) Expecting parse error :%v\nGot error : %v", p.Name, p.WantParseError, err)
	}
}

func (p ParseFieldTest) Run(t *testing.T) {
	parser, err := NewParser(&Config{TagName: "rql", Model: struct{}{}})
	if err != nil {
		t.Fatalf("New parser error: %v", err)
	}
	err = parser.parseField(p.StructField)
	if p.WantError != (err != nil) {
		t.Fatalf("(%s) Expecting error: %v, got: %v", p.Name, p.WantError, err)
	}
}
