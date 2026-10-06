# Stage 1: build SPA admin
FROM node:22-alpine AS web
WORKDIR /src/web/admin
COPY web/admin/package*.json ./
RUN npm ci
COPY web/admin/ ./
# vite outDir = ../dist
RUN npm run build

# Stage 2: build Go binary
FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

# Stage 3: runtime
FROM alpine:3.20
RUN adduser -D -u 10001 app
WORKDIR /app
COPY --from=build /out/server ./server
COPY --from=web /src/web/dist ./web/dist
USER app
ENV APP_ENV=production APP_PORT=8080
EXPOSE 8080
ENTRYPOINT ["./server"]
