package graph

import "github.com/jackc/pgx/v5/pgxpool"

// Resolver serves as dependency injection for your app, add services here.
type Resolver struct {
	DB *pgxpool.Pool
}
