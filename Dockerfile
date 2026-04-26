# Stage 1: Build the frontend
FROM node:20-slim AS frontend-builder
WORKDIR /app/web
# Use pnpm as in the project
RUN corepack enable && corepack prepare pnpm@latest --activate
COPY web/package.json web/pnpm-lock.yaml ./
RUN pnpm install
COPY web/ ./
RUN pnpm build

# Stage 2: Build the Go backend
FROM golang:1.24-alpine AS backend-builder
WORKDIR /app
# Install build dependencies
RUN apk add --no-cache git
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Copy built frontend from Stage 1
COPY --from=frontend-builder /app/web/dist ./web/dist
RUN go build -o bin/radar ./cmd/radar

# Stage 3: Final image
FROM alpine:latest
RUN apk add --no-cache ca-certificates git
WORKDIR /app
COPY --from=backend-builder /app/bin/radar .
# Create a default data directory
RUN mkdir -p /data/.tech-radar
# Set the data directory as a volume
VOLUME /data
# Command to run the server
# We assume it should run in 'serve' mode on port 8088
EXPOSE 8088
ENTRYPOINT ["./radar"]
CMD ["serve", "--port", "8088", "--data-dir", "/data"]
