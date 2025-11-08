package graph

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// Resolver serves as dependency injection for your app, add services here.
type Resolver struct {
	DB        *pgxpool.Pool
	Redis     *redis.Client
	JWTSecret string
}
