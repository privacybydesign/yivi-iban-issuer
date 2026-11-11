FROM node:24-slim AS frontend-build
WORKDIR /app/frontend
COPY react-cra .
RUN npm install
RUN npm run build

# -----------------------------------------------------

FROM golang:1.26.11@sha256:7af425da11f4fc1a9308eb704369e3f6d1c44e35bac0f874a59ade3f20fdc07e AS backend-build
WORKDIR /app/backend
COPY server .
RUN go mod download

# compile with static linking
RUN CGO_ENABLED=0 go build -o server 

# -----------------------------------------------------

FROM alpine:3.24@sha256:be2656061b536c1883c4034790e21d0a09c5aaaf51968b89368a3c0371ea2c9a

COPY --from=backend-build /app/backend/server /app/backend/server
# Only the build output; static_path in the config is ../frontend/build.
COPY --from=frontend-build /app/frontend/build /app/frontend/build

WORKDIR /app/backend
EXPOSE 8080
CMD ["./server", "--config", "/secrets/local.json"]