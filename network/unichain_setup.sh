#!/bin/bash
# =============================================================
#  UNICHAIN — One-Shot Setup Script
#  Handles every known issue automatically
# =============================================================
set -euo pipefail

NETWORK_DIR="$HOME/hyperledger/network"
CHANNEL_NAME="universitychannel"
CC_NAME="university"
CC_VERSION="1.0"
CC_SEQUENCE=1
PEER_ORG_DIR="$NETWORK_DIR/organizations/peerOrganizations/org1.university.com/peers/peer0.org1.university.com"
ORDERER_DIR="$NETWORK_DIR/organizations/ordererOrganizations/university.com/orderers/orderer.university.com"
CLI_BASE="/opt/gopath/src/github.com/hyperledger/fabric/peer"

print_step() { echo ""; echo "══════════════════════════════════════"; echo "  $1"; echo "══════════════════════════════════════"; }
ok()   { echo "  ✓ $1"; }
info() { echo "  → $1"; }
fail() { echo "  ✗ $1"; exit 1; }

cd "$NETWORK_DIR"

# ── 0. Pre-flight checks ─────────────────────────────────────
print_step "Step 0: Pre-flight checks"

command -v docker   >/dev/null || fail "docker not found"
command -v osnadmin >/dev/null || fail "osnadmin not found — run: source ~/.bashrc"
command -v python3  >/dev/null || fail "python3 not found"
docker ps >/dev/null 2>&1      || fail "Docker not accessible — run: newgrp docker"
ok "All prerequisites present"

# ── 1. Apply fixes to docker-compose.yml ────────────────────
print_step "Step 1: Patching docker-compose.yml"

python3 << 'EOF'
content = open("docker-compose.yml").read()
target = "      - ../chaincode:/opt/gopath/src/github.com/hyperledger/fabric/peer/chaincode"
socket = "\n      - /var/run/docker.sock:/var/run/docker.sock"
if "/var/run/docker.sock" not in content:
    content = content.replace(target, target + socket, 1)
    open("docker-compose.yml", "w").write(content)
    print("  ✓ Docker socket mount added")
else:
    print("  ✓ Docker socket mount already present")
EOF

COUNT=$(grep -c "docker.sock" docker-compose.yml)
[ "$COUNT" -eq 1 ] || fail "docker-compose.yml has $COUNT socket entries (expected 1) — restore original and rerun"

# ── 2. Generate crypto material ──────────────────────────────
print_step "Step 2: Generating crypto material"

if [ -d "organizations/peerOrganizations" ]; then
    ok "Crypto already exists — skipping"
else
    docker run --rm \
        -v "$(pwd):/workspace" -w /workspace \
        hyperledger/fabric-tools:2.5 \
        cryptogen generate --config=crypto-config.yaml --output=organizations
    ok "Crypto material generated"
fi

sudo chown -R "$USER:$USER" "$NETWORK_DIR/organizations" 2>/dev/null || true

# ── 3. Stage core.yaml for peer ──────────────────────────────
print_step "Step 3: Staging core.yaml for peer container"

mkdir -p "$PEER_ORG_DIR"
cp "$HOME/fabric/config/core.yaml" "$PEER_ORG_DIR/core.yaml"
ok "core.yaml copied"

# Patch NetworkMode in core.yaml
python3 << 'EOF'
import re, os
path = os.path.expanduser(
    "~/hyperledger/network/organizations/peerOrganizations"
    "/org1.university.com/peers/peer0.org1.university.com/core.yaml"
)
content = open(path).read()
if "university_network" in content:
    print("  ✓ NetworkMode already set")
elif "NetworkMode: host" in content:
    content = content.replace("NetworkMode: host", "NetworkMode: university_network", 1)
    open(path, "w").write(content)
    print("  ✓ NetworkMode set to university_network")
else:
    content = re.sub(
        r'(        hostConfig:)',
        r'\1\n            NetworkMode: university_network',
        content, count=1
    )
    open(path, "w").write(content)
    print("  ✓ NetworkMode inserted into hostConfig")
EOF

# ── 4. Generate channel genesis block ────────────────────────
print_step "Step 4: Generating channel genesis block"

mkdir -p channel-artifacts
docker run --rm \
    -v "$(pwd):/workspace" -w /workspace \
    -e FABRIC_CFG_PATH=/workspace \
    hyperledger/fabric-tools:2.5 \
    configtxgen -profile UniversityChannel \
        -outputBlock "./channel-artifacts/${CHANNEL_NAME}.block" \
        -channelID "$CHANNEL_NAME"
ok "Genesis block created"

# ── 5. Start Docker containers ───────────────────────────────
print_step "Step 5: Starting Docker containers"

docker compose up -d
ok "Containers started"
info "Waiting 20s for network to stabilize..."
sleep 20

# ── 6. Verify peer is running ────────────────────────────────
print_step "Step 6: Verifying peer container"

PEER_STATUS=$(docker inspect -f '{{.State.Status}}' peer0.org1.university.com 2>/dev/null || echo "missing")
if [ "$PEER_STATUS" != "running" ]; then
    echo "  ✗ Peer status: $PEER_STATUS"
    docker logs peer0.org1.university.com 2>&1 | tail -5
    fail "Peer container is not running"
fi
ok "Peer is running"

# ── 7. Fix Docker socket inside CLI ─────────────────────────
print_step "Step 7: Fixing Docker socket permissions in CLI"

docker exec -u root peer0.org1.university.com chmod 666 /var/run/docker.sock
ok "Docker socket is world-writable inside CLI"

# ── 8. Join orderer to channel (from host) ───────────────────
print_step "Step 8: Joining orderer to channel"

osnadmin channel join \
    --channelID "$CHANNEL_NAME" \
    --config-block "./channel-artifacts/${CHANNEL_NAME}.block" \
    -o localhost:7053 \
    --ca-file   "${ORDERER_DIR}/tls/ca.crt" \
    --client-cert "${ORDERER_DIR}/tls/server.crt" \
    --client-key  "${ORDERER_DIR}/tls/server.key"
ok "Orderer joined channel"
sleep 5

# ── 9. Join peer to channel ──────────────────────────────────
print_step "Step 9: Joining peer to channel"

docker exec cli peer channel join \
    -b "./channel-artifacts/${CHANNEL_NAME}.block"
ok "Peer joined channel"

# ── 10. Package chaincode ────────────────────────────────────
print_step "Step 10: Packaging chaincode"

docker exec cli peer lifecycle chaincode package "${CC_NAME}.tar.gz" \
    --path "${CLI_BASE}/chaincode/university" \
    --lang golang \
    --label "${CC_NAME}_${CC_VERSION}"
ok "Chaincode packaged"

# ── 11. Install chaincode ────────────────────────────────────
print_step "Step 11: Installing chaincode on peer"

docker exec cli peer lifecycle chaincode install "${CC_NAME}.tar.gz"
ok "Chaincode installed"
sleep 5

# ── 12. Get package ID ───────────────────────────────────────
print_step "Step 12: Querying package ID"

PACKAGE_ID=$(docker exec cli peer lifecycle chaincode queryinstalled --output json | \
    python3 -c "
import sys, json
pkgs = json.load(sys.stdin)['installed_chaincodes']
label = '${CC_NAME}_${CC_VERSION}'
match = [p['package_id'] for p in pkgs if p['label'] == label]
print(match[0])
")
ok "Package ID: $PACKAGE_ID"

# ── 13. Approve chaincode ────────────────────────────────────
print_step "Step 13: Approving chaincode"

CAFILE="${CLI_BASE}/organizations/ordererOrganizations/university.com/orderers/orderer.university.com/msp/tlscacerts/tlsca.university.com-cert.pem"

docker exec cli peer lifecycle chaincode approveformyorg \
    -o orderer.university.com:7050 \
    --ordererTLSHostnameOverride orderer.university.com \
    --tls --cafile "$CAFILE" \
    --channelID "$CHANNEL_NAME" \
    --name "$CC_NAME" \
    --version "$CC_VERSION" \
    --package-id "$PACKAGE_ID" \
    --sequence $CC_SEQUENCE \
    --init-required
ok "Chaincode approved"

# ── 14. Commit chaincode ─────────────────────────────────────
print_step "Step 14: Committing chaincode"

PEER_TLS="${CLI_BASE}/organizations/peerOrganizations/org1.university.com/peers/peer0.org1.university.com/tls/ca.crt"

docker exec cli peer lifecycle chaincode commit \
    -o orderer.university.com:7050 \
    --ordererTLSHostnameOverride orderer.university.com \
    --tls --cafile "$CAFILE" \
    --channelID "$CHANNEL_NAME" \
    --name "$CC_NAME" \
    --version "$CC_VERSION" \
    --sequence $CC_SEQUENCE \
    --init-required \
    --peerAddresses peer0.org1.university.com:7051 \
    --tlsRootCertFiles "$PEER_TLS"
ok "Chaincode committed"
sleep 5

# ── 15. InitLedger ───────────────────────────────────────────
print_step "Step 15: Initializing ledger (InitLedger)"

docker exec cli peer chaincode invoke \
    -o orderer.university.com:7050 \
    --ordererTLSHostnameOverride orderer.university.com \
    --tls --cafile "$CAFILE" \
    -C "$CHANNEL_NAME" \
    -n "$CC_NAME" \
    --isInit \
    -c '{"function":"InitLedger","Args":[]}' \
    --peerAddresses peer0.org1.university.com:7051 \
    --tlsRootCertFiles "$PEER_TLS"
ok "Ledger initialized"
sleep 3

# ── 16. Smoke test ───────────────────────────────────────────
print_step "Step 16: Smoke test — querying roles from chain"

docker exec cli peer chaincode query \
    -C "$CHANNEL_NAME" \
    -n "$CC_NAME" \
    -c '{"function":"ListRoles","Args":[]}'

echo ""
echo "══════════════════════════════════════════════════════"
echo "  ✓ UNICHAIN network is fully operational!"
echo ""
echo "  Next step — start the API:"
echo "    cd ~/hyperledger/api"
echo "    source ~/.bashrc"
echo "    pip3 install --break-system-packages -r requirements.txt"
echo "    uvicorn main:app --reload --port 8000"
echo ""
echo "  Then open in browser:"
echo "    http://localhost:8000/docs"
echo "══════════════════════════════════════════════════════"
