#!/usr/bin/env bash

# Certificate generation script for Snitch Discord bot system
# Generates Ed25519 TLS certificates for internal service communication

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
CERTS_DIR="$PROJECT_ROOT/certs"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if openssl is available
check_openssl() {
    if ! command -v openssl &> /dev/null; then
        error "OpenSSL is required but not installed"
        exit 1
    fi
    
    # Check if Ed25519 is supported
    if ! openssl genpkey -algorithm Ed25519 -help &> /dev/null; then
        error "OpenSSL version does not support Ed25519 algorithm"
        exit 1
    fi
    
    info "OpenSSL version: $(openssl version)"
}

# Create certificate directory structure
create_dirs() {
    info "Creating certificate directory structure..."
    mkdir -p "$CERTS_DIR"/{ca,db,backend,bot}
}

# Generate CA certificate and key
generate_ca() {
    local ca_dir="$CERTS_DIR/ca"
    local ca_key="$ca_dir/ca-key.pem"
    local ca_cert="$ca_dir/ca-cert.pem"
    
    if [[ -f "$ca_cert" && -f "$ca_key" ]]; then
        warn "CA certificate already exists. Skipping CA generation."
        return 0
    fi
    
    info "Generating CA private key..."
    openssl genpkey -algorithm Ed25519 -out "$ca_key"
    
    info "Generating CA certificate..."
    openssl req -new -x509 -key "$ca_key" -sha256 -days 3650 -out "$ca_cert" \
        -subj "/C=US/ST=CA/L=SF/O=Snitch/OU=Internal/CN=Snitch CA"
    
    info "CA certificate generated: $ca_cert"
}

# Generate service certificate
generate_service_cert() {
    local service="$1"
    local service_dir="$CERTS_DIR/$service"
    local service_key="$service_dir/key.pem"
    local service_cert="$service_dir/cert.pem"
    local service_csr="$service_dir/cert.csr"
    local service_conf="$service_dir/cert.conf"
    
    local ca_key="$CERTS_DIR/ca/ca-key.pem"
    local ca_cert="$CERTS_DIR/ca/ca-cert.pem"
    
    if [[ -f "$service_cert" && -f "$service_key" ]]; then
        warn "$service certificate already exists. Skipping $service certificate generation."
        return 0
    fi
    
    info "Generating $service private key..."
    openssl genpkey -algorithm Ed25519 -out "$service_key"
    
    # Create certificate configuration
    local cn="snitch-$service"
    local ext_key_usage="serverAuth"
    local ou_name=""
    
    # Set proper organizational unit names and key usage
    case "$service" in
        "db")
            ou_name="Database"
            ;;
        "backend")
            ou_name="Backend"
            ;;
        "bot")
            ou_name="Bot"
            ext_key_usage="clientAuth"  # Bot acts as client
            ;;
        *)
            ou_name="Service"
            ;;
    esac
    
    info "Creating $service certificate configuration..."
    cat > "$service_conf" << EOF
[req]
distinguished_name = req_distinguished_name
req_extensions = v3_req
prompt = no

[req_distinguished_name]
C = US
ST = CA
L = SF
O = Snitch
OU = $ou_name
CN = $cn

[v3_req]
keyUsage = keyEncipherment, dataEncipherment
extendedKeyUsage = $ext_key_usage
subjectAltName = @alt_names

[alt_names]
DNS.1 = $cn
DNS.2 = localhost
IP.1 = 127.0.0.1
EOF
    
    info "Generating $service certificate signing request..."
    openssl req -new -key "$service_key" -out "$service_csr" -config "$service_conf"
    
    info "Signing $service certificate..."
    openssl x509 -req -in "$service_csr" -CA "$ca_cert" -CAkey "$ca_key" \
        -CAcreateserial -out "$service_cert" -days 365 -sha256 \
        -extensions v3_req -extfile "$service_conf"
    
    # Clean up temporary files
    rm -f "$service_csr" "$service_conf"
    
    info "$service certificate generated: $service_cert"
}

# Verify certificate chain
verify_certificates() {
    local ca_cert="$CERTS_DIR/ca/ca-cert.pem"
    
    for service in db backend bot; do
        local service_cert="$CERTS_DIR/$service/cert.pem"
        if [[ -f "$service_cert" ]]; then
            info "Verifying $service certificate..."
            if openssl verify -CAfile "$ca_cert" "$service_cert" > /dev/null 2>&1; then
                info "✓ $service certificate is valid"
            else
                error "✗ $service certificate verification failed"
                exit 1
            fi
        fi
    done
}

# Set proper permissions
set_permissions() {
    info "Setting certificate permissions..."
    
    # CA key should be most restrictive
    chmod 600 "$CERTS_DIR/ca/ca-key.pem" 2>/dev/null || true
    
    # Service keys should be restrictive
    for service in db backend bot; do
        chmod 600 "$CERTS_DIR/$service/key.pem" 2>/dev/null || true
    done
    
    # Certificates can be more permissive (readable)
    find "$CERTS_DIR" -name "*.pem" -not -name "*-key.pem" -not -name "key.pem" -exec chmod 644 {} \; 2>/dev/null || true
}

# Display certificate information
show_certificate_info() {
    local ca_cert="$CERTS_DIR/ca/ca-cert.pem"
    
    if [[ -f "$ca_cert" ]]; then
        info "Certificate Authority Information:"
        openssl x509 -in "$ca_cert" -noout -subject -dates
        echo
        
        for service in db backend bot; do
            local service_cert="$CERTS_DIR/$service/cert.pem"
            if [[ -f "$service_cert" ]]; then
                info "$service Service Certificate Information:"
                openssl x509 -in "$service_cert" -noout -subject -dates
                echo "  Subject Alternative Names:"
                openssl x509 -in "$service_cert" -noout -text | grep -A 1 "Subject Alternative Name:" || echo "    None"
                echo
            fi
        done
    fi
}

# Main function
main() {
    info "Starting certificate generation for Snitch services..."
    
    check_openssl
    create_dirs
    generate_ca
    
    for service in db backend bot; do
        generate_service_cert "$service"
    done
    
    verify_certificates
    set_permissions
    show_certificate_info
    
    info "Certificate generation completed successfully!"
    info "Certificates are located in: $CERTS_DIR"
}

# Handle script arguments
case "${1:-}" in
    --force)
        warn "Force mode: removing existing certificates..."
        rm -rf "$CERTS_DIR"
        main
        ;;
    --verify)
        info "Verifying existing certificates..."
        verify_certificates
        show_certificate_info
        ;;
    --help|-h)
        echo "Usage: $0 [--force|--verify|--help]"
        echo "  --force   Remove existing certificates and regenerate all"
        echo "  --verify  Verify existing certificates without regenerating"
        echo "  --help    Show this help message"
        exit 0
        ;;
    "")
        main
        ;;
    *)
        error "Unknown option: $1"
        echo "Use --help for usage information"
        exit 1
        ;;
esac
