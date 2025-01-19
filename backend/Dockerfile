FROM golang:1.23-alpine AS builder

WORKDIR /app

# Install necessary build tools
RUN apk add --no-cache git make

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code and Makefile
COPY . .

# Build the application using make
RUN make build

# Final stage
FROM alpine:3.19

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/smart-insights .

# Expose the application port
EXPOSE 8080

# Run the application
ENTRYPOINT ["/app/smart-insights"]
CMD ["api"]