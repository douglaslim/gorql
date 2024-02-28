package mongo

import (
	"gorql"
	"strings"
	"testing"
	"time"
)

type MongodbTest struct {
	Name                string      // Name of the test
	RQL                 string      // Input RQL query
	Expected            string      // Expected Output Expected
	WantParseError      bool        // Test should raise an error when parsing the RQL query
	WantTranslatorError bool        // Test should raise an error when translating to Expected
	Model               interface{} // Input Model for query
}

func (test *MongodbTest) Run(t *testing.T) {
	p, err := gorql.NewParser(&gorql.Config{Model: test.Model})
	if err != nil {
		t.Fatalf("(%s) New parser error :%v\n", test.Name, err)
	}
	rqlNode, err := p.Parse(strings.NewReader(test.RQL))
	if test.WantParseError != (err != nil) {
		t.Fatalf("(%s) Expecting error :%v\nGot error : %v", test.Name, test.WantParseError, err)
	}
	mongoTranslator := NewMongoTranslator(rqlNode)
	s, err := mongoTranslator.Where()
	if test.WantTranslatorError != (err != nil) {
		t.Fatalf("(%s) Expecting error :%v\nGot error : %v \n\tQuery = %s", test.Name, test.WantTranslatorError, err, s)
	}

	if s != test.Expected {
		t.Fatalf("(%s) Translated Mongo query doesn’t match the expected one %s vs %s", test.Name, s, test.Expected)
	}
}

var mongodbTests = []MongodbTest{
	{
		Name:                `Basic translation with double equal operators`,
		RQL:                 `and(foo=eq=42,price=eq=10)`,
		Expected:            `{"$and": [{"foo": {"$eq": "42"}}, {"price": {"$eq": 10}}]}`,
		WantParseError:      false,
		WantTranslatorError: false,
		Model: new(struct {
			Foo   string  `rql:"filter"`
			Price float64 `rql:"filter"`
		}),
	},
	{
		Name:                `Basic translation with func style operators`,
		RQL:                 `and(eq(foo,42),gt(price,10),not(disabled=false))`,
		Expected:            `{"$and": [{"foo": {"$eq": "42"}}, {"price": {"$gt": 10}}, {"$not": {"disabled": {"$eq": false}}}]}`,
		WantParseError:      false,
		WantTranslatorError: false,
		Model: new(struct {
			Foo      string  `rql:"filter"`
			Price    float64 `rql:"filter"`
			Disabled bool    `rql:"filter"`
		}),
	},
	{
		Name:                `Basic translation with func simple equal operators`,
		RQL:                 `foo=42&price=10`,
		Expected:            `{"$and": [{"foo": {"$eq": "42"}}, {"price": {"$eq": 10}}]}`,
		WantParseError:      false,
		WantTranslatorError: false,
		Model: new(struct {
			Foo   string  `rql:"filter"`
			Price float64 `rql:"filter"`
		}),
	},
	{
		Name:                `Basic translation with LIKE operator`,
		RQL:                 `foo=like=weird`,
		Expected:            `{"foo": {"$regex": "weird"}}`,
		WantParseError:      false,
		WantTranslatorError: false,
		Model: new(struct {
			Foo string `rql:"filter"`
		}),
	},
	{
		Name:                `Basic translation with ILIKE operator`,
		RQL:                 `foo=match=john%20doe`,
		Expected:            `{"foo": {"$regex": "john doe", "$options": "i"}}`,
		WantParseError:      false,
		WantTranslatorError: false,
		Model: new(struct {
			Foo string `rql:"filter"`
		}),
	},
	{
		Name:                `Basic translation with IN Operator`,
		RQL:                 `in(foo,hello,this%20is%20low,wow)`,
		Expected:            `{"foo": {"$in": ["hello", "this is low", "wow"]}}`,
		WantParseError:      false,
		WantTranslatorError: false,
		Model: new(struct {
			Foo string `rql:"filter"`
		}),
	},
	{
		Name:                `Mixed style translation`,
		RQL:                 `((eq(foo,42)&ge(price,10))|price=ge=500)&disabled=eq=false`,
		Expected:            `{"$and": [{"$or": [{"$and": [{"foo": {"$eq": "42"}}, {"price": {"$gte": 10}}]}, {"price": {"$gte": 500}}]}, {"disabled": {"$eq": false}}]}`,
		WantParseError:      false,
		WantTranslatorError: false,
		Model: new(struct {
			Foo      string  `rql:"filter"`
			Price    float64 `rql:"filter"`
			Disabled bool    `rql:"filter"`
		}),
	},
	{
		Name:                `Translation with date fields`,
		RQL:                 `now=gt=2018-01-01`,
		Expected:            `{"now": {"$gt": 1514764800000}}`,
		WantParseError:      false,
		WantTranslatorError: false,
		Model: new(struct {
			Now time.Time `rql:"filter,layout=2006-01-02"`
		}),
	},
	{
		Name:                `Empty RQL`,
		RQL:                 ``,
		Expected:            ``,
		WantParseError:      false,
		WantTranslatorError: false,
		Model:               new(struct{}),
	},
	{
		Name:                `Invalid RQL query (Unmanaged RQL operator)`,
		RQL:                 `foo=missing_operator=42`,
		Expected:            ``,
		WantParseError:      false,
		WantTranslatorError: true,
		Model: new(struct {
			Foo string `rql:"filter"`
		}),
	},
	{
		Name:                `Invalid RQL query (Unescaped character)`,
		RQL:                 `like(foo,hello world)`,
		Expected:            ``,
		WantParseError:      true,
		WantTranslatorError: false,
		Model: new(struct {
			Foo string `rql:"filter"`
		}),
	},
	{
		Name:                `Invalid RQL query (Missing comma)`,
		RQL:                 `and(not(test=weird),eq(foo,toto)gt(price,10))`,
		Expected:            ``,
		WantParseError:      true,
		WantTranslatorError: false,
		Model:               new(struct{}),
	},
	{
		Name:                `Invalid RQL query (Invalid field name)`,
		RQL:                 `eq(foo%20tot,42)`,
		Expected:            ``,
		WantParseError:      true,
		WantTranslatorError: false,
		Model:               new(struct{}),
	},
	{
		Name:                `Invalid RQL query (Invalid field name 2)`,
		RQL:                 `eq(foo*,toto)`,
		Expected:            ``,
		WantParseError:      true,
		WantTranslatorError: false,
		Model:               new(struct{}),
	},
}

func TestMongodbParser(t *testing.T) {
	for _, test := range mongodbTests {
		test.Run(t)
	}
}
