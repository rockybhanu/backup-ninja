# Stage 1: Build the Go binary
FROM golang:1.22-alpine as builder

# Set environment variables for Go
ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64

# Create and set the working directory
WORKDIR /app

# Copy the Go modules manifests
COPY go.mod ./

# Download the Go modules
RUN go mod download

# Copy the source code
COPY . .

# Build the Go binary
RUN go build -o db_tool .

# Stage 2: Create the final minimal image
FROM alpine:latest

# Install the MariaDB client and Restic
RUN apk --no-cache add mariadb-client restic

# Print versions of installed tools
RUN mariadb --version && restic version

# Copy the Go binary from the builder stage
COPY --from=builder /app/db_tool /usr/local/bin/db_tool

# Create a directory for backups
RUN mkdir -p /backup

# Ensure the binary has execution permissions
RUN chmod +x /usr/local/bin/db_tool

# Set the entrypoint to the Go binary
ENTRYPOINT ["/usr/local/bin/db_tool"]
