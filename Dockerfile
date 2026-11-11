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

FROM gcr.io/distroless/static-debian12:nonroot@sha256:afa5c872c891853ca7fcf1f12c3edb23f7eeef36189728842dd51042ff57f7ab

COPY --from=backend-build /app/backend/server /app/backend/server
COPY --from=frontend-build /app/frontend/build/ /app/frontend/build

# Declared explicitly to satisfy static scanners; the :nonroot tag already
# defaults to this same user (UID 65532).
USER nonroot:nonroot

EXPOSE 8080
ENTRYPOINT [ "/app/backend/server", "--config", "/secrets/local.json" ]