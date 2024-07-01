package graph

//go:generate go run github.com/99designs/gqlgen generate

import "ozon-graphql/database"

type Resolver struct {
	DB database.DB
}
