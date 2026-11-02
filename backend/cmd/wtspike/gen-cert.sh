#!/usr/bin/env bash
# Generate a local CA + localhost server certificate for the wtspike prototype.
# Usage: ./gen-cert.sh [output-dir]   (default: /tmp/wtcert)
set -euo pipefail

dir="${1:-/tmp/wtcert}"
mkdir -p "$dir"
cd "$dir"

openssl req -x509 -newkey ec -pkeyopt ec_paramgen_curve:prime256v1 -nodes \
  -keyout ca.key -out ca.crt -days 30 -subj "/CN=wtspike-local-ca"

openssl req -newkey ec -pkeyopt ec_paramgen_curve:prime256v1 -nodes \
  -keyout srv.key -out srv.csr -subj "/CN=localhost"

printf 'subjectAltName=DNS:localhost,IP:127.0.0.1,IP:::1\nextendedKeyUsage=serverAuth\n' > ext.cnf

openssl x509 -req -in srv.csr -CA ca.crt -CAkey ca.key -CAcreateserial \
  -out srv.crt -days 30 -extfile ext.cnf >/dev/null

echo "CA:      $dir/ca.crt"
echo "Server:  $dir/srv.crt  $dir/srv.key"
echo
if command -v update-ca-certificates >/dev/null 2>&1; then
  echo "Trust the CA for OS tools (Debian/Ubuntu):"
  echo "  sudo cp $dir/ca.crt /usr/local/share/ca-certificates/wtspike-local-ca.crt"
  echo "  sudo update-ca-certificates"
elif command -v trust >/dev/null 2>&1; then
  echo "Trust the CA for OS tools (Arch/CachyOS):"
  echo "  sudo trust anchor --store $dir/ca.crt"
fi
echo
echo "Chrome 132+ needs an enterprise policy for QUIC/WebTransport trust:"
echo "  CERT_B64=\$(openssl x509 -in $dir/ca.crt -outform der | base64 -w0)"
echo "  sudo mkdir -p /etc/opt/chrome/policies/managed"
echo "  printf '{\"CACertificates\":[\"%s\"]}\\n' \"\$CERT_B64\" | sudo tee /etc/opt/chrome/policies/managed/wtspike-ca.json >/dev/null"
echo "  # restart Chrome, verify chrome://policy -> CACertificates"
echo
echo "Remove Chrome policy afterwards:"
echo "  sudo rm /etc/opt/chrome/policies/managed/wtspike-ca.json"
echo
echo "Run the spike:"
echo "  cd backend && go run ./cmd/wtspike -addr :9443 -cert $dir/srv.crt -key $dir/srv.key"
