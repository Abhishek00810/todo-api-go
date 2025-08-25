# Stage 1: The "builder" stage, where we compile our app
FROM golang:1.22-alpine AS builder

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy go mod and sum files first for better caching
COPY go.mod go.sum ./

# Download all dependencies
RUN go mod download

# Copy the source code into the container
COPY . .

# Build the Go app as a static binary
RUN CGO_ENABLED=0 GOOS=linux go build -o todo-api .

# ---

# Stage 2: The "final" stage, where we create the lightweight image
FROM alpine:latest

# Copy only the compiled binary from the "builder" stage
COPY --from=builder /app/todo-api .

# Document the port that the container listens on
EXPOSE 8080

# This is the command that will run when the container starts
CMD ["./todo-api"]