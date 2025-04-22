```bash
mkdir -p .secrets
openssl genrsa 4096 > .secrets/priv.pem
openssl rsa -in .secrets/priv.pem -pubout > .secrets/pub.pem
```

irma scheme download https://schemes.staging.yivi.app/pbdf-staging

irma server --no-tls --no-auth=false --port=8088 --config=./local-secrets/irma-server/config.json

go run . --config ../local-secrets/local.json