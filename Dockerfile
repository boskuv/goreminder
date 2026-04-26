# Use Alpine-based Go image for smaller size
FROM golang:1.25-alpine AS builder

# Install Git and build dependencies
RUN apk add --no-cache \
    git \
    ca-certificates \
    tzdata \
    gcc \
    musl-dev

# Add Golang specific environment variables for compiling
ENV GOOS=linux \
    GOARCH=amd64 \
    CGO_ENABLED=0 \
    GO111MODULE=on

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy the go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies
RUN go mod download && go mod verify

# Copy the config example
COPY ./cmd/core/config.yaml.example ./config.yaml

# Copy the source code into the container
COPY . .

# Build the Go app
RUN go build -ldflags="-w -s" -trimpath -o goreminder ./cmd/core/main.go

# Start a new stage from scratch
FROM alpine:latest

# Install ca-certificates for HTTPS and tzdata for timezone support
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    && update-ca-certificates

WORKDIR /app

# Create non-root user
ENV USER=apiuser UID=10001
RUN adduser \
    --disabled-password \
    --gecos "" \
    --home "/nonexistent" \
    --shell "/sbin/nologin" \
    --no-create-home \
    --uid "${UID}" \
    "${USER}"

# Copy the compiled binary from 'builder' stage
COPY --from=builder --chown=${USER}:${USER} /app/goreminder .
COPY --from=builder --chown=${USER}:${USER} /app/config.yaml .
COPY --from=builder --chown=${USER}:${USER} /app/migrations ./migrations

# Verify binary
RUN file ./goreminder && ./goreminder --version || true

# Set permissions
RUN chmod 550 ./goreminder

# Switch to non-root user
USER ${USER}:${USER}

# Command to run the executable
CMD ["./goreminder"]
