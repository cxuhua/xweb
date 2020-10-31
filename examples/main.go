package main

import (
	"github.com/graphql-go/handler"

	"github.com/graphql-go/graphql"

	"github.com/cxuhua/xweb"
)

func main() {
	schema, err := graphql.NewSchema(graphql.SchemaConfig{
		Query: graphql.NewObject(graphql.ObjectConfig{
			Name: "Query",
			Fields: graphql.Fields{
				"version": &graphql.Field{
					Type: graphql.String,
					Resolve: func(p graphql.ResolveParams) (interface{}, error) {
						return "1.0.1", nil
					},
				},
			},
		}),
	})
	if err != nil {
		panic(err)
	}
	xweb.GraphQL("/graphql", &handler.Config{
		Schema:   &schema,
		Pretty:   true,
		GraphiQL: true,
	})
	_ = xweb.Serve(":9200")
}
