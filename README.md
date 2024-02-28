# GORQL

## Overview
A lightweight Go package for constructing Resource Query Language (RQL) queries, commonly used for filtering, sorting, and querying data.
It provides a simple and intuitive API for building RQL queries dynamically.

## Installation
    go get -u github.com/douglaslim/gorql

## RQL Rules

Here is a definition of the common operators:

* sort(&lt;+|->&lt;property) - Sorts by the given property in order specified by the prefix (+ for ascending, - for descending)
* select(&lt;property>,&lt;property>,...) - Trims each object down to the set of properties defined in the arguments
* match(&lt;property>,&lt;value | expression>) - Filters for objects where the specified property's value is an array and the array contains any value that equals the provided value or satisfies the provided expression.
* limit(count,offset) - Returns the given range of objects from the result set
* and(&lt;query>,&lt;query>,...) - Applies all the given queries
* or(&lt;query>,&lt;query>,...) - The union of the given queries
* in(&lt;property>,&lt;array-of-values>) - Filters for objects where the specified property's value is in the provided array
* contains(&lt;property>,&lt;value | expression>) - Filters for objects where the specified property's value is an array and the array contains any value that equals the provided value or satisfies the provided expression.
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