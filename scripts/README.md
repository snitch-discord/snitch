# Certificate Management

This directory contains scripts for managing TLS certificates for the Snitch Discord bot system.

## Certificate Generation Script

The `generate-certs.sh` script automatically generates Ed25519 TLS certificates for all services.

### Usage

```bash
# Generate certificates (if they don't exist)
./scripts/generate-certs.sh

# Verify existing certificates
./scripts/generate-certs.sh --verify

# Force regenerate all certificates
./scripts/generate-certs.sh --force

# Show help
./scripts/generate-certs.sh --help
```

### What it generates

- **CA Certificate** (`certs/ca/ca-cert.pem`): Self-signed Certificate Authority
- **CA Private Key** (`certs/ca/ca-key.pem`): CA private key (keep secure!)
- **Database Service Certificate** (`certs/db/cert.pem`): For database service (port 5200)
- **Backend Service Certificate** (`certs/backend/cert.pem`): For backend service (port 4200)
- **Bot Service Certificate** (`certs/bot/cert.pem`): For bot client connections

### Certificate Details

- **Algorithm**: Ed25519 (modern, fast, secure)
- **CA Validity**: 10 years
- **Service Certificates Validity**: 1 year
- **Subject Alternative Names**: Each service certificate includes:
  - `snitch-{service}` (Docker service name)
  - `localhost` (local development)
  - `127.0.0.1` (loopback IP)

### Automatic Integration

The certificate generation is automatically integrated into the main `run.sh` script:

1. When you run `./run.sh`, it checks for existing certificates
2. If certificates don't exist, it automatically generates them
3. If certificates exist, it verifies they're valid
4. If verification fails, you'll be prompted to regenerate

### Security Notes

- Certificates are automatically excluded from git (see `.gitignore`)
- Private keys have restricted permissions (600)
- Certificates use strong Ed25519 cryptography
- All service-to-service communication is encrypted and authenticated

### Troubleshooting

If you encounter certificate issues:

1. **Regenerate certificates**: `./scripts/generate-certs.sh --force`
2. **Check certificate validity**: `./scripts/generate-certs.sh --verify`
3. **Ensure OpenSSL supports Ed25519**: Update OpenSSL if needed
4. **Verify Docker volume mounts**: Check that certificates are mounted in containers

For additional help, see the main project documentation.