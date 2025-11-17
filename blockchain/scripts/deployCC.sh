#!/bin/bash
set -e

# ---------------------------------------------------------
# CONFIG
# ---------------------------------------------------------

CHAINCODE_NAME="transfer"
CHAINCODE_VERSION="1.0"
CHAINCODE_LABEL="${CHAINCODE_NAME}_${CHAINCODE_VERSION}"
CHAINCODE_LANG="golang"
CHAINCODE_PATH="/opt/gopath/src/github.com/hyperledger/fabric/peer/chaincode/transfer"
CHANNEL_NAME="transfer-channel"

ORDERER_CA="/opt/gopath/src/github.com/hyperledger/fabric/peer/organizations/ordererOrganizations/union.gov/orderers/orderer.union.gov/msp/tlscacerts/tlsca.union.gov-cert.pem"
PEER_CA="/opt/gopath/src/github.com/hyperledger/fabric/peer/organizations/peerOrganizations/union.gov/peers/peer0.union.gov/tls/ca.crt"

# ---------------------------------------------------------
# ENTER CLI CONTAINER
# ---------------------------------------------------------

function exec_cli() {
  docker exec cli bash -c "$1"
}

echo "============================================================"
echo " STEP 1 — Packaging chaincode"
echo "============================================================"

exec_cli "peer lifecycle chaincode package ${CHAINCODE_NAME}.tar.gz \
  --path ${CHAINCODE_PATH} \
  --lang ${CHAINCODE_LANG} \
  --label ${CHAINCODE_LABEL}"

echo "✓ Chaincode packaged."

# ---------------------------------------------------------

echo "============================================================"
echo " STEP 2 — Installing chaincode on peer"
echo "============================================================"

exec_cli "peer lifecycle chaincode install ${CHAINCODE_NAME}.tar.gz"

echo "✓ Chaincode installed."

# ---------------------------------------------------------

echo "============================================================"
echo " STEP 3 — Querying installed chaincodes"
echo "============================================================"

PACKAGE_ID=$(docker exec cli bash -c "peer lifecycle chaincode queryinstalled | grep ${CHAINCODE_LABEL} | awk -F 'Package ID: ' '{print \$2}' | awk -F ',' '{print \$1}'")

if [ -z "$PACKAGE_ID" ]; then
  echo "ERROR: PACKAGE_ID not found!"
  exit 1
fi

echo "✓ PACKAGE_ID = $PACKAGE_ID"

# ---------------------------------------------------------

echo "============================================================"
echo " STEP 4 — Approving chaincode"
echo "============================================================"

exec_cli "peer lifecycle chaincode approveformyorg \
  -o orderer.union.gov:7050 \
  --tls \
  --cafile ${ORDERER_CA} \
  --channelID ${CHANNEL_NAME} \
  --name ${CHAINCODE_NAME} \
  --version ${CHAINCODE_VERSION} \
  --package-id ${PACKAGE_ID} \
  --sequence 1"

echo "✓ Chaincode approved."

# ---------------------------------------------------------

echo "============================================================"
echo " STEP 5 — Checking commit readiness"
echo "============================================================"

exec_cli "peer lifecycle chaincode checkcommitreadiness \
  --channelID ${CHANNEL_NAME} \
  --name ${CHAINCODE_NAME} \
  --version ${CHAINCODE_VERSION} \
  --sequence 1 \
  --output json"

# ---------------------------------------------------------

echo "============================================================"
echo " STEP 6 — Committing chaincode"
echo "============================================================"

exec_cli "peer lifecycle chaincode commit \
  -o orderer.union.gov:7050 \
  --tls \
  --cafile ${ORDERER_CA} \
  --channelID ${CHANNEL_NAME} \
  --name ${CHAINCODE_NAME} \
  --version ${CHAINCODE_VERSION} \
  --sequence 1 \
  --peerAddresses peer0.union.gov:7051 \
  --tlsRootCertFiles ${PEER_CA}"

echo "✓ Chaincode committed."

# ---------------------------------------------------------

echo "============================================================"
echo " STEP 7 — Querying committed chaincode"
echo "============================================================"

exec_cli "peer lifecycle chaincode querycommitted \
  --channelID ${CHANNEL_NAME} \
  --name ${CHAINCODE_NAME}"

# ---------------------------------------------------------

echo "============================================================"
echo " DEPLOY COMPLETE!"
echo "============================================================"

echo "You can now invoke:"
echo "docker exec -it cli peer chaincode invoke -o orderer.union.gov:7050 --tls --cafile ${ORDERER_CA} -C ${CHANNEL_NAME} -n ${CHAINCODE_NAME} -c '{\"Args\":[\"InitLedger\"]}'"