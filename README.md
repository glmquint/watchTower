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

## 💡 Future Enhancements