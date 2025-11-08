package graph

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	authv1 "watchtower/proto/gen/auth/v1"
	incidentv1 "watchtower/proto/gen/incident/v1"
)

// Resolver serves as dependency injection for your app, add services here.
type Resolver struct {
	DB        *pgxpool.Pool
	Redis     *redis.Client
	JWTSecret string
	// gRPC clients
	AuthClient     authv1.AuthServiceClient
	IncidentClient incidentv1.IncidentServiceClient
}
