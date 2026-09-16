FROM node:22-alpine AS frontend
WORKDIR /src/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

FROM golang:1.27-alpine AS backend
WORKDIR /src/backend
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN CGO_ENABLED=0 go build -o /out/server ./cmd/server

FROM alpine:3.22
WORKDIR /app
COPY --from=backend /out/server ./server
COPY --from=frontend /src/frontend/dist ./web
ENV STATIC_DIR=/app/web
ENV DB_PATH=/data/chatbot.db
EXPOSE 8080
ENTRYPOINT ["/app/server"]
