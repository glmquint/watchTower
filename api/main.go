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
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	authv1 "watchtower/proto/gen/auth/v1"
	incidentv1 "watchtower/proto/gen/incident/v1"

	"watchtower/api/graph"
	"watchtower/api/graph/generated"
)

func main() {
	redisAddr := getenv("REDIS_ADDR", "localhost:6379")
	rdb := redis.NewClient(&redis.Options{Addr: redisAddr})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("failed to connect to redis: %v", err)
	}

	// gRPC clients
	authAddr := getenv("AUTH_ADDR", "localhost:50051")
	incAddr := getenv("INCIDENT_ADDR", "localhost:50052")

	authConn, err := grpc.Dial(authAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("failed to dial auth service: %v", err)
	}
	defer authConn.Close()
	incConn, err := grpc.Dial(incAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("failed to dial incident service: %v", err)
	}
	defer incConn.Close()

	resolvers := &graph.Resolver{
		Redis:          rdb,
		JWTSecret:      getenv("JWT_SECRET", "dev-secret-change"),
		AuthClient:     authv1.NewAuthServiceClient(authConn),
		IncidentClient: incidentv1.NewIncidentServiceClient(incConn),
	}
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
		// Delegate creation to incident-service (it will publish to Redis)
		resp, err := resolvers.IncidentClient.CreateIncident(r.Context(), &incidentv1.CreateIncidentRequest{Title: p.Title})
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"id": resp.GetId(), "title": resp.GetTitle()})
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
