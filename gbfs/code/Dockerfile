# Step 1: Use Golang base image to build the Go app
FROM golang:1.20 AS builder

# Set the working directory
WORKDIR /app

# Copy the Go module files and download dependencies (caching layer)
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the application source code
COPY . .

# Build the Go app with CGO disabled for a static binary
RUN CGO_ENABLED=0 go build -o my-go-app .

# Step 2: Use a minimal Alpine image
FROM alpine

# Install CA certificates
RUN apk add --no-cache ca-certificates

# Set the working directory in the final image
WORKDIR /root/

# Copy the statically linked Go binary from the builder image
COPY --from=builder /app/my-go-app .

# Expose the port the app runs on
EXPOSE 8080

# Run the Go app
CMD ["./my-go-app"]
