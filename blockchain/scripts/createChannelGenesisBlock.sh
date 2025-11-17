#!/bin/bash

CHANNEL_NAME="$1"
PROFILE="$2"

if [ -z "$CHANNEL_NAME" ]; then
  echo "Usage: createChannelGenesisBlock.sh <channel-name> <profile>"
  exit 1
fi

if [ -z "$PROFILE" ]; then
  PROFILE="UnionChannel"
fi

echo "Creating genesis block for channel: $CHANNEL_NAME with profile: $PROFILE"

# Ensure we have the crypto material
if [ ! -d "organizations/ordererOrganizations/union.gov" ]; then
  echo "Error: Orderer crypto material not found. Please run './network.sh up' first."
  exit 1
fi

if [ ! -d "organizations/peerOrganizations/union.gov" ]; then
  echo "Error: Peer crypto material not found. Please run './network.sh up' first."
  exit 1
fi

# Create channel-artifacts directory if it doesn't exist
mkdir -p channel-artifacts

# Generate the genesis block
configtxgen -profile $PROFILE \
  -outputBlock ./channel-artifacts/${CHANNEL_NAME}.block \
  -channelID $CHANNEL_NAME \
  -configPath ${PWD}/configtx

if [ $? -ne 0 ]; then
  echo "Error: Failed to generate genesis block"
  exit 1
fi

echo "Genesis block created successfully at channel-artifacts/${CHANNEL_NAME}.block"