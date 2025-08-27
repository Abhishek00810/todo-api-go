# Stage 1: The "builder" stage
# We still use the golang-alpine image
FROM golang:1.25-alpine AS builder

# Set the working directory
WORKDIR /app

# --- NEW STEP ---
# Install the C build tools (gcc) needed for the go-sqlite3 driver.
# 'apk' is Alpine's package manager.
RUN apk add --no-cache gcc musl-dev

# Copy and download dependencies (same as before)
COPY go.mod go.sum ./
RUN go mod download

# Copy source code (same as before)
COPY . .

# --- MODIFIED STEP ---
# Build the Go app, but this time KEEP CGO ENABLED.
# We remove the CGO_ENABLED=0 part.
RUN GOOS=linux go build -o todo-api .

# ---

# Stage 2: The "final" stage (this part remains almost the same)
FROM alpine:latest

# Copy only the compiled binary from the "builder" stage.
# The binary is still self-contained, even with CGo.
COPY --from=builder /app/todo-api .

# Expose the port
EXPOSE 8080

# Set the command to run
CMD ["./todo-api"]