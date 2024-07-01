package server

import (
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"os"
	"ozon-graphql/database"
	"ozon-graphql/database/memory"
	"ozon-graphql/database/postgres"
	"ozon-graphql/graph"
)

func Start() {
	storageType := os.Getenv("STORAGE_TYPE")
	var db database.DB

	if storageType == "postgres" {
		dsn := os.Getenv("DATABASE_URL")
		postgresDB, err := postgres.NewPostgresDB(dsn)
		if err != nil {
			log.Fatalf("failed to connect to database: %v", err)
		}
		db = postgresDB
	} else {
		db = memory.NewInMemoryDB()
	}

	srv := handler.NewDefaultServer(graph.NewExecutableSchema(graph.Config{Resolvers: &graph.Resolver{DB: db}}))
	srv.AddTransport(&transport.Websocket{
		Upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
	})

	http.Handle("/", playground.Handler("GraphQL playground", "/query"))
	http.Handle("/query", srv)

	log.Printf("connect to http://localhost:8080/ for GraphQL playground")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
