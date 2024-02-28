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
	query := `and(eq(foo,3),lt(price,10))&sort(+price)&limit(10,20)`
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

	sort := mongoTranslator.Sort()
	if err != nil {
		panic(err)
	}
	fmt.Println(sort) // {"$sort": {"price": 1}}

	limit := mongoTranslator.Limit()
	if err != nil {
		panic(err)
	}
	fmt.Println(limit) // {"$limit": 10}

	offset := mongoTranslator.Offset()
	if err != nil {
		panic(err)
	}
	fmt.Println(offset) // {"$skip": 20}
}
