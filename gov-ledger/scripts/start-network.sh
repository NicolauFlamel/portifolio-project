#!/bin/bash

# Government Spending Blockchain - Network Startup Script

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
NETWORK_DIR="$PROJECT_DIR/network"

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

check_prerequisites() {
    log_step "Checking prerequisites"
    
    if ! command -v docker &> /dev/null; then
        log_error "Docker is not installed"
        exit 1
    fi
    log_info "Docker: $(docker --version)"
    
    if docker compose version &> /dev/null; then
        DOCKER_COMPOSE="docker compose"
    elif command -v docker-compose &> /dev/null; then
        DOCKER_COMPOSE="docker-compose"
    else
        log_error "Docker Compose is not installed"
        exit 1
    fi
    log_info "Docker Compose available"
    
    if ! command -v cryptogen &> /dev/null; then
        log_error "Fabric binaries not found in PATH"
        log_info "Run: curl -sSL https://bit.ly/2ysbOFE | bash -s -- 2.5.0 1.5.7 -s -d"
        exit 1
    fi
    log_info "Fabric binaries available"
}

generate_crypto() {
    log_step "Generating crypto materials"
    
    cd "$NETWORK_DIR"
    rm -rf crypto-config
    cryptogen generate --config=crypto-config.yaml --output=crypto-config
    
    log_info "Crypto materials generated"
}

generate_channel_artifacts() {
    log_step "Generating channel artifacts"
    
    cd "$NETWORK_DIR"
    rm -rf channel-artifacts
    mkdir -p channel-artifacts
    
    export FABRIC_CFG_PATH="$NETWORK_DIR"
    
    log_info "Creating union-channel genesis block..."
    configtxgen -profile UnionChannel -outputBlock ./channel-artifacts/union-channel.block -channelID union-channel
    
    log_info "Creating state-channel genesis block..."
    configtxgen -profile StateChannel -outputBlock ./channel-artifacts/state-channel.block -channelID state-channel
    
    log_info "Creating region-channel genesis block..."
    configtxgen -profile RegionChannel -outputBlock ./channel-artifacts/region-channel.block -channelID region-channel
    
    log_info "Channel artifacts generated"
}

start_containers() {
    log_step "Starting Docker containers"
    
    cd "$NETWORK_DIR"
    
    $DOCKER_COMPOSE down -v 2>/dev/null || true
    docker volume prune -f 2>/dev/null || true
    
    $DOCKER_COMPOSE up -d
    
    log_info "Waiting for containers to start..."
    sleep 10
    
    log_info "Container status:"
    docker ps --filter "network=gov-spending-network" --format "table {{.Names}}\t{{.Status}}"
}

join_orderer_to_channel() {
    local channel_name=$1
    
    log_info "Joining orderer to $channel_name..."
    
    osnadmin channel join \
        --channelID "$channel_name" \
        --config-block "$NETWORK_DIR/channel-artifacts/${channel_name}.block" \
        -o localhost:7053 \
        --ca-file "$NETWORK_DIR/crypto-config/ordererOrganizations/orderer.gov.br/orderers/orderer.orderer.gov.br/msp/tlscacerts/tlsca.orderer.gov.br-cert.pem" \
        --client-cert "$NETWORK_DIR/crypto-config/ordererOrganizations/orderer.gov.br/orderers/orderer.orderer.gov.br/tls/server.crt" \
        --client-key "$NETWORK_DIR/crypto-config/ordererOrganizations/orderer.gov.br/orderers/orderer.orderer.gov.br/tls/server.key"
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

join_peer_to_channel() {
    local org=$1
    local port=$2
    local domain=$3
    local channel=$4
    
    log_info "Joining peer0.$domain to $channel..."
    
    set_peer_env "$org" "$port" "$domain"
    
    peer channel join \
        -b "$NETWORK_DIR/channel-artifacts/${channel}.block" \
        -o localhost:7050 \
        --tls \
        --cafile "$NETWORK_DIR/crypto-config/ordererOrganizations/orderer.gov.br/orderers/orderer.orderer.gov.br/msp/tlscacerts/tlsca.orderer.gov.br-cert.pem"
}

setup_channels() {
    log_step "Setting up channels"
    
    cd "$NETWORK_DIR"
    export FABRIC_CFG_PATH="$NETWORK_DIR"
    
    log_info "Waiting for orderer to be ready..."
    sleep 5
    
    join_orderer_to_channel "union-channel"
    join_orderer_to_channel "state-channel"
    join_orderer_to_channel "region-channel"
    
    log_info "Channels on orderer:"
    osnadmin channel list \
        -o localhost:7053 \
        --ca-file "$NETWORK_DIR/crypto-config/ordererOrganizations/orderer.gov.br/orderers/orderer.orderer.gov.br/msp/tlscacerts/tlsca.orderer.gov.br-cert.pem" \
        --client-cert "$NETWORK_DIR/crypto-config/ordererOrganizations/orderer.gov.br/orderers/orderer.orderer.gov.br/tls/server.crt" \
        --client-key "$NETWORK_DIR/crypto-config/ordererOrganizations/orderer.gov.br/orderers/orderer.orderer.gov.br/tls/server.key"
    
    sleep 3
    
    join_peer_to_channel "Union" "7051" "union.gov.br" "union-channel"
    join_peer_to_channel "State" "9051" "state.gov.br" "state-channel"
    join_peer_to_channel "Region" "11051" "region.gov.br" "region-channel"
    
    log_info "All peers joined to channels"
}

verify_network() {
    log_step "Verifying network status"
    
    export FABRIC_CFG_PATH="$NETWORK_DIR"
    
    set_peer_env "Union" "7051" "union.gov.br"
    log_info "Channels for peer0.union.gov.br:"
    peer channel list
    
    set_peer_env "State" "9051" "state.gov.br"
    log_info "Channels for peer0.state.gov.br:"
    peer channel list
    
    set_peer_env "Region" "11051" "region.gov.br"
    log_info "Channels for peer0.region.gov.br:"
    peer channel list
}

stop_network() {
    log_step "Stopping network"
    cd "$NETWORK_DIR"
    $DOCKER_COMPOSE down -v 2>/dev/null || docker-compose down -v 2>/dev/null || true
    log_info "Network stopped"
}

cleanup() {
    log_step "Cleaning up"
    cd "$NETWORK_DIR"
    $DOCKER_COMPOSE down -v 2>/dev/null || docker-compose down -v 2>/dev/null || true
    rm -rf crypto-config channel-artifacts
    docker volume prune -f 2>/dev/null || true
    log_info "Cleanup completed"
}

start_network() {
    check_prerequisites
    generate_crypto
    generate_channel_artifacts
    start_containers
    setup_channels
    verify_network
    
    echo ""
    log_info "=========================================="
    log_info "Network is ready!"
    log_info "=========================================="
    log_info ""
    log_info "Architecture:"
    log_info "  - union-channel:  UnionMSP (peer0.union.gov.br:7051)"
    log_info "  - state-channel:  StateMSP (peer0.state.gov.br:9051)"
    log_info "  - region-channel: RegionMSP (peer0.region.gov.br:11051)"
    log_info ""
    log_info "Cross-channel document anchoring is handled by the backend API"
    log_info "(documents reference each other by ID across channels)"
    log_info ""
    log_info "Next steps:"
    log_info "  1. Deploy chaincode: ./scripts/deploy-chaincode.sh deploy"
    log_info "  2. Start backend: cd backend && go run cmd/api/main.go"
    log_info "=========================================="
}

case "${1:-}" in
    start)
        start_network
        ;;
    stop)
        stop_network
        ;;
    restart)
        stop_network
        sleep 2
        start_containers
        ;;
    clean)
        cleanup
        ;;
    verify)
        verify_network
        ;;
    *)
        echo "Usage: $0 {start|stop|restart|clean|verify}"
        echo ""
        echo "Commands:"
        echo "  start   - Start the network from scratch"
        echo "  stop    - Stop the network"
        echo "  restart - Restart the network containers"
        echo "  clean   - Clean up all network artifacts"
        echo "  verify  - Verify network status"
        exit 1
        ;;
esac