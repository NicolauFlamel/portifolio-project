#!/bin/bash

# Network management script for Hyperledger Fabric

export PATH=${PWD}/bin:$PATH

CHANNEL_NAME="transfer-channel"
DELAY=3
MAX_RETRY=5

# Print helper text
function printHelp() {
  echo "Usage: "
  echo "  network.sh <Mode>"
  echo "    <Mode>"
  echo "      - 'up' - bring up the network"
  echo "      - 'down' - clear the network"
  echo "      - 'restart' - restart the network"
  echo "      - 'createChannel' - create and join channel"
  echo
  echo "  network.sh -h (print this message)"
}

# Generate crypto material using cryptogen
function generateCryptoMaterial() {
  which cryptogen
  if [ "$?" -ne 0 ]; then
    echo "cryptogen tool not found. Downloading Fabric binaries..."
    curl -sSL https://bit.ly/2ysbOFE | bash -s -- 2.5.4 1.5.7 -d -s
  fi

  echo "##########################################################"
  echo "##### Generate certificates using cryptogen tool #########"
  echo "##########################################################"

  if [ -d "organizations/peerOrganizations" ]; then
    rm -Rf organizations/peerOrganizations
  fi
  if [ -d "organizations/ordererOrganizations" ]; then
    rm -Rf organizations/ordererOrganizations
  fi

  mkdir -p organizations

  cryptogen generate --config=./crypto-config.yaml --output="organizations"

  if [ "$?" -ne 0 ]; then
    echo "Failed to generate crypto material..."
    exit 1
  fi

  # Ensure MSP directories exist
  mkdir -p organizations/ordererOrganizations/union.gov/msp/tlscacerts
  mkdir -p organizations/peerOrganizations/union.gov/msp/tlscacerts

  # Copy TLS CA certificates to MSP
  echo "Setting up MSP TLS CA certificates..."
  cp organizations/ordererOrganizations/union.gov/tlsca/tlsca.union.gov-cert.pem \
     organizations/ordererOrganizations/union.gov/msp/tlscacerts/

  cp organizations/peerOrganizations/union.gov/tlsca/tlsca.union.gov-cert.pem \
     organizations/peerOrganizations/union.gov/msp/tlscacerts/

  # Create config.yaml for orderer MSP to enable NodeOUs
  echo "NodeOUs:
  Enable: true
  ClientOUIdentifier:
    Certificate: cacerts/ca.union.gov-cert.pem
    OrganizationalUnitIdentifier: client
  PeerOUIdentifier:
    Certificate: cacerts/ca.union.gov-cert.pem
    OrganizationalUnitIdentifier: peer
  AdminOUIdentifier:
    Certificate: cacerts/ca.union.gov-cert.pem
    OrganizationalUnitIdentifier: admin
  OrdererOUIdentifier:
    Certificate: cacerts/ca.union.gov-cert.pem
    OrganizationalUnitIdentifier: orderer" > organizations/ordererOrganizations/union.gov/msp/config.yaml

  # Create config.yaml for peer MSP to enable NodeOUs
  echo "NodeOUs:
  Enable: true
  ClientOUIdentifier:
    Certificate: cacerts/ca.union.gov-cert.pem
    OrganizationalUnitIdentifier: client
  PeerOUIdentifier:
    Certificate: cacerts/ca.union.gov-cert.pem
    OrganizationalUnitIdentifier: peer
  AdminOUIdentifier:
    Certificate: cacerts/ca.union.gov-cert.pem
    OrganizationalUnitIdentifier: admin
  OrdererOUIdentifier:
    Certificate: cacerts/ca.union.gov-cert.pem
    OrganizationalUnitIdentifier: orderer" > organizations/peerOrganizations/union.gov/msp/config.yaml

  echo "Generate crypto material completed."
  
  # Show what was created
  echo ""
  echo "Generated MSP structure:"
  ls -la organizations/ordererOrganizations/union.gov/
  ls -la organizations/peerOrganizations/union.gov/
}

# Generate channel configuration block
function generateChannelArtifacts() {
  which configtxgen
  if [ "$?" -ne 0 ]; then
    echo "configtxgen tool not found. Please install Fabric binaries."
    exit 1
  fi

  echo "##########################################################"
  echo "#########  Generating Channel Artifacts  #################"
  echo "##########################################################"

  mkdir -p channel-artifacts

  echo "Generating channel configuration block..."
  
  # Generate from the project root where organizations folder is
  cd ${PWD}
  
  configtxgen -profile UnionChannel \
    -outputBlock ./channel-artifacts/${CHANNEL_NAME}.block \
    -channelID $CHANNEL_NAME \
    -configPath ./configtx

  if [ "$?" -ne 0 ]; then
    echo "Failed to generate channel configuration block..."
    exit 1
  fi

  echo "Channel artifacts generated successfully."
  echo "Genesis block location: ${PWD}/channel-artifacts/${CHANNEL_NAME}.block"
}

# Start the network
function networkUp() {
  if [ ! -d "organizations/peerOrganizations" ]; then
    generateCryptoMaterial
  fi

  # Generate channel artifacts AFTER crypto material exists
  if [ ! -f "channel-artifacts/${CHANNEL_NAME}.block" ]; then
    generateChannelArtifacts
  fi

  docker-compose up -d

  if [ $? -ne 0 ]; then
    echo "ERROR: Failed to start the network"
    exit 1
  fi

  echo "Waiting for containers to start..."
  sleep 5

  echo "Network started successfully."
}

# Stop and clean the network
function networkDown() {
  docker-compose down --volumes --remove-orphans
  
  # Cleanup
  rm -rf organizations/peerOrganizations
  rm -rf organizations/ordererOrganizations
  rm -rf channel-artifacts
  rm -rf organizations/fabric-ca

  # Remove chaincode containers and images
  docker rm -f $(docker ps -aq --filter "name=dev-peer") 2>/dev/null || true
  docker rmi -f $(docker images -q "dev-peer*") 2>/dev/null || true

  echo "Network stopped and cleaned."
}

# Create and join channel
function createChannel() {
  echo "##########################################################"
  echo "############### Creating Channel #########################"
  echo "##########################################################"

  # Use osnadmin CLI to join orderer to channel
  docker exec cli osnadmin channel join \
    --channelID $CHANNEL_NAME \
    --config-block /opt/gopath/src/github.com/hyperledger/fabric/peer/channel-artifacts/${CHANNEL_NAME}.block \
    -o orderer.union.gov:7053 \
    --ca-file /opt/gopath/src/github.com/hyperledger/fabric/peer/organizations/ordererOrganizations/union.gov/orderers/orderer.union.gov/msp/tlscacerts/tlsca.union.gov-cert.pem \
    --client-cert /opt/gopath/src/github.com/hyperledger/fabric/peer/organizations/ordererOrganizations/union.gov/orderers/orderer.union.gov/tls/server.crt \
    --client-key /opt/gopath/src/github.com/hyperledger/fabric/peer/organizations/ordererOrganizations/union.gov/orderers/orderer.union.gov/tls/server.key

  if [ $? -ne 0 ]; then
    echo "ERROR: Failed to join orderer to channel"
    exit 1
  fi

  echo "Orderer joined channel successfully."
  sleep 2

  # List channels on orderer to verify
  echo "Listing channels on orderer..."
  docker exec cli osnadmin channel list \
    -o orderer.union.gov:7053 \
    --ca-file /opt/gopath/src/github.com/hyperledger/fabric/peer/organizations/ordererOrganizations/union.gov/orderers/orderer.union.gov/msp/tlscacerts/tlsca.union.gov-cert.pem \
    --client-cert /opt/gopath/src/github.com/hyperledger/fabric/peer/organizations/ordererOrganizations/union.gov/orderers/orderer.union.gov/tls/server.crt \
    --client-key /opt/gopath/src/github.com/hyperledger/fabric/peer/organizations/ordererOrganizations/union.gov/orderers/orderer.union.gov/tls/server.key

  sleep 3

  echo "##########################################################"
  echo "############### Joining Peer to Channel ##################"
  echo "##########################################################"

  docker exec cli peer channel join \
    -b /opt/gopath/src/github.com/hyperledger/fabric/peer/channel-artifacts/${CHANNEL_NAME}.block

  if [ $? -ne 0 ]; then
    echo "ERROR: Failed to join peer to channel"
    exit 1
  fi

  echo "Peer joined channel successfully."
  sleep 2

  echo "##########################################################"
  echo "############ Setting Anchor Peer ########################"
  echo "##########################################################"

  docker exec cli peer channel fetch config /opt/gopath/src/github.com/hyperledger/fabric/peer/channel-artifacts/config_block.pb \
    -o orderer.union.gov:7050 \
    -c $CHANNEL_NAME \
    --tls --cafile /opt/gopath/src/github.com/hyperledger/fabric/peer/organizations/ordererOrganizations/union.gov/orderers/orderer.union.gov/msp/tlscacerts/tlsca.union.gov-cert.pem

  # This is already set in the genesis block, so we just verify
  echo "Anchor peer configuration included in genesis block."
}

# Parse commandline args
if [[ $# -lt 1 ]] ; then
  printHelp
  exit 0
else
  MODE=$1
  shift
fi

# Determine mode
if [ "${MODE}" == "up" ]; then
  networkUp
elif [ "${MODE}" == "down" ]; then
  networkDown
elif [ "${MODE}" == "restart" ]; then
  networkDown
  networkUp
elif [ "${MODE}" == "createChannel" ]; then
  createChannel
else
  printHelp
  exit 1
fi