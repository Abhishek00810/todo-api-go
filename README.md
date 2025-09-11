# 📝 Go Todo API
**A Production-Grade RESTful Service with PostgreSQL, Redis, JWT Authentication, and Kubernetes Deployment**

![Go](https://img.shields.io/badge/Go-00ADD8?style=flat&logo=go&logoColor=white)
![Postgres](https://img.shields.io/badge/PostgreSQL-336791?style=flat&logo=postgresql&logoColor=white)
![Redis](https://img.shields.io/badge/Redis-DC382D?style=flat&logo=redis&logoColor=white)
![Docker](https://img.shields.io/badge/Docker-2496ED?style=flat&logo=docker&logoColor=white)
![Kubernetes](https://img.shields.io/badge/Kubernetes-326CE5?style=flat&logo=kubernetes&logoColor=white)
![Helm](https://img.shields.io/badge/Helm-0F1689?style=flat&logo=helm&logoColor=white)

## 🚀 Live Demo
The API is deployed on Render with Kubernetes-ready configuration.

🌍 **Base URL**: https://todo-api-n1s3.onrender.com

⚠️ **Note**: This is running on Render's free tier, so the service may sleep during inactivity.
The first request after inactivity may take 15–30 seconds to respond (cold start).

## 🏛️ Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Kubernetes Cluster                       │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐     │
│  │   Ingress   │    │ LoadBalancer│    │   Service   │     │
│  │             │────│             │────│  Discovery  │     │
│  └─────────────┘    └─────────────┘    └─────────────┘     │
│         │                                       │          │
│         ▼                                       ▼          │
│  ┌─────────────────────────────────────────────────────┐   │
│  │              Application Layer                      │   │
│  │  ┌─────────┐  ┌─────────┐  ┌─────────┐           │   │
│  │  │Go API   │  │Go API   │  │Go API   │           │   │
│  │  │Pod 1    │  │Pod 2    │  │Pod 3    │           │   │
│  │  └─────────┘  └─────────┘  └─────────┘           │   │
│  └─────────────────────────────────────────────────────┘   │
│         │                               │                  │
│         ▼                               ▼                  │
│  ┌─────────────┐                ┌─────────────┐           │
│  │ PostgreSQL  │                │   Redis     │           │
│  │   Master    │                │   Cache     │           │
│  │    Pod      │                │    Pod      │           │
│  └─────────────┘                └─────────────┘           │
│         │                               │                  │
│         ▼                               ▼                  │
│  ┌─────────────┐                ┌─────────────┐           │
│  │Persistent   │                │Persistent   │           │
│  │Volume (DB)  │                │Volume (Cache│           │
│  └─────────────┘                └─────────────┘           │
└─────────────────────────────────────────────────────────────┘
```

## ✨ Features

### 🔐 Authentication & Security
- ✅ User Registration with `/register` & `/login`
- ✅ JWT-Protected Routes for all Todo operations
- ✅ Password hashing with bcrypt
- ✅ Kubernetes Secrets for sensitive data

### 📊 Database & Caching
- ✅ CRUD for Todos (Create, Read, Update, Delete)
- ✅ Cache-aside Pattern using Redis for fast reads
- ✅ Cache Invalidation on writes to ensure consistency
- ✅ Persistent Volumes for data durability

### 🚀 Production Ready
- ✅ Graceful Shutdown to prevent data loss & leaks
- ✅ Health Checks for Kubernetes probes
- ✅ Horizontal Pod Autoscaling (HPA)
- ✅ Resource limits and requests
- ✅ ConfigMaps for environment configuration

### 🔧 DevOps & Deployment
- ✅ Dockerized Setup for local and prod parity
- ✅ Integration Tests (API + DB + Cache end-to-end)
- ✅ Kubernetes manifests for production deployment
- ✅ Helm charts for easy deployment
- ✅ CI/CD with GitHub Actions

## ⚡ Getting Started

### 🔹 Prerequisites

- **Docker** & **Docker Compose**
- **Kubernetes** cluster (minikube, kind, or cloud provider)
- **kubectl** configured
- **Helm** (optional, for easier deployment)
- **make** (optional for shortcuts)

### 🔹 Local Development (Docker Compose)

1. **Clone the repository**
```bash
git clone https://github.com/Abhishek00810/go-todo-api.git
cd go-todo-api
```

2. **Copy environment file**
```bash
cp .env.example .env
```

3. **Start services**
```bash
docker-compose up --build
```

API will be live at: 👉 **http://localhost:8080**

### 🔹 Kubernetes Deployment

#### Option 1: Using Kubectl (Manual)

1. **Apply PostgreSQL with Persistent Storage**
```bash
kubectl apply -f k8s/postgres.yaml
```

2. **Apply Redis**
```bash
kubectl apply -f k8s/redis.yaml
```

3. **Apply Application**
```bash
kubectl apply -f k8s/app.yaml
```

4. **Apply Ingress/LoadBalancer**
```bash
kubectl apply -f k8s/ingress.yaml
```

#### Option 2: Using Helm (Recommended)

1. **Install the Helm chart**
```bash
helm install todo-api ./chart/todo-api
```

2. **Upgrade deployment**
```bash
helm upgrade todo-api ./chart/todo-api
```

3. **Check status**
```bash
kubectl get pods
kubectl get services
```

## 🧪 Testing

### Local Testing
```bash
# Start test environment
docker-compose -f docker-compose.test.yml up -d

# Run tests
make test

# Cleanup
docker-compose -f docker-compose.test.yml down
```

### Kubernetes Testing
```bash
# Port forward to access API
kubectl port-forward service/todo-api-service 8080:80

# Test endpoints
curl http://localhost:8080/health
```

## 📖 API Endpoints

**Base URL**: https://todo-api-n1s3.onrender.com

### 🔹 Health Check
```bash
curl -X GET https://todo-api-n1s3.onrender.com/health
```

### 🔹 Authentication

**Register**
```bash
curl -X POST -H "Content-Type: application/json" \
  -d '{"username": "testuser", "password": "password123"}' \
  https://todo-api-n1s3.onrender.com/register
```

**Login**
```bash
curl -X POST -H "Content-Type: application/json" \
  -d '{"username": "testuser", "password": "password123"}' \
  https://todo-api-n1s3.onrender.com/login
```

Returns: `{ "token": "<JWT_TOKEN>" }`

### 🔹 Todos (Protected Routes)

**Requires header**: `Authorization: Bearer <JWT_TOKEN>`

**Get All Todos**
```bash
curl -H "Authorization: Bearer $TOKEN" \
  https://todo-api-n1s3.onrender.com/todos/
```

**Create Todo**
```bash
curl -X POST -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"task": "Deploy to Kubernetes", "completed": false}' \
  https://todo-api-n1s3.onrender.com/todos/
```

**Update Todo**
```bash
curl -X PUT -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"task": "Updated Task", "completed": true}' \
  https://todo-api-n1s3.onrender.com/todos/1
```

**Delete Todo**
```bash
curl -X DELETE -H "Authorization: Bearer $TOKEN" \
  https://todo-api-n1s3.onrender.com/todos/1
```

## 📦 Project Structure

```
go-todo-api/
├── .github/
│   └── workflows/
│       └── go.yml              # CI/CD pipeline
├── api/                        # HTTP handlers & middleware
├── chart/
│   └── todo-api/              # Helm chart
│       ├── Chart.yaml
│       ├── values.yaml
│       └── templates/
├── k8s/                       # Kubernetes manifests
│   ├── postgres.yaml         # PostgreSQL StatefulSet + PVC
│   ├── redis.yaml            # Redis Deployment + Service
│   ├── app.yaml              # Application Deployment + Service
│   ├── configmap.yaml        # Configuration
│   ├── secrets.yaml          # Sensitive data
│   ├── ingress.yaml          # Load balancer / Ingress
│   └── hpa.yaml              # Horizontal Pod Autoscaler
├── store/                     # Database layer
├── tmp/                       # Temporary files
├── .env.example              # Environment template
├── .gitignore
├── Dbmain.go                 # Database initialization
├── Dockerfile                # Production container
├── Dockerfile.test           # Test container
├── Makefile                  # Build automation
├── README.md
├── docker-compose.yml        # Local development
├── docker-compose.test.yml   # Test environment
├── go.mod                    # Go modules
├── go.sum
├── main_test.go             # Integration tests
├── render.yaml              # Render.com deployment
└── todos.db                 # SQLite (local development)
```

## 🛠️ Development Commands

```bash
# Build the application
make build

# Run tests
make test

# Build Docker image
make docker-build

# Deploy to Kubernetes
make k8s-deploy

# Clean up Kubernetes resources
make k8s-clean

# View logs
kubectl logs -f deployment/todo-api

# Scale application
kubectl scale deployment todo-api --replicas=5

# Check resource usage
kubectl top pods
```

## 🔧 Configuration

### Environment Variables
```bash
# Database
DB_HOST=postgres-service
DB_PORT=5432
DB_USER=myuser
DB_PASSWORD=mysecretpassword
DB_NAME=todos

# Redis Cache  
REDIS_HOST=redis-service
REDIS_PORT=6379

# JWT
JWT_SECRET=your-jwt-secret-key

# Application
PORT=8080
```

### Kubernetes Resources
- **CPU Request**: 100m, **Limit**: 500m
- **Memory Request**: 128Mi, **Limit**: 512Mi
- **Replicas**: 3 (auto-scaling 2-10)
- **PostgreSQL Storage**: 2Gi PVC
- **Redis Storage**: 1Gi PVC

## 🚀 Production Deployment Checklist

- [ ] **Security**: Update JWT secrets and database passwords
- [ ] **Resources**: Configure appropriate CPU/Memory limits
- [ ] **Storage**: Set up persistent volumes with proper storage classes
- [ ] **Monitoring**: Add health checks and metrics
- [ ] **Scaling**: Configure HPA based on CPU/Memory usage
- [ ] **Backup**: Set up database backup strategy
- [ ] **SSL**: Configure TLS termination at ingress
- [ ] **Logging**: Set up centralized logging (ELK/Fluentd)

## 🤝 Contributing

1. **Fork the repository** 🍴
2. **Create a feature branch** (`git checkout -b feature/amazing-feature`)
3. **Commit your changes** (`git commit -m 'Add amazing feature'`)
4. **Push to the branch** (`git push origin feature/amazing-feature`)
5. **Open a Pull Request** 🚀

## 📊 Monitoring & Observability

### Health Checks
- **Liveness Probe**: `/health`
- **Readiness Probe**: `/ready`
- **Startup Probe**: `/health`

### Metrics (Future Enhancement)
- Request count and latency
- Database connection pool status
- Cache hit/miss rates
- Custom business metrics

## 🔒 Security

- JWT tokens for authentication
- Kubernetes Secrets for sensitive data
- Network policies for pod-to-pod communication
- RBAC for service accounts
- Container security contexts

## 📜 License

This project is licensed under the **MIT License**. Feel free to use, modify, and distribute with attribution.

---

## 🌟 Star this repository if you found it helpful!

**Happy Coding!** 🎉
