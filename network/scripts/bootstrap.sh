#!/bin/bash
# bootstrap.sh — Portable version using Docker tools
set -euo pipefail

CHANNEL_NAME="universitychannel"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
NETWORK_DIR="$(dirname "$SCRIPT_DIR")"

cd "$NETWORK_DIR"

echo "═══════════════════════════════════════════════════════"
echo "  UNICHAIN Fabric Network Bootstrap (Portable)"
echo "═══════════════════════════════════════════════════════"

# ── Step 1: Generate crypto material ─────────────────────────────────────────
echo ""
echo "→ Step 1: Generating crypto material..."
if [ -d "organizations/peerOrganizations" ]; then
    echo "  ↩ Crypto material already exists — skipping"
else
    # We mount the current directory to /workspace inside the container to avoid Windows path issues
    docker run --rm -v "$(pwd):/workspace" -w /workspace hyperledger/fabric-tools:2.5 cryptogen generate --config=crypto-config.yaml --output=organizations
    echo "  ✓ Crypto material generated"
fi

# ── Step 2: Generate channel artifacts ───────────────────────────────────────
echo ""
echo "→ Step 2: Generating channel artifacts..."
mkdir -p channel-artifacts

docker run --rm -v "$(pwd):/workspace" -w /workspace -e FABRIC_CFG_PATH=/workspace hyperledger/fabric-tools:2.5 configtxgen -profile UniversityChannel \
    -outputBlock "./channel-artifacts/${CHANNEL_NAME}.block" \
    -channelID "$CHANNEL_NAME"
echo "  ✓ Genesis block created"

# ── Step 3: Start Docker containers ──────────────────────────────────────────
echo ""
echo "→ Step 3: Starting Docker containers..."
docker-compose up -d
echo "  ✓ Containers started"

# Wait for peer and orderer to be ready
echo "  Waiting for network to stabilize (15s)..."
sleep 15

# ── Step 4: Join channel ─────────────────────────────────────────────────────
echo ""
echo "→ Step 4: Joining peer to channel..."

# Join orderer via docker exec
osnadmin channel join \
    --channelID "$CHANNEL_NAME" \
    --config-block "./channel-artifacts/${CHANNEL_NAME}.block" \
    -o localhost:7053 \
    --ca-file "./organizations/ordererOrganizations/university.com/orderers/orderer.university.com/tls/ca.crt" \
    --client-cert "./organizations/ordererOrganizations/university.com/orderers/orderer.university.com/tls/server.crt" \
    --client-key "./organizations/ordererOrganizations/university.com/orderers/orderer.university.com/tls/server.key"
echo "  ✓ Channel created on orderer"

# Join peer via cli container
docker exec cli peer channel join -b "./channel-artifacts/${CHANNEL_NAME}.block"
echo "  ✓ Peer joined channel"

echo ""
echo "═══════════════════════════════════════════════════════"
echo "  ✓ Network bootstrap complete!"
echo "═══════════════════════════════════════════════════════"
