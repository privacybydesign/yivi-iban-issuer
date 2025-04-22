# Yivi IBAN Issuer

This repository implements a system for verifying IBANs using iDEAL payments and issuing credentials via Yivi. It consists of a Go-based backend and a React-based frontend.

## Prerequisites

- **Node.js**: For running the React frontend.
- **Go**: For running the backend.
- **Docker**: For containerized deployment.
- **OpenSSL**: For generating private/public keys.

## Setup Instructions

### 1. Generate Secrets

Run the following commands to generate the private and public keys:

```bash
mkdir -p .secrets
openssl genrsa 4096 > .secrets/priv.pem
openssl rsa -in .secrets/priv.pem -pubout > .secrets/pub.pem
```

### 2. Install Dependencies

#### Backend
Navigate to the `server` directory and install Go dependencies:

```bash
cd server
go mod download
```

#### Frontend
Navigate to the `react-cra` directory and install Node.js dependencies:

```bash
cd react-cra
npm install
```

## Running the Application

### 1. Run Backend

Start the backend server with the following command:

```bash
go run . --config ../local-secrets/local.json
```

### 2. Run Frontend

Start the React app:

```bash
cd react-cra
npm start
```

The frontend will be available at `http://localhost:3000`.

---

## Docker Deployment

To deploy the application using Docker, run:

```bash
docker-compose up --build
```

This will start the backend, frontend, and required services (e.g., Redis, IRMA server).

## Configuration

The backend configuration is stored in `local-secrets/local.json`. Key fields include:

- `server_config`: Host and port for the backend server.
- `jwt_private_key_path`: Path to the private key for JWT signing.
- `cm_iban_config`: Configuration for CM's IBAN verification API.

---

## License

This project is licensed under the [Apache License 2.0](LICENSE).

---

## Contributing

Contributions are welcome! Please fork the repository and submit a pull request.