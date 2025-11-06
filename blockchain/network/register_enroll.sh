#!/bin/bash
#
# Usage:
#   ./register_enroll.sh org1
#   ./register_enroll.sh org2
#   ./register_enroll.sh org3
#
# Result:
#   wallet/<org>/appUser/msp/  (with all certs + keys)
#

set -e

ORG=$1
if [ -z "$ORG" ]; then
  echo "❌ Usage: $0 <org1|org2|org3>"
  exit 1
fi

# === 1. Determine CA info per org =====================================
case "$ORG" in
  org1)
    CA_NAME="ca-org1"
    CA_PORT=7054
    ;;
  org2)
    CA_NAME="ca-org2"
    CA_PORT=8054
    ;;
  org3)
    CA_NAME="ca-org3"
    CA_PORT=9054
    ;;
  *)
    echo "❌ Unknown org: $ORG (expected org1, org2, or org3)"
    exit 1
    ;;
esac

# === 2. Set environment ===============================================
NETWORK_ROOT=$(pwd)
FABRIC_CA_CLIENT_HOME="$NETWORK_ROOT/organizations/peerOrganizations/${ORG}.example.com"
TLS_CERT="$NETWORK_ROOT/organizations/fabric-ca/${ORG}/tls-cert.pem"

export PATH=${NETWORK_ROOT}/../bin:$PATH
export FABRIC_CFG_PATH=${NETWORK_ROOT}/../config
export FABRIC_CA_CLIENT_HOME

# === 3. Check CA certificate ==========================================
if [ ! -f "$TLS_CERT" ]; then
  echo "❌ TLS cert not found: $TLS_CERT"
  echo "Make sure the network is running with -ca"
  exit 1
fi

echo "✅ Using TLS cert: $TLS_CERT"
echo "✅ Using CA: $CA_NAME at port $CA_PORT"
echo "✅ Fabric CA client home: $FABRIC_CA_CLIENT_HOME"

# === 4. Register appUser (ignore if already exists) ==================
echo "🔹 Registering appUser with $CA_NAME ..."
set +e
fabric-ca-client register \
  --caname ${CA_NAME} \
  --id.name appUser \
  --id.secret appUserpw \
  --id.type client \
  --tls.certfiles ${TLS_CERT} 2>/dev/null
set -e

# === 5. Enroll appUser ================================================
echo "🔹 Enrolling appUser ..."
fabric-ca-client enroll \
  -u https://appUser:appUserpw@localhost:${CA_PORT} \
  --caname ${CA_NAME} \
  -M ${NETWORK_ROOT}/wallet/${ORG}/appUser/msp \
  --tls.certfiles ${TLS_CERT}

# === 6. Add config.yaml for NodeOUs ===================================
cat > wallet/${ORG}/appUser/msp/config.yaml <<EOF
NodeOUs:
  Enable: true
  ClientOUIdentifier:
    Certificate: cacerts/localhost-${CA_PORT}.pem
    OrganizationalUnitIdentifier: client
EOF

echo "✅ Enrollment complete for ${ORG}"
echo "📂 Certificates stored at: wallet/${ORG}/appUser/msp"