package driver

import (
	"gorql"
	"strings"
	"testing"
)

type MongodbTest struct {
	Name                string // Name of the test
	RQL                 string // Input RQL query
	Expected            string // Expected Output Expected
	WantParseError      bool   // Test should raise an error when parsing the RQL query
	WantTranslatorError bool   // Test should raise an error when translating to Expected
}

type MongoModel struct {
	Foo      string  `rql:"filter"`
	Price    float64 `rql:"filter"`
	Disabled bool    `rql:"filter"`
}

func (test *MongodbTest) Run(t *testing.T) {
	p, err := gorql.NewParser(&gorql.Config{Model: MongoModel{}})
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
		t.Fatalf("(%s) Expecting error :%v\nGot error : %v \n\tSQL = %s", test.Name, test.WantTranslatorError, err, s)
	}

	if s != test.Expected {
		t.Fatalf("(%s) Translated Mongo query doesn’t match the expected one %s vs %s", test.Name, s, test.Expected)
	}
}

var mongodbTests = []MongodbTest{
	{
		Name:                `Basic translation with double equal operators`,
		RQL:                 `and(foo=eq=42,price=eq=10)`,
		Expected:            `{'$and': [{'foo': {'$eq': '42'}}, {'price': {'$eq': 10}}]}`,
		WantParseError:      false,
		WantTranslatorError: false,
	},
	{
		Name:                `Basic translation with func style operators`,
		RQL:                 `and(eq(foo,42),gt(price,10),not(disabled=false))`,
		Expected:            `{'$and': [{'foo': {'$eq': '42'}}, {'price': {'$gt': 10}}, {'$not': {'disabled': {'$eq': false}}}]}`,
		WantParseError:      false,
		WantTranslatorError: false,
	},
}

func TestMongodbParser(t *testing.T) {
	for _, test := range mongodbTests {
		test.Run(t)
	}
}
