package gorql

import (
	"net/url"
	"strings"
	"testing"
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
