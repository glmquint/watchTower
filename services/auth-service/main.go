package main

import (
	"context"
	"log"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc"

	authv1 "watchtower/proto/gen/auth/v1"
)

type authServer struct {
	authv1.UnimplementedAuthServiceServer
	db        *pgxpool.Pool
	jwtSecret string
}

func (s *authServer) Register(ctx context.Context, req *authv1.RegisterRequest) (*authv1.AuthResponse, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(req.GetPassword()), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	var id int64
	if err := s.db.QueryRow(ctx, `INSERT INTO users (email, name, password_hash) VALUES ($1,$2,$3) RETURNING id`, req.GetEmail(), req.GetName(), string(hash)).Scan(&id); err != nil {
		return nil, err
	}
	token, err := s.signToken(strconv.FormatInt(id, 10))
	if err != nil {
		return nil, err
	}
	return &authv1.AuthResponse{JwtToken: token, User: &authv1.User{Id: strconv.FormatInt(id, 10), Email: req.GetEmail(), Name: req.GetName()}}, nil
}

func (s *authServer) Login(ctx context.Context, req *authv1.LoginRequest) (*authv1.AuthResponse, error) {
	var id int64
	var name string
	var hash string
	if err := s.db.QueryRow(ctx, `SELECT id, name, password_hash FROM users WHERE email=$1`, req.GetEmail()).Scan(&id, &name, &hash); err != nil {
		return nil, grpc.Errorf(16, "invalid credentials") // codes.Unauthenticated
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(req.GetPassword())); err != nil {
		return nil, grpc.Errorf(16, "invalid credentials")
	}
	token, err := s.signToken(strconv.FormatInt(id, 10))
	if err != nil {
		return nil, err
	}
	return &authv1.AuthResponse{JwtToken: token, User: &authv1.User{Id: strconv.FormatInt(id, 10), Email: req.GetEmail(), Name: name}}, nil
}

func (s *authServer) ValidateToken(ctx context.Context, req *authv1.ValidateTokenRequest) (*authv1.ValidateTokenResponse, error) {
	claims := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(req.GetJwtToken(), claims, func(token *jwt.Token) (interface{}, error) { return []byte(s.jwtSecret), nil })
	if err != nil || !token.Valid {
		return &authv1.ValidateTokenResponse{Valid: false}, nil
	}
	if sub, ok := claims["sub"].(string); ok {
		return &authv1.ValidateTokenResponse{Valid: true, UserId: sub}, nil
	}
	return &authv1.ValidateTokenResponse{Valid: false}, nil
}

func (s *authServer) signToken(userID string) (string, error) {
	claims := jwt.MapClaims{
		"sub": userID,
		"exp": time.Now().Add(24 * time.Hour).Unix(),
		"iat": time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecret))
}

func main() {
	// DB
	cfg := dbConfigFromEnv()
	pool, err := pgxpool.New(context.Background(), cfg)
	if err != nil {
		log.Fatalf("failed to connect to postgres: %v", err)
	}
	defer pool.Close()

	addr := ":50051"
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	authv1.RegisterAuthServiceServer(grpcServer, &authServer{db: pool, jwtSecret: getenv("JWT_SECRET", "dev-secret-change")})
	log.Printf("auth-service gRPC server listening on %s", addr)
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
