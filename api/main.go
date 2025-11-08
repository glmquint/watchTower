package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/jackc/pgx/v5/pgxpool"

	"watchtower/api/graph"
	"watchtower/api/graph/generated"
)

func main() {
	cfg := dbConfigFromEnv()
	pool, err := pgxpool.New(context.Background(), cfg)
	if err != nil {
		log.Fatalf("failed to connect to postgres: %v", err)
	}
	defer pool.Close()

	resolvers := &graph.Resolver{DB: pool}
	srv := handler.NewDefaultServer(generated.NewExecutableSchema(generated.Config{Resolvers: resolvers}))

	http.Handle("/query", corsMiddleware(srv))
	http.Handle("/playground", playground.Handler("GraphQL playground", "/query"))

	addr := ":8080"

	// Print local network interfaces to help debugging device connectivity
	ifaces, ierr := net.Interfaces()
	if ierr == nil {
		log.Println("Network interfaces:")
		for _, iface := range ifaces {
			addrs, aerr := iface.Addrs()
			if aerr != nil {
				continue
			}
			for _, a := range addrs {
				log.Printf(" - %s: %s", iface.Name, a.String())
			}
		}
	} else {
		log.Printf("could not list interfaces: %v", ierr)
	}

	log.Printf("GraphQL server started and listening on %s (playground at http://<host>:8080/playground)", addr)
	if err := http.ListenAndServe("0.0.0.0:8080", nil); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}

func dbConfigFromEnv() string {
	host := getenv("DB_HOST", "localhost")
	port := getenv("DB_PORT", "5432")
	user := getenv("DB_USER", "watchtower")
	pass := getenv("DB_PASSWORD", "watchtower")
	name := getenv("DB_NAME", "watchtower")
	return "postgres://" + user + ":" + pass + "@" + host + ":" + port + "/" + name + "?sslmode=disable"
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// Simple CORS for dev & Expo on LAN
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
