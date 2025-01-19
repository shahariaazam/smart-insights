# Frontend build stage
FROM node:20-slim AS frontend-builder
WORKDIR /app

# Install dependencies
COPY frontend/smart-insights/package*.json ./
RUN npm ci

# Copy frontend source code
COPY frontend/smart-insights/ .

RUN npm install -g vite

# Build the frontend application
RUN npm run build

# Backend build stage
FROM golang:1.23-alpine AS backend-builder
WORKDIR /app

# Install necessary build tools
RUN apk add --no-cache git make

# Copy go mod files
COPY backend/go.mod backend/go.sum ./
RUN go mod download

# Copy backend source code and Makefile
COPY backend/ .

# Build the application using make
RUN make build

# Final stage
FROM alpine:3.19

WORKDIR /app

# Copy the backend binary
COPY --from=backend-builder /app/smart-insights .

# Create dist directory and copy frontend assets
RUN mkdir -p /app/app/dist
COPY --from=frontend-builder /app/dist /app/app/dist

# Set default environment variable
ENV STATIC_FILE_PATH=/app/app/dist

# Expose the application port
EXPOSE 8080

# Run the application
ENTRYPOINT ["/app/smart-insights"]
CMD ["api"]