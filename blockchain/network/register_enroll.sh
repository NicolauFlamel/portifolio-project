#!/bin/bash
#
# Usage:
#   ./register_enroll.sh org1
#   ./register_enroll.sh org2
#
# Creates:
#   blockchain/network/wallet/<org>/appUser/msp/*
#

set -e

ORG=$1
if [ -z "$ORG" ]; then
  echo "❌ Usage: $0 <org1|org2>"
  exit 1
fi

# ------------------------------------------------------------
# 1. Resolve absolute network root
# ------------------------------------------------------------
NETWORK_ROOT=$(cd "$(dirname "$0")" && pwd)

# ------------------------------------------------------------
# 2. Determine CA info for the selected org
# ------------------------------------------------------------
case "$ORG" in
  org1)
    CA_NAME="ca-org1"
    CA_PORT=7054
    ;;
  org2)
    CA_NAME="ca-org2"
    CA_PORT=8054
    ;;
  *)
    echo "❌ Unknown org: $ORG (expected org1 or org2)"
    exit 1
    ;;
esac

# ------------------------------------------------------------
# 3. Define CA certificate + client home paths
# ------------------------------------------------------------
TLS_CERT="${NETWORK_ROOT}/organizations/fabric-ca/${ORG}/tls-cert.pem"
FABRIC_CA_CLIENT_HOME="${NETWORK_ROOT}/organizations/peerOrganizations/${ORG}.example.com"

export PATH="${NETWORK_ROOT}/../bin:$PATH"
export FABRIC_CFG_PATH="${NETWORK_ROOT}/../config"
export FABRIC_CA_CLIENT_HOME

# Where we will store the wallet
WALLET_PATH="${NETWORK_ROOT}/wallet/${ORG}/appUser/msp"

# ------------------------------------------------------------
# 4. Validate CA certificate
# ------------------------------------------------------------
if [ ! -f "$TLS_CERT" ]; then
  echo "❌ TLS certificate not found at:"
  echo "    $TLS_CERT"
  echo "Make sure the network is UP with '-ca'"
  exit 1
fi

echo "✅ Using TLS cert:             $TLS_CERT"
echo "✅ CA:                         $CA_NAME ($CA_PORT)"
echo "✅ FABRIC_CA_CLIENT_HOME:      $FABRIC_CA_CLIENT_HOME"
echo "✅ Wallet output directory:    $WALLET_PATH"

# ------------------------------------------------------------
# 5. Register appUser (if not already)
# ------------------------------------------------------------
echo "🔹 Registering appUser..."

set +e
fabric-ca-client register \
  --caname "${CA_NAME}" \
  --id.name appUser \
  --id.secret appUserpw \
  --id.type client \
  --tls.certfiles "${TLS_CERT}" 2>/dev/null
set -e

# ------------------------------------------------------------
# 6. Enroll appUser into wallet
# ------------------------------------------------------------
echo "🔹 Enrolling appUser..."

mkdir -p "${WALLET_PATH}"

fabric-ca-client enroll \
  -u "https://appUser:appUserpw@localhost:${CA_PORT}" \
  --caname "${CA_NAME}" \
  -M "${WALLET_PATH}" \
  --tls.certfiles "${TLS_CERT}"

# ------------------------------------------------------------
# 7. Inject config.yaml
# ------------------------------------------------------------
cat > "${WALLET_PATH}/config.yaml" <<EOF
NodeOUs:
  Enable: true
  ClientOUIdentifier:
    Certificate: cacerts/localhost-${CA_PORT}-${CA_NAME}.pem
    OrganizationalUnitIdentifier: client
EOF

# ------------------------------------------------------------
# 8. Validate certificate creation
# ------------------------------------------------------------
CERT_FILE="${WALLET_PATH}/signcerts/cert.pem"

if [ ! -f "${CERT_FILE}" ]; then
  echo "❌ Enrollment FAILED — cert.pem not found at:"
  echo "   ${CERT_FILE}"
  exit 1
fi

echo "✅ Enrollment COMPLETE for ${ORG}"
echo "📂 Wallet stored at:"
echo "   ${WALLET_PATH}"