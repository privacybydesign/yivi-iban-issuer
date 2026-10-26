FROM node:23-slim AS frontend-build
WORKDIR /app/frontend
COPY react-cra .
RUN npm install
RUN npm run build

# -----------------------------------------------------

FROM golang:1.26.10@sha256:672c944cfd07599b5f1bb179869270cdc94c6254e41db4d431259693cb286b82 AS backend-build
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