package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
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
	var rows pgx.Rows
	var err error
	if req.GetStatus() != "" {
		rows, err = s.db.Query(ctx, `SELECT id, title, severity, status, details, assignee_id FROM incidents WHERE status=$1 ORDER BY id ASC LIMIT $2`, req.GetStatus(), limit)
	} else {
		rows, err = s.db.Query(ctx, `SELECT id, title, severity, status, details, assignee_id FROM incidents ORDER BY id ASC LIMIT $1`, limit)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	resp := &incidentv1.ListIncidentsResponse{}
	for rows.Next() {
		var id int64
		var title, severity, status, details, assignee sql.NullString
		if err := rows.Scan(&id, &title, &severity, &status, &details, &assignee); err != nil {
			return nil, err
		}
		inc := &incidentv1.Incident{Id: strconv.FormatInt(id, 10), Title: title.String, Severity: severity.String, Status: status.String, Details: details.String, AssigneeId: assignee.String}
		resp.Incidents = append(resp.Incidents, inc)
	}
	return resp, rows.Err()
}

func (s *incidentServer) CreateIncident(ctx context.Context, req *incidentv1.CreateIncidentRequest) (*incidentv1.Incident, error) {
	if req.GetTitle() == "" {
		return nil, grpc.Errorf(3, "title required")
	}
	var id int64
	if err := s.db.QueryRow(ctx, `INSERT INTO incidents (title, severity, details, status) VALUES ($1,$2,$3,$4) RETURNING id`, req.GetTitle(), req.GetSeverity(), req.GetDetails(), "open").Scan(&id); err != nil {
		return nil, err
	}
	inc := &incidentv1.Incident{Id: strconv.FormatInt(id, 10), Title: req.GetTitle(), Severity: req.GetSeverity(), Status: "open", Details: req.GetDetails()}
	// publish new incident
	msg := map[string]any{"id": id, "title": req.GetTitle(), "severity": req.GetSeverity(), "status": "open", "details": req.GetDetails()}
	b, _ := json.Marshal(msg)
	if err := s.redis.Publish(ctx, "incidents:new", string(b)).Err(); err != nil {
		log.Printf("publish error: %v", err)
	}
	return inc, nil
}

func (s *incidentServer) GetIncident(ctx context.Context, req *incidentv1.GetIncidentRequest) (*incidentv1.Incident, error) {
	var id int64
	var title, severity, status, details, assignee sql.NullString
	if err := s.db.QueryRow(ctx, `SELECT id, title, severity, status, details, assignee_id FROM incidents WHERE id=$1`, req.GetId()).Scan(&id, &title, &severity, &status, &details, &assignee); err != nil {
		return nil, err
	}
	inc := &incidentv1.Incident{Id: strconv.FormatInt(id, 10), Title: title.String, Severity: severity.String, Status: status.String, Details: details.String, AssigneeId: assignee.String}
	// load comments for this incident
	rows, err := s.db.Query(ctx, `SELECT id, text, author_id, created_at FROM comments WHERE incident_id=$1 ORDER BY id ASC`, req.GetId())
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var cid int64
			var text, author sql.NullString
			var createdAt time.Time
			if err := rows.Scan(&cid, &text, &author, &createdAt); err != nil {
				// skip problematic row
				continue
			}
			c := &incidentv1.Comment{Id: strconv.FormatInt(cid, 10), Text: text.String, AuthorId: author.String, CreatedAt: createdAt.UTC().Format(time.RFC3339)}
			inc.Comments = append(inc.Comments, c)
		}
	}
	return inc, nil
}

func (s *incidentServer) AcknowledgeIncident(ctx context.Context, req *incidentv1.AcknowledgeIncidentRequest) (*incidentv1.Incident, error) {
	// set status to acknowledged
	if _, err := s.db.Exec(ctx, `UPDATE incidents SET status=$1 WHERE id=$2`, "acknowledged", req.GetIncidentId()); err != nil {
		return nil, err
	}
	// fetch updated
	inc, err := s.GetIncident(ctx, &incidentv1.GetIncidentRequest{Id: req.GetIncidentId()})
	if err == nil {
		// publish update
		idNum, _ := strconv.ParseInt(inc.GetId(), 10, 64)
		msg := map[string]any{"id": idNum, "title": inc.GetTitle(), "severity": inc.GetSeverity(), "status": inc.GetStatus(), "details": inc.GetDetails()}
		b, _ := json.Marshal(msg)
		if err := s.redis.Publish(ctx, "incidents:updates", string(b)).Err(); err != nil {
			log.Printf("publish error: %v", err)
		}
	}
	return inc, err
}

func (s *incidentServer) AddComment(ctx context.Context, req *incidentv1.AddCommentRequest) (*incidentv1.Comment, error) {
	var id int64
	var createdAt time.Time
	if err := s.db.QueryRow(ctx, `INSERT INTO comments (incident_id, text, author_id) VALUES ($1,$2,$3) RETURNING id, created_at`, req.GetIncidentId(), req.GetText(), req.GetAuthorId()).Scan(&id, &createdAt); err != nil {
		// fallback: try to insert without returning created_at
		if err2 := s.db.QueryRow(ctx, `INSERT INTO comments (incident_id, text, author_id) VALUES ($1,$2,$3) RETURNING id`, req.GetIncidentId(), req.GetText(), req.GetAuthorId()).Scan(&id); err2 != nil {
			return nil, err
		}
		createdAt = time.Now().UTC()
	}
	// created_at retrieval
	createdAtStr := createdAt.UTC().Format(time.RFC3339)
	c := &incidentv1.Comment{Id: strconv.FormatInt(id, 10), Text: req.GetText(), AuthorId: req.GetAuthorId(), CreatedAt: createdAtStr}
	// publish update for subscribers
	idNum, _ := strconv.ParseInt(req.GetIncidentId(), 10, 64)
	msg := map[string]any{"id": idNum, "comment": map[string]any{"id": id, "text": req.GetText(), "author_id": req.GetAuthorId(), "created_at": createdAt}}
	b, _ := json.Marshal(msg)
	if err := s.redis.Publish(ctx, "incidents:updates", string(b)).Err(); err != nil {
		log.Printf("publish error: %v", err)
	}
	return c, nil
}

func main() {
	cfg := dbConfigFromEnv()
	pool, err := pgxpool.New(context.Background(), cfg)
	if err != nil {
		log.Fatalf("failed to connect to postgres: %v", err)
	}
	// ensure schema exists
	if err := ensureSchema(pool); err != nil {
		log.Fatalf("failed to ensure schema: %v", err)
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

func ensureSchema(pool *pgxpool.Pool) error {
	ctx := context.Background()
	// incidents table
	_, err := pool.Exec(ctx, `
	CREATE TABLE IF NOT EXISTS incidents (
		id SERIAL PRIMARY KEY,
		title TEXT NOT NULL,
		severity TEXT DEFAULT 'medium',
		status TEXT DEFAULT 'open',
		details TEXT,
		assignee_id TEXT
	)`)
	if err != nil {
		return err
	}
	// ensure columns exist (for migrations from older schema)
	_, err = pool.Exec(ctx, `ALTER TABLE incidents ADD COLUMN IF NOT EXISTS severity TEXT DEFAULT 'medium'`)
	if err != nil {
		return err
	}
	_, err = pool.Exec(ctx, `ALTER TABLE incidents ADD COLUMN IF NOT EXISTS status TEXT DEFAULT 'open'`)
	if err != nil {
		return err
	}
	_, err = pool.Exec(ctx, `ALTER TABLE incidents ADD COLUMN IF NOT EXISTS details TEXT`)
	if err != nil {
		return err
	}
	_, err = pool.Exec(ctx, `ALTER TABLE incidents ADD COLUMN IF NOT EXISTS assignee_id TEXT`)
	if err != nil {
		return err
	}
	// comments table
	_, err = pool.Exec(ctx, `
	CREATE TABLE IF NOT EXISTS comments (
		id SERIAL PRIMARY KEY,
		incident_id TEXT NOT NULL,
		text TEXT NOT NULL,
		author_id TEXT,
		created_at TIMESTAMPTZ DEFAULT now()
	)`)
	if err != nil {
		return err
	}
	return nil
}
