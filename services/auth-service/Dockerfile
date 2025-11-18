FROM golang:1.22 AS builder
WORKDIR /src/services/auth-service

# Copy module files for the service and the local proto module referenced by replace
COPY services/auth-service/go.mod services/auth-service/go.sum ./
COPY proto /src/proto
RUN go mod download

# Copy service sources and build
COPY services/auth-service ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /app/auth-service ./

FROM alpine:3.18
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/auth-service /app/auth-service
ENV PATH="/app:${PATH}"
EXPOSE 50051
ENTRYPOINT ["/app/auth-service"]
