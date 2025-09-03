Go Todo API: A Production-Grade RESTful ServiceThis repository contains the source code for a high-performance, containerized REST API for a Todo application. This project is a demonstration of modern backend development practices, including a multi-service architecture, secure authentication, a full test suite, and cloud deployment.🚀 Live DemoThe API is deployed and live on Render. You can interact with it using curl or Postman.Live URL: https://todo-api-n1s3.onrender.com/Note: The service is on a free tier and may "sleep" during periods of inactivity. The first request might take 15-30 seconds to respond as the container starts up (this is known as a "cold start").🏛️ ArchitectureThis project runs as a multi-container application orchestrated locally by Docker Compose and deployed as a multi-service blueprint on Render. It follows a clean separation of concerns between the API, data, and caching layers.graph TD
    subgraph Client
        User[👨‍💻 Postman / curl]
    end

    subgraph "Cloud Environment (Render)"
        subgraph "Go API Web Service"
            API[Go HTTP Server w/ Middleware]
        end

        subgraph "PostgreSQL Managed Database"
            DB[(fa:fa-database Todos & Users DB)]
        end

        subgraph "Redis Managed Cache"
            Cache[(fa:fa-bolt Redis Cache)]
        end
    end

    User -- "HTTPS Request" --> API
    API -- "1. Cache Miss" --> DB
    API -- "2. Populate Cache" --> Cache
    API -- "Cache Hit" --> Cache
    API -- "Cache Invalidation (on Write)" --> Cache
    API -- "All DB Writes" --> DB

🛠️ Tech StackCategoryTechnologyPurposeLanguageGo (Golang)For building a high-performance, concurrent backend.DatabasePostgreSQLThe primary, persistent data store for users and todos.CacheRedisAn in-memory cache to reduce database load and improve read latency.ContainerizationDocker & Docker ComposeFor creating a consistent, portable development and production environment.APIRESTful JSON APIStandard-based communication for any type of client.AuthenticationJWT (JSON Web Tokens)Secure, stateless authentication for protecting API endpoints.Password SecuritybcryptThe industry-standard algorithm for securely hashing passwords.TestingGo testing packageFor comprehensive unit and integration tests.DeploymentRenderA modern cloud platform for deploying containerized services.✨ Key FeaturesFull CRUD Functionality for Todos (Create, Read, Update, Delete).Secure User Authentication with /register and /login endpoints.JWT-Protected Routes for all Todo operations using standard Go middleware.High-Performance Caching using the "cache-aside" pattern with Redis.Cache Invalidation on UPDATE and DELETE to ensure data consistency.Professional Project Structure with a clear separation between API and data storage layers.Persistent Data using Docker Volumes for local development and managed databases in production.Graceful Shutdown implemented to ensure data integrity and prevent resource leaks.Comprehensive Integration Tests that verify the API, database, and cache work together correctly.🚀 Getting Started (Running Locally)PrerequisitesDocker & Docker Composemake (optional, for shortcuts)InstructionsClone the repository:git clone [https://github.com/your-username/your-repo-name.git](https://github.com/your-username/your-repo-name.git)
cd your-repo-name
Create your environment file:Copy the example file and fill in your desired local database credentials.cp .env.example .env
Run the application stack:This single command will build the Go binary, start the Postgres and Redis containers, and run your API.docker-compose up --build
The API will be available at http://localhost:8080.🧪 Running TestsThe project includes a full integration test suite that runs against a separate, temporary test database.Start the test environment:docker-compose -f docker-compose.test.yml up -d
Run the tests:make test
Tear down the test environment:docker-compose -f docker-compose.test.yml down
📖 API EndpointsBase URL: https://todo-api-n1s3.onrender.comAuthenticationPOST /registerCreates a new user.curl -X POST -H "Content-Type: application/json" \
  -d '{"username": "testuser", "password": "password123"}' \
  [https://todo-api-n1s3.onrender.com/register](https://todo-api-n1s3.onrender.com/register)
POST /loginAuthenticates a user and returns a JWT.curl -X POST -H "Content-Type: application/json" \
  -d '{"username": "testuser", "password": "password123"}' \
  [https://todo-api-n1s3.onrender.com/login](https://todo-api-n1s3.onrender.com/login)
Todos (Protected)Requires Authorization: Bearer <your_jwt_token> header.GET /todos/Fetches all todos for the authenticated user.TOKEN="your_jwt_token_here"
curl -H "Authorization: Bearer $TOKEN" [https://todo-api-n1s3.onrender.com/todos/](https://todo-api-n1s3.onrender.com/todos/)
POST /todos/Creates a new todo.TOKEN="your_jwt_token_here"
curl -X POST -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"task": "Write professional README", "completed": true}' \
  [https://todo-api-n1s3.onrender.com/todos/](https://todo-api-n1s3.onrender.com/todos/)
