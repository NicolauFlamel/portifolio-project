#!/bin/bash

# Government Spending Blockchain - Chaincode Deployment Script

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
NETWORK_DIR="$PROJECT_DIR/network"
CHAINCODE_DIR="$PROJECT_DIR/chaincode/spending"
export PATH="$PROJECT_DIR/bin:$PATH"

CC_NAME="spending"
CC_VERSION="1.0"
CC_SEQUENCE="1"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_step() {
    echo -e "\n${CYAN}==== $1 ====${NC}"
}

set_peer_env() {
    local org=$1
    local port=$2
    local domain=$3
    
    export CORE_PEER_TLS_ENABLED=true
    export CORE_PEER_LOCALMSPID="${org}MSP"
    export CORE_PEER_TLS_ROOTCERT_FILE="$NETWORK_DIR/crypto-config/peerOrganizations/$domain/peers/peer0.$domain/tls/ca.crt"
    export CORE_PEER_MSPCONFIGPATH="$NETWORK_DIR/crypto-config/peerOrganizations/$domain/users/Admin@$domain/msp"
    export CORE_PEER_ADDRESS="localhost:$port"
}

package_chaincode() {
    log_step "Packaging chaincode"
    
    cd "$CHAINCODE_DIR"
    GO111MODULE=on go mod tidy
    GO111MODULE=on go mod vendor
    
    cd "$PROJECT_DIR"
    
    peer lifecycle chaincode package ${CC_NAME}.tar.gz \
        --path ./chaincode/spending \
        --lang golang \
        --label ${CC_NAME}_${CC_VERSION}
    
    log_info "Chaincode packaged: ${CC_NAME}.tar.gz"
}

install_chaincode_on_peer() {
    local org=$1
    local port=$2
    local domain=$3
    
    log_info "Installing chaincode on peer0.$domain..."
    
    set_peer_env "$org" "$port" "$domain"
    
    peer lifecycle chaincode install "$PROJECT_DIR/${CC_NAME}.tar.gz"
}

get_package_id() {
    local org=$1
    local port=$2
    local domain=$3
    
    set_peer_env "$org" "$port" "$domain"
    
    peer lifecycle chaincode queryinstalled --output json | \
        jq -r ".installed_chaincodes[] | select(.label==\"${CC_NAME}_${CC_VERSION}\") | .package_id"
}

deploy_to_channel() {
    local channel=$1
    local org=$2
    local port=$3
    local domain=$4
    local package_id=$5
    
    log_step "Deploying to $channel"
    
    set_peer_env "$org" "$port" "$domain"
    
    local ORDERER_CA="$NETWORK_DIR/crypto-config/ordererOrganizations/orderer.gov.br/orderers/orderer.orderer.gov.br/msp/tlscacerts/tlsca.orderer.gov.br-cert.pem"
    
    log_info "Approving chaincode for ${org}MSP on $channel..."
    peer lifecycle chaincode approveformyorg \
        -o localhost:7050 \
        --tls \
        --cafile "$ORDERER_CA" \
        --channelID "$channel" \
        --name "$CC_NAME" \
        --version "$CC_VERSION" \
        --package-id "$package_id" \
        --sequence "$CC_SEQUENCE"
    
    log_info "Checking commit readiness..."
    peer lifecycle chaincode checkcommitreadiness \
        --channelID "$channel" \
        --name "$CC_NAME" \
        --version "$CC_VERSION" \
        --sequence "$CC_SEQUENCE" \
        --output json
    
    log_info "Committing chaincode on $channel..."
    peer lifecycle chaincode commit \
        -o localhost:7050 \
        --tls \
        --cafile "$ORDERER_CA" \
        --channelID "$channel" \
        --name "$CC_NAME" \
        --version "$CC_VERSION" \
        --sequence "$CC_SEQUENCE" \
        --peerAddresses "localhost:$port" \
        --tlsRootCertFiles "$NETWORK_DIR/crypto-config/peerOrganizations/$domain/peers/peer0.$domain/tls/ca.crt"
    
    log_info "Verifying deployment on $channel..."
    peer lifecycle chaincode querycommitted \
        --channelID "$channel" \
        --name "$CC_NAME"
    
    log_info "Chaincode deployed to $channel"
}

test_chaincode() {
    local channel=$1
    local org=$2
    local port=$3
    local domain=$4
    
    log_info "Testing chaincode on $channel..."
    
    set_peer_env "$org" "$port" "$domain"
    
    local ORDERER_CA="$NETWORK_DIR/crypto-config/ordererOrganizations/orderer.gov.br/orderers/orderer.orderer.gov.br/msp/tlscacerts/tlsca.orderer.gov.br-cert.pem"
    
    peer chaincode query \
        -C "$channel" \
        -n "$CC_NAME" \
        -c '{"function":"ListDocumentTypes","Args":[""]}'
    
    log_info "Chaincode test on $channel: OK"
}

deploy_all() {
    log_step "Starting full chaincode deployment"
    
    export FABRIC_CFG_PATH="$NETWORK_DIR"
    
    package_chaincode

    log_step "Installing chaincode on all peers"
    install_chaincode_on_peer "Union" "7051" "union.gov.br"
    install_chaincode_on_peer "State" "9051" "state.gov.br"
    install_chaincode_on_peer "Region" "11051" "region.gov.br"
    
    PACKAGE_ID=$(get_package_id "Union" "7051" "union.gov.br")
    log_info "Package ID: $PACKAGE_ID"
    
    if [ -z "$PACKAGE_ID" ]; then
        log_error "Failed to get package ID"
        exit 1
    fi
    
    deploy_to_channel "union-channel" "Union" "7051" "union.gov.br" "$PACKAGE_ID"
    deploy_to_channel "state-channel" "State" "9051" "state.gov.br" "$PACKAGE_ID"
    deploy_to_channel "region-channel" "Region" "11051" "region.gov.br" "$PACKAGE_ID"
    
    log_step "Testing chaincode on all channels"
    test_chaincode "union-channel" "Union" "7051" "union.gov.br"
    test_chaincode "state-channel" "State" "9051" "state.gov.br"
    test_chaincode "region-channel" "Region" "11051" "region.gov.br"
    
    rm -f "$PROJECT_DIR/${CC_NAME}.tar.gz"
    
    echo ""
    log_info "=========================================="
    log_info "Chaincode deployment completed!"
    log_info "=========================================="
    log_info ""
    log_info "Chaincode '$CC_NAME' deployed to:"
    log_info "  - union-channel"
    log_info "  - state-channel"
    log_info "  - region-channel"
    log_info ""
    log_info "Next: Start the backend API"
    log_info "  cd backend && go run cmd/api/main.go"
    log_info "=========================================="
}

case "${1:-}" in
    package)
        export FABRIC_CFG_PATH="$NETWORK_DIR"
        package_chaincode
        ;;
    deploy)
        deploy_all
        ;;
    test)
        export FABRIC_CFG_PATH="$NETWORK_DIR"
        test_chaincode "union-channel" "Union" "7051" "union.gov.br"
        test_chaincode "state-channel" "State" "9051" "state.gov.br"
        test_chaincode "region-channel" "Region" "11051" "region.gov.br"
        ;;
    *)
        echo "Usage: $0 {package|deploy|test}"
        echo ""
        echo "Commands:"
        echo "  package - Package the chaincode"
        echo "  deploy  - Full deployment to all channels"
        echo "  test    - Test chaincode on all channels"
        exit 1
        ;;
esac