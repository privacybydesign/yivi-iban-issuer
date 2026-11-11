FROM node:25-slim AS frontend-build
WORKDIR /app/frontend
COPY react-cra .
RUN npm install
RUN npm run build

# -----------------------------------------------------

FROM golang:1.27.0@sha256:4013ae0f9e7994f8535c58c811f8f863fbed38b72e0d51e6592156f758d66146 AS backend-build
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