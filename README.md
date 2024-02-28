# GORQL

## Overview
A lightweight Go package for constructing Resource Query Language (RQL) queries, commonly used for filtering, sorting, and querying data.
It provides a simple and intuitive API for building RQL queries dynamically.

## Installation
    go get -u github.com/douglaslim/gorql

## Getting Started

In order to start using rql, you can optionally configure the parser. Let's go over a basic example of how to do this.

To build a parser, use `gorql.NewParser(*gorql.Config)`.
```go
type User struct {
	ID          uint      `rql:"filter,sort"`
	Admin       bool      `rql:"filter"`
	Name        string    `rql:"filter"`
	AddressName string    `rql:"filter"`
	CreatedAt   time.Time `rql:"filter,sort"`
}

var Parser = gorql.NewParser(&gorql.Config{
	// User if the resource we want to query.
	Model: User{},
	// Use your own custom logger. This logger is used only in the building stage.
	Log: logrus.Printf,
	// Default limit returned by the `Parse` function if no limit provided by the user.
	DefaultLimit: 100,
	// Accept only requests that pass limit value that is greater than or equal to 200.
	LimitMaxValue: 200,
})
```

gorql uses reflection in the build process to detect the type of each field, and create a set of validation rules for each one. If one of the validation rules fails or rql encounters an unknown field, it returns an informative error to the user.
Don't worry about the usage of reflection, it happens only once when you build the parser.
Let's go over the validation rules:
1. `int` (8,16,32,64) - Round number
2. `uint` (8,16,32,64) - Round number and greater than or equal to 0
3. `float` (32,64): - Number
4. `bool` - Boolean
5. `string` - String
6. `time.Time`, and other types that convertible to `time.Time` - The default layout is time.RFC3339 format (JS format), and parsable to `time.Time`.
   It's possible to override the `time.Time` layout format with custom one. You can either use one of the standard layouts in the `time` package, or use a custom one. For example:
   ```go
   type User struct {
		T1 time.Time `rql:"filter"`                         // time.RFC3339
		T2 time.Time `rql:"filter,layout=UnixDate"`         // time.UnixDate
		T3 time.Time `rql:"filter,layout=2006-01-02 15:04"` // 2006-01-02 15:04 (custom)
   }
   ```

Note that all rules are applied to pointers as well. It means, if you have a field `Name *string` in your struct, we still use the string validation rule for it.

## RQL Rules

Here is a definition of the common operators:

* sort(&lt;+|->&lt;property) - Sorts by the given property in order specified by the prefix (+ for ascending, - for descending)
* select(&lt;property>,&lt;property>,...) - Trims each object down to the set of properties defined in the arguments
* limit(count,offset) - Returns the given range of objects from the result set
* and(&lt;query>,&lt;query>,...) - Applies all the given queries
* or(&lt;query>,&lt;query>,...) - The union of the given queries
* in(&lt;property>,&lt;array-of-values>) - Filters for objects where the specified property's value is in the provided array
* like(&lt;property>,&lt;value>) - Filters records where property contains value as a substring. This applies to strings or arrays of strings.
* match(&lt;property>,&lt;value | expression>) - Filters for objects where the specified property's value is an array and the array contains any value that equals the provided value or satisfies the provided expression.
* eq(&lt;property>,&lt;value>) - Filters for objects where the specified property's value is equal to the provided value
* lt(&lt;property>,&lt;value>) - Filters for objects where the specified property's value is less than the provided value
* le(&lt;property>,&lt;value>) - Filters for objects where the specified property's value is less than or equal to the provided value
* gt(&lt;property>,&lt;value>) - Filters for objects where the specified property's value is greater than the provided value
* ge(&lt;property>,&lt;value>) - Filters for objects where the specified property's value is greater than or equal to the provided value
* ne(&lt;property>,&lt;value>) - Filters for objects where the specified property's value is not equal to the provided value

## Usage
Here's a quick example of how to use gorql to construct an RQL query and translate to mongo query:
```
package main

import (
	"fmt"
	"gorql"
	"gorql/pkg/driver/mongo"
	"strings"
)

func main() {
	p, err := gorql.NewParser(nil)
	if err != nil {
		panic(fmt.Sprintf("New parser error :%s", err))
	}
	query := `and(eq(foo,3),lt(price,10))&sort(+price)`
	rqlNode, err := p.Parse(strings.NewReader(query))
	if err != nil {
		panic(err)
	}
	mongoTranslator := mongo.NewMongoTranslator(rqlNode)
	w, err := mongoTranslator.Where()
	if err != nil {
		panic(err)
	}
	fmt.Println(w) // {"$and": [{"$and": [{"foo": {"$eq": "3"}}, {"price": {"$lt": "10"}}]}]}
}
```

## Contributions

Contributions are welcome! If you encounter any bugs, issues, or have feature requests, please open an issue. Pull requests are also appreciated.

Before contributing, please review the contribution guidelines.

## License
This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.