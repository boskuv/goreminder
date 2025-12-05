# Use the official Golang image as the build stage
FROM golang:1.23-alpine AS builder

# Add Golang specific environment variables for compiling
ENV GOOS=linux \
    GOARCH=amd64 \
    CGO_ENABLED=0 \
    GOPROXY=direct \
    GOSUMDB=off

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy the go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and 
# go.sum files are not changed
RUN go mod download && go mod verify

# Copy the source code into the container
COPY . .

# Build the Go app
RUN go build -ldflags="-w -s" -trimpath -o goreminder ./cmd/core/main.go

# Start a new stage from scratch
FROM alpine:latest

# Security hardening
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    && update-ca-certificates \
    && rm -rf /var/cache/apk/*

WORKDIR /app

# Create non-root user with specific UID/GID
ARG USER=apiuser
ARG UID=10001
ARG GID=10001

RUN addgroup -g ${GID} ${USER} && \
    adduser \
    --disabled-password \
    --gecos "" \
    --home "/nonexistent" \
    --shell "/sbin/nologin" \
    --no-create-home \
    --uid "${UID}" \
    --ingroup "${USER}" \
    "${USER}"

# Copy the compiled binary from 'builder' stage
COPY --from=builder --chown=${UID}:${GID} /app/goreminder .
COPY --from=builder --chown=${UID}:${GID} /app/config.yaml .

# Verify binary security
RUN ["/app/goreminder", "--version"] || true

# Set secure permissions
RUN chmod 550 /app/goreminder && \
    chmod 440 /app/config.yaml

# Create necessary directories with correct permissions
RUN mkdir -p /app/logs /app/data && \
    chown -R ${UID}:${GID} /app/logs /app/data && \
    chmod 750 /app/logs /app/data

# Switch to our new user
USER ${UID}:${GID}

# Security-related environment variables
ENV PATH="/app:${PATH}" \
    GIN_MODE=release

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ["/app/goreminder", "health"] || exit 1

# Command to run the executable
ENTRYPOINT ["/app/goreminder"]
CMD ["serve"]