FROM node:25-slim AS frontend-build
WORKDIR /app/frontend
COPY react-cra .
RUN npm install
RUN npm run build

# -----------------------------------------------------

FROM golang:1.27.2@sha256:820153096fda4de6ac1b7194d12e416b7dd8f4e9a33d17e0345b111b945162d8 AS backend-build
WORKDIR /app/backend
COPY server .
RUN go mod download

# compile with static linking
RUN CGO_ENABLED=0 go build -o server 

# -----------------------------------------------------

FROM alpine:latest

COPY --from=backend-build /app/backend/server /app/backend/server
# Only the build output; static_path in the config is ../frontend/build.
COPY --from=frontend-build /app/frontend/build /app/frontend/build

WORKDIR /app/backend
EXPOSE 8080
CMD ["./server", "--config", "/secrets/local.json"]