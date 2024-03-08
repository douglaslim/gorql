package cosmos

import (
	"github.com/douglaslim/gorql"
	"reflect"
	"strings"
	"testing"
)

type Test struct {
	Name                string      // Name of the test
	Model               interface{} // Input Model for query
	RQL                 string      // Input RQL query
	ExpectedSQL         string      // Expected Output ExpectedSQL
	ExpectedArgs        []interface{}
	WantParseError      bool // Test should raise an error when parsing the RQL query
	WantTranslatorError bool // Test should raise an error when translating to ExpectedSQL
}

func (test *Test) Run(t *testing.T) {
	p, err := gorql.NewParser(&gorql.Config{
		Model: test.Model,
	})
	if err != nil {
		t.Fatalf("(%s) New parser error :%v\n", test.Name, err)
	}
	rqlNode, err := p.Parse(strings.NewReader(test.RQL))
	if test.WantParseError != (err != nil) {
		t.Fatalf("(%s) Expecting error :%v\nGot error : %v", test.Name, test.WantParseError, err)
	}
	cosmosTranslator := NewCosmosTranslator(rqlNode)
	s, err := cosmosTranslator.Sql()
	if test.WantTranslatorError != (err != nil) {
		t.Fatalf("(%s) Expecting error :%v\nGot error : %v \n\tSQL = %s", test.Name, test.WantTranslatorError, err, s)
	}
	if s != test.ExpectedSQL {
		t.Fatalf("(%s) Translated SQL doesn’t match the expected one %s vs %s", test.Name, s, test.ExpectedSQL)
	}
	if len(test.ExpectedArgs) > 0 {
		if len(test.ExpectedArgs) != len(cosmosTranslator.Args()) {
			t.Fatalf("(%s) Length of expected arguments does not match with translated arguments\nExpected length: %d but got length %d", test.Name, len(test.ExpectedArgs), len(cosmosTranslator.Args()))
		}
		for i := range test.ExpectedArgs {
			if !reflect.DeepEqual(test.ExpectedArgs[i], cosmosTranslator.Args()[i]) {
				t.Fatalf("(%s) Translated arguments doesn’t match the expected one %v vs %v", test.Name, test.ExpectedArgs[i], cosmosTranslator.Args()[i])
			}
		}
	}
}

var tests = []Test{
	{
		Name: `Basic translation with func style operators`,
		RQL:  `and(eq(foo,42),gt(price,10),not(disabled=false))`,
		Model: new(struct {
			Foo      string  `rql:"filter"`
			Price    float64 `rql:"filter"`
			Disabled bool    `rql:"filter"`
		}),
		ExpectedSQL: `WHERE ((c.foo = @p1) AND (c.price > @p2) AND NOT((c.disabled = @p3)))`,
		ExpectedArgs: []interface{}{
			Param{
				Name:  "@p1",
				Value: "42",
			},
			Param{
				Name:  "@p2",
				Value: float64(10),
			},
			Param{
				Name:  "@p3",
				Value: false,
			},
		},
		WantParseError:      false,
		WantTranslatorError: false,
	},
	{
		Name: `Basic translation with func simple equal operators`,
		RQL:  `foo=42&price=10`,
		Model: new(struct {
			Foo   string  `rql:"filter"`
			Price float64 `rql:"filter"`
		}),
		ExpectedSQL: `WHERE ((c.foo = @p1) AND (c.price = @p2))`,
		ExpectedArgs: []interface{}{
			Param{
				Name:  "@p1",
				Value: "42",
			},
			Param{
				Name:  "@p2",
				Value: float64(10),
			},
		},
		WantParseError:      false,
		WantTranslatorError: false,
	},
	{
		Name: `Sort and limit`,
		RQL:  `eq(foo,42)&sort(+price,-length)&limit(10,20)`,
		Model: new(struct {
			Foo    string  `rql:"filter"`
			Price  float64 `rql:"sort"`
			Length int     `rql:"sort"`
		}),
		ExpectedSQL: `WHERE ((c.foo = @p1)) ORDER BY c.price, c.length DESC`,
		ExpectedArgs: []interface{}{
			Param{
				Name:  "@p1",
				Value: "42",
			},
		},
		WantParseError:      false,
		WantTranslatorError: false,
	},
	{
		Name: `Sort only`,
		RQL:  `sort(-price)`,
		Model: new(struct {
			Price float64 `rql:"sort"`
		}),
		ExpectedSQL:         ` ORDER BY c.price DESC`,
		WantParseError:      false,
		WantTranslatorError: false,
	},
	{
		Name: `LIKE empty string`,
		RQL:  `like(foo,)`,
		Model: new(struct {
			Foo string `rql:"filter"`
		}),
		ExpectedSQL:         `WHERE (c.foo LIKE '')`,
		WantParseError:      false,
		WantTranslatorError: false,
	},
	{
		Name: `Mixed style translation`,
		RQL:  `((eq(foo,42)&gt(price,10))|ge(price,500))&eq(disabled,false)`,
		Model: new(struct {
			Foo      string  `rql:"filter"`
			Price    float64 `rql:"filter"`
			Disabled bool    `rql:"filter"`
		}),
		ExpectedSQL: `WHERE ((((c.foo = @p1) AND (c.price > @p2)) OR (c.price >= @p3)) AND (c.disabled = @p4))`,
		ExpectedArgs: []interface{}{
			Param{
				Name:  "@p1",
				Value: "42",
			},
			Param{
				Name:  "@p2",
				Value: float64(10),
			},
			Param{
				Name:  "@p3",
				Value: float64(500),
			},
			Param{
				Name:  "@p4",
				Value: false,
			},
		},
		WantParseError:      false,
		WantTranslatorError: false,
	},
	{
		Name: `Try a simple SQL injection`,
		RQL:  `like(foo,toto%27%3BSELECT%20column%20IN%20table)`,
		Model: new(struct {
			Foo string `rql:"filter"`
		}),
		ExpectedSQL: `WHERE (c.foo LIKE @p1)`,
		ExpectedArgs: []interface{}{
			Param{
				Name:  "@p1",
				Value: "toto';SELECT column IN table",
			},
		},
		WantParseError:      false,
		WantTranslatorError: false,
	},
	{
		Name:                `Empty RQL`,
		RQL:                 ``,
		Model:               new(struct{}),
		ExpectedSQL:         ``,
		WantParseError:      false,
		WantTranslatorError: false,
	},
	{
		Name: `Invalid RQL query (Unmanaged RQL operator)`,
		RQL:  `missing_operator(foo,42)`,
		Model: new(struct {
			Foo string `rql:"filter"`
		}),
		ExpectedSQL:         ``,
		WantParseError:      false,
		WantTranslatorError: true,
	},
	{
		Name:                `Invalid RQL query (Unescaped character)`,
		RQL:                 `like(foo,hello world)`,
		Model:               new(struct{}),
		ExpectedSQL:         ``,
		WantParseError:      true,
		WantTranslatorError: false,
	},
	{
		Name:                `Invalid RQL query (Missing comma)`,
		RQL:                 `and(not(test),eq(foo,toto)gt(price,10))`,
		Model:               new(struct{}),
		ExpectedSQL:         ``,
		WantParseError:      true,
		WantTranslatorError: false,
	},
	{
		Name: `Invalid RQL query (Invalid field name)`,
		RQL:  `eq(foo%20tot,42)`,
		Model: new(struct {
			Foo string `rql:"filter,column=foo tot"`
		}),
		ExpectedSQL:         ``,
		WantParseError:      false,
		WantTranslatorError: true,
	},
	{
		Name: `Invalid RQL query (Invalid field name 2)`,
		RQL:  `eq(foo*,toto)`,
		Model: new(struct {
			Foo string `rql:"filter,column=foo*"`
		}),
		ExpectedSQL:         ``,
		WantParseError:      false,
		WantTranslatorError: true,
	},
}

func TestParser(t *testing.T) {
	for _, test := range tests {
		test.Run(t)
	}
}
