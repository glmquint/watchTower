FROM golang:1.22 AS builder
WORKDIR /src/services/incident-service

# Copy module files for the service and the local proto module referenced by replace
COPY services/incident-service/go.mod services/incident-service/go.sum ./
COPY proto /src/proto
RUN go mod download

# Copy service sources and build
COPY services/incident-service ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /app/incident-service ./

FROM alpine:3.18
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/incident-service /app/incident-service
ENV PATH="/app:${PATH}"
EXPOSE 50052
ENTRYPOINT ["/app/incident-service"]
