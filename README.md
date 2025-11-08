# 🛡️ WatchTower: Real-Time Incident Response Platform

WatchTower is a mobile-first platform designed for Security Operations Center (SOC) teams to ingest, triage, and respond to security alerts in real-time. It showcases a modern, secure, and highly scalable cloud-native architecture using a full-stack, distributed system.

## ✨ Features

  * **Real-Time Alerting:** Instant push notifications and live dashboard updates for new incidents using GraphQL Subscriptions and Redis Pub/Sub.
  * **Secure Mobile Client (React Native):** Utilizes biometric authentication (FaceID/Fingerprint) and secure storage (Expo SecureStore) for sensitive data access.
  * **Incident Triage:** Mobile interface for analysts to **Acknowledge**, **Investigate**, and **Resolve** incidents.
  * **Microservices Architecture (Go):** Decoupled services for ingestion, incident management, and authentication, communicating via high-performance **gRPC**.
  * **Immutable Audit Log:** All actions (logins, acknowledgments, status changes) are recorded via a dedicated **Audit Service** for compliance and non-repudiation.
  * **Cloud-Native Deployment:** Fully orchestrated on **Google Kubernetes Engine (GKE)** with integrated secret management and network policies.

## 🏗️ Architecture and Technology Stack

This project is built on a distributed microservices model to demonstrate expertise across the entire stack, from frontend to cloud infrastructure.

| Component | Technology | Role |
| :--- | :--- | :--- |
| **Mobile Client** | **React Native (Expo), TypeScript** | Provides the secure, cross-platform interface for on-call analysts. |
| **API Gateway** | **GraphQL (Go/gqlgen)** | Single, authenticated entry point for the mobile client. Handles real-time subscriptions via WebSockets. |
| **Backend Logic** | **Go, gRPC** | High-performance microservices (`auth`, `ingestor`, `incident`, `audit`) responsible for business logic. |
| **Real-Time Layer** | **Redis (Pub/Sub, List)** | Used as a resilient **Job Queue** for alert ingestion and the **Pub/Sub** mechanism for pushing live updates. |
| **Database** | **Postgres** | Primary data store for all user data, incident state, and the immutable audit log. |
| **Containerization** | **Docker** | Used for packaging every microservice with multi-stage builds for minimal image sizes. |
| **Orchestration** | **Kubernetes (GKE)** | Manages the deployment, scaling, health, and networking of all microservices. |
| **Cloud Provider** | **Google Cloud Platform (GCP)** | Hosts the GKE cluster, Cloud SQL (Postgres), Memorystore (Redis), and managed networking (Ingress). |

## 🛠️ Local Development Setup

### Prerequisites

1.  **Go:** Version 1.21+
2.  **Node.js & npm:** For the React Native client.
3.  **Expo CLI:** `npm install -g expo-cli`
4.  **Docker & Docker Compose:** For running local infrastructure.
5.  **`protoc`:** Protocol Buffer Compiler and associated Go plugins.

### Step 1: Initialize Infrastructure

Start the local Postgres database and Redis instance using Docker Compose.

```bash
docker compose -f infra/docker-compose.yaml up -d
```

### Step 2: Build and Run Go Services

1.  **Generate Protobuf/gRPC Code:**

    ```bash
    protoc --go_out=. --go-grpc_out=. proto/*.proto
    ```

2.  **Run Services:** Build and run each service in separate terminals. Each service connects to the `db` and `redis` containers via their hostnames, as defined in `docker-compose.yaml`.

    ```bash
    # Example for Incident Service
    go run ./services/incident-service/cmd/main.go

    # Example for GraphQL API
    go run ./api/server.go
    ```

### Step 3: Run Mobile Application

1.  **Install dependencies:**
    ```bash
    cd app
    npm install
    ```
2.  **Start the Expo Development Server:**
    ```bash
    npx expo start
    ```
3.  Scan the QR code with your mobile device or simulator using the Expo Go app. **Note:** You will need to configure your local development environment to access the backend services over your network (e.g., using a tool like ngrok if testing on a real device).

## ☁️ Deployment and Security Deep Dive

The production environment is hosted on GCP, leveraging best practices for security and scaling.

### Kubernetes Networking and Security

  * **Zero Trust with Network Policies:** Kubernetes `NetworkPolicy` objects are configured to restrict traffic. For example, the `incident-service` can only receive traffic from the `graphql-api` pod (gRPC) and **cannot** be directly accessed from the public internet.
  * **GCP Secret Manager:** Sensitive credentials (database passwords, JWT signing keys) are stored securely in **GCP Secret Manager**.
  * **Workload Identity:** GKE is configured with Workload Identity, allowing Kubernetes Service Accounts to impersonate Google IAM accounts, granting pods permission to access Secret Manager **without** needing secrets in environment variables or configuration files.

### CI/CD Pipeline (Simulated)

A production pipeline (often built with **Google Cloud Build**) would follow this flow:

1.  **`git push`** to the main branch.
2.  **Cloud Build** triggers:
    a.  Runs Go unit tests and linting.
    b.  Builds the Docker image for each updated microservice (multi-stage build).
    c.  Pushes the container images to **Google Container Registry (GCR)**.
    d.  Updates the Kubernetes deployment manifests on **GKE** via `kubectl apply`.

## 💡 Future Enhancements

  * **External Integration:** Add connectors to ingest alerts from real external sources (e.g., GitHub Security Alerts, simulated firewall logs).
  * **Machine Learning (Go):** Implement a simple Go-based anomaly detection service that reads transactions from Redis and flags unusual patterns before they hit the incident database.
  * **Web Dashboard:** Create a basic React-based web interface for managers to view high-level dashboards and analytics.

## 🚀 Monolithic PoC (Phase 0)

This initial PoC delivers a single GraphQL endpoint (Go + gqlgen) backed by Postgres, consumed by the Expo mobile client via Apollo.

### Added in this Phase
* `infra/docker-compose.yaml` – Local Postgres (host port 5432) + Redis.
* `infra/db/init/001_init.sql` – Creates `users` & `incidents` and seeds demo rows.
* `api/` – Go monolith with `Query.incidents` resolver hitting Postgres directly.
* `app/` – Existing Expo project extended with Apollo Client and an Incidents tab.

### Run Stack
1. Infra:
  ```bash
  docker compose -f infra/docker-compose.yaml up -d postgres redis
  ```
2. API:
  ```bash
  cd api
  DB_HOST=localhost DB_PORT=5432 DB_USER=watchtower DB_PASSWORD=watchtower DB_NAME=watchtower go run .
  # Playground: http://localhost:8080/playground
  # Sample query:
  # {
  #   incidents { id title }
  # }
  ```
3. Mobile:
  ```bash
  cd app
  npm install   # first time
  EXPO_PUBLIC_API_URL=http://<LAN_IP>:8080/query npx expo start
  ```
  Replace `<LAN_IP>` with your machine's LAN IP (e.g. `192.168.1.42`) so a physical device can reach the API.

### Verify
Open the Incidents tab. You should see seeded incidents ("Server down", "High latency"). If not:
* Test query in playground.
* Confirm API env vars point to the running Postgres (port 5432).
* Ensure Expo uses the correct LAN IP (not localhost).

### Next Enhancements
* Add createIncident mutation.
* Introduce auth (JWT) & protected resolvers.
* Redis Pub/Sub + GraphQL subscriptions for real-time updates.
* Gradually split services (incident, auth) behind gRPC.
* Add integration tests & CI workflow.

This completes Phase 0: Postgres → Go GraphQL → Apollo → Mobile UI end-to-end.