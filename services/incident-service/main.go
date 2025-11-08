package main

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"os"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"

	incidentv1 "watchtower/proto/gen/incident/v1"
)

type incidentServer struct {
	incidentv1.UnimplementedIncidentServiceServer
	db    *pgxpool.Pool
	redis *redis.Client
}

func (s *incidentServer) ListIncidents(ctx context.Context, req *incidentv1.ListIncidentsRequest) (*incidentv1.ListIncidentsResponse, error) {
	limit := req.GetLimit()
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := s.db.Query(ctx, `SELECT id, title FROM incidents ORDER BY id ASC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	resp := &incidentv1.ListIncidentsResponse{}
	for rows.Next() {
		var id int64
		var title string
		if err := rows.Scan(&id, &title); err != nil {
			return nil, err
		}
		resp.Incidents = append(resp.Incidents, &incidentv1.Incident{Id: strconv.FormatInt(id, 10), Title: title})
	}
	return resp, rows.Err()
}

func (s *incidentServer) CreateIncident(ctx context.Context, req *incidentv1.CreateIncidentRequest) (*incidentv1.Incident, error) {
	if req.GetTitle() == "" {
		return nil, grpc.Errorf(3, "title required")
	} // codes.InvalidArgument
	var id int64
	if err := s.db.QueryRow(ctx, `INSERT INTO incidents (title) VALUES ($1) RETURNING id`, req.GetTitle()).Scan(&id); err != nil {
		return nil, err
	}
	inc := &incidentv1.Incident{Id: strconv.FormatInt(id, 10), Title: req.GetTitle()}
	// publish
	msg := map[string]any{"id": id, "title": req.GetTitle()}
	b, _ := json.Marshal(msg)
	if err := s.redis.Publish(ctx, "incidents:new", string(b)).Err(); err != nil {
		log.Printf("publish error: %v", err)
	}
	return inc, nil
}

func main() {
	cfg := dbConfigFromEnv()
	pool, err := pgxpool.New(context.Background(), cfg)
	if err != nil {
		log.Fatalf("failed to connect to postgres: %v", err)
	}
	defer pool.Close()
	rdb := redis.NewClient(&redis.Options{Addr: getenv("REDIS_ADDR", "localhost:6379")})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("failed to connect to redis: %v", err)
	}
	addr := ":50052"
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	incidentv1.RegisterIncidentServiceServer(grpcServer, &incidentServer{db: pool, redis: rdb})
	log.Printf("incident-service gRPC server listening on %s", addr)
	if err := grpcServer.Serve(lis); err != nil {
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
