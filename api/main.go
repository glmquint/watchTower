package main

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

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

	redisAddr := getenv("REDIS_ADDR", "localhost:6379")
	rdb := redis.NewClient(&redis.Options{Addr: redisAddr})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("failed to connect to redis: %v", err)
	}

	resolvers := &graph.Resolver{DB: pool, Redis: rdb, JWTSecret: getenv("JWT_SECRET", "dev-secret-change")}
	srv := handler.NewDefaultServer(generated.NewExecutableSchema(generated.Config{Resolvers: resolvers}))

	http.Handle("/query", corsMiddleware(authMiddleware(resolvers.JWTSecret, srv)))
	http.Handle("/playground", playground.Handler("GraphQL playground", "/query"))
	http.HandleFunc("/send-test-alert", func(w http.ResponseWriter, r *http.Request) {
		type payload struct {
			Title string `json:"title"`
		}
		var p payload
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil || p.Title == "" {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "provide {title}"})
			return
		}
		var id int64
		if err := pool.QueryRow(r.Context(), `INSERT INTO incidents (title) VALUES ($1) RETURNING id`, p.Title).Scan(&id); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		msg := map[string]any{"id": id, "title": p.Title}
		b, _ := json.Marshal(msg)
		if err := rdb.Publish(r.Context(), "incidents:new", string(b)).Err(); err != nil {
			log.Printf("publish error: %v", err)
		}
		_ = json.NewEncoder(w).Encode(msg)
	})

	addr := ":8080"
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

// auth middleware
func authMiddleware(secret string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "" {
			const prefix = "Bearer "
			if len(auth) > len(prefix) && auth[:len(prefix)] == prefix {
				tokenStr := auth[len(prefix):]
				claims := jwt.MapClaims{}
				token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) { return []byte(secret), nil })
				if err == nil && token.Valid {
					if sub, ok := claims["sub"].(string); ok {
						ctx := context.WithValue(r.Context(), "userID", sub)
						r = r.WithContext(ctx)
					}
				}
			}
		}
		next.ServeHTTP(w, r)
	})
}
