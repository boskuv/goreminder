# Use the official Golang image as the build stage
FROM golang:1.23 AS builder

# Add Golang specific environment variables for compiling
ENV GOOS=linux GOARCH=amd64

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy the go mod and sum files
COPY go.mod go.sum ./

# Copy the go mod and sum files
COPY ./cmd/core/config.yaml.example ./config.yaml

# Download all dependencies. Dependencies will be cached if the go.mod and 
# go.sum files are not changed
RUN go mod download && go mod verify

# Copy the source code into the container
COPY . .

# Build the Go app
RUN CGO_ENABLED=0 go build -o goreminder ./cmd/core/main.go

# Start a new stage from scratch
FROM alpine:latest

# Install file utility to check binary format
RUN apk add --no-cache file

WORKDIR /app

# We don't want to run our container as the root user for security reasons
# Therefore, we define a new non-root user and UID for it
# We disable login via password and omit creating a home directory
# to protect us against malicious SSH login attempts
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
# Binary permissions are given to our custom user to make the app runnable
COPY --from=builder --chown=${USER}:${USER} /app/goreminder .
COPY --from=builder --chown=${USER}:${USER} /app/config.yaml .

RUN chmod +x ./goreminder

# Switch to our new user
USER ${USER}:${USER}

# Command to run the executable
CMD ["./goreminder"]