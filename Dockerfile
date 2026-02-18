# Stage 1: Build frontend
FROM node:22-alpine AS frontend
WORKDIR /app/web
COPY web/package.json web/package-lock.json* ./
RUN npm ci --silent
COPY web/ .
RUN npm run build

# Stage 2: Build Go binary
FROM golang:1.22-alpine AS backend
RUN apk add --no-cache git
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend /app/cmd/dist ./cmd/dist
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags "-s -w -X main.version=$(git describe --tags --always 2>/dev/null || echo docker)" \
    -o /odyssey .

# Stage 3: Minimal runtime
FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=backend /odyssey /usr/local/bin/odyssey
EXPOSE 8080
ENTRYPOINT ["odyssey"]
CMD ["server", "-p", "8080"]
