# GORQL

## Overview
A lightweight Go package for constructing Resource Query Language (RQL) queries, commonly used for filtering, sorting, and querying data.
It provides a simple and intuitive API for building RQL queries dynamically.

## Installation
    go get -u github.com/douglaslim/gorql

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