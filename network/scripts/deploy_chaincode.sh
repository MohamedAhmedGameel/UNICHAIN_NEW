#!/bin/bash
# deploy_chaincode.sh — Portable version using Docker CLI
set -euo pipefail

CHANNEL_NAME="universitychannel"
CC_NAME="university"
CC_VERSION="1.0"
CC_SEQUENCE=1

echo "═══════════════════════════════════════════════════════"
echo "  Deploy Chaincode: $CC_NAME v$CC_VERSION (Portable)"
echo "═══════════════════════════════════════════════════════"

# ── Step 1: Package ──────────────────────────────────────────────────────────
echo ""
echo "→ Step 1: Packaging chaincode..."
docker exec cli peer lifecycle chaincode package "${CC_NAME}.tar.gz" \
    --path "/opt/gopath/src/github.com/hyperledger/fabric/peer/chaincode/university" \
    --lang golang \
    --label "${CC_NAME}_${CC_VERSION}"
echo "  ✓ Packaged"

# ── Step 2: Install ──────────────────────────────────────────────────────────
echo ""
echo "→ Step 2: Installing chaincode on peer..."
docker exec cli peer lifecycle chaincode install "${CC_NAME}.tar.gz"
echo "  ✓ Installed"

# ── Step 3: Get package ID ───────────────────────────────────────────────────
echo ""
echo "→ Step 3: Querying package ID..."
PACKAGE_ID=$(docker exec cli peer lifecycle chaincode queryinstalled --output json | \
    python3 -c "import sys,json; pkgs=json.load(sys.stdin)['installed_chaincodes']; print([p['package_id'] for p in pkgs if p['label']=='${CC_NAME}_${CC_VERSION}'][0])")
echo "  Package ID: $PACKAGE_ID"

# ── Step 4: Approve ──────────────────────────────────────────────────────────
echo ""
echo "→ Step 4: Approving chaincode..."
docker exec cli peer lifecycle chaincode approveformyorg \
    -o orderer.university.com:7050 \
    --ordererTLSHostnameOverride orderer.university.com \
    --tls \
    --cafile "/opt/gopath/src/github.com/hyperledger/fabric/peer/organizations/ordererOrganizations/university.com/orderers/orderer.university.com/msp/tlscacerts/tlsca.university.com-cert.pem" \
    --channelID "$CHANNEL_NAME" \
    --name "$CC_NAME" \
    --version "$CC_VERSION" \
    --package-id "$PACKAGE_ID" \
    --sequence $CC_SEQUENCE \
    --init-required
echo "  ✓ Approved"

# ── Step 5: Commit ───────────────────────────────────────────────────────────
echo ""
echo "→ Step 5: Committing chaincode..."
docker exec cli peer lifecycle chaincode commit \
    -o orderer.university.com:7050 \
    --ordererTLSHostnameOverride orderer.university.com \
    --tls \
    --cafile "/opt/gopath/src/github.com/hyperledger/fabric/peer/organizations/ordererOrganizations/university.com/orderers/orderer.university.com/msp/tlscacerts/tlsca.university.com-cert.pem" \
    --channelID "$CHANNEL_NAME" \
    --name "$CC_NAME" \
    --version "$CC_VERSION" \
    --sequence $CC_SEQUENCE \
    --init-required \
    --peerAddresses peer0.org1.university.com:7051 \
    --tlsRootCertFiles "/opt/gopath/src/github.com/hyperledger/fabric/peer/organizations/peerOrganizations/org1.university.com/peers/peer0.org1.university.com/tls/ca.crt"
echo "  ✓ Committed"

# ── Step 6: Init ─────────────────────────────────────────────────────────────
echo ""
echo "→ Step 6: Invoking InitLedger..."
docker exec cli peer chaincode invoke \
    -o orderer.university.com:7050 \
    --ordererTLSHostnameOverride orderer.university.com \
    --tls \
    --cafile "/opt/gopath/src/github.com/hyperledger/fabric/peer/organizations/ordererOrganizations/university.com/orderers/orderer.university.com/msp/tlscacerts/tlsca.university.com-cert.pem" \
    -C "$CHANNEL_NAME" \
    -n "$CC_NAME" \
    --isInit \
    -c '{"function":"InitLedger","Args":[]}' \
    --peerAddresses peer0.org1.university.com:7051 \
    --tlsRootCertFiles "/opt/gopath/src/github.com/hyperledger/fabric/peer/organizations/peerOrganizations/org1.university.com/peers/peer0.org1.university.com/tls/ca.crt"
echo "  ✓ InitLedger complete"

echo ""
echo "═══════════════════════════════════════════════════════"
echo "  ✓ Chaincode deployed successfully!"
echo "═══════════════════════════════════════════════════════"
