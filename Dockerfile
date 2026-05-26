# Build stage
FROM golang:1.22-alpine AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy dependency files first
COPY go.mod ./

# Download all dependencies.
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the Go app statically linked
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o main ./cmd/api

# Run stage
FROM alpine:3.19

# Add CA certificates for HTTPS calls if needed
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the pre-built binary from the builder stage
COPY --from=builder /app/main .

# Expose port 8080 to the outside world
EXPOSE 8080

# Command to run the executable
CMD ["./main"]
