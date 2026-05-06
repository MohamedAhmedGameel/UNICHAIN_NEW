# Start From Scratch

**First time only.** Run this when you have no network, no crypto material, no channel.  
If you've already done this and just want to restart, see [start-second.md](./start-second.md).

---

## Before You Begin

Make sure you completed [windows-wsl-setup.md](./windows-wsl-setup.md) first.

Verify Docker Desktop is running on Windows, then check inside WSL2:

```bash
docker ps
# Must return a table, not an error
```

---

## Step 1 — Verify Go Toolchain

```bash
which go
# Must be /usr/local/go/bin/go  — NOT /mnt/c/...

go version
# Must be go1.21.x linux/amd64
```

If it shows a Windows path, go back to [windows-wsl-setup.md](./windows-wsl-setup.md) Step 4.

---

## Step 2 — Fix Line Endings

Always do this after copying from Windows:

```bash
find ~/hyperledger/chaincode -name "*.go" -exec sed -i 's/\r//' {} +
find ~/hyperledger/network -name "*.yaml" -exec sed -i 's/\r//' {} +
find ~/hyperledger/network -name "*.sh" -exec sed -i 's/\r//' {} +
```

---

## Step 3 — Build and Verify Chaincode Compiles

```bash
cd ~/hyperledger/chaincode/university
rm -rf vendor/
go mod tidy
go mod vendor
go build ./...
# Zero output = success. Any output = fix the error before continuing.
```

---

## Step 4 — Generate Crypto Material

```bash
cd ~/hyperledger/network

docker run --rm \
  -v "$(pwd):/workspace" -w /workspace \
  hyperledger/fabric-tools:2.5 \
  cryptogen generate \
    --config=crypto-config.yaml \
    --output=organizations

sudo chown -R "$USER:$USER" ~/hyperledger/network/organizations
```

Verify:

```bash
ls organizations/peerOrganizations/org1.university.com/peers/
# peer0.org1.university.com/
ls organizations/ordererOrganizations/university.com/orderers/
# orderer.university.com/
```

---

## Step 5 — Generate Channel Genesis Block

```bash
cd ~/hyperledger/network
mkdir -p channel-artifacts

docker run --rm \
  -v "$(pwd):/workspace" -w /workspace \
  -e FABRIC_CFG_PATH=/workspace \
  hyperledger/fabric-tools:2.5 \
  configtxgen \
    -profile UniversityChannel \
    -outputBlock "./channel-artifacts/universitychannel.block" \
    -channelID universitychannel

ls channel-artifacts/
# universitychannel.block  ✓
```

---

## Step 6 — Start Docker Containers

```bash
cd ~/hyperledger/network
docker compose up -d

echo "Waiting for network to stabilize..."
sleep 20

docker ps --format "table {{.Names}}\t{{.Status}}"
```

Expected output:

```
NAMES                           STATUS
peer0.org1.university.com       Up 20 seconds
orderer.university.com          Up 20 seconds
ca.org1.university.com          Up 20 seconds
```

If any container is not `Up`, check its logs:

```bash
docker logs peer0.org1.university.com 2>&1 | tail -20
```

---

## Step 7 — Set Environment Variables

```bash
export PATH=$PATH:$HOME/hyperledger/network/fabric-samples/bin
export FABRIC_CFG_PATH=$HOME/hyperledger/network/fabric-samples/config
export CORE_PEER_TLS_ENABLED=true
export CORE_PEER_LOCALMSPID="Org1MSP"
export CORE_PEER_ADDRESS=localhost:7051
export CORE_PEER_TLS_ROOTCERT_FILE=$HOME/hyperledger/network/organizations/peerOrganizations/org1.university.com/peers/peer0.org1.university.com/tls/ca.crt
export CORE_PEER_MSPCONFIGPATH=$HOME/hyperledger/network/organizations/peerOrganizations/org1.university.com/users/Admin@org1.university.com/msp
export ORDERER_CA=$HOME/hyperledger/network/organizations/ordererOrganizations/university.com/orderers/orderer.university.com/msp/tlscacerts/tlsca.university.com-cert.pem
```

---

## Step 8 — Join Orderer to Channel

```bash
osnadmin channel join \
  --channelID universitychannel \
  --config-block ./channel-artifacts/universitychannel.block \
  -o localhost:7053 \
  --ca-file   organizations/ordererOrganizations/university.com/orderers/orderer.university.com/tls/ca.crt \
  --client-cert organizations/ordererOrganizations/university.com/orderers/orderer.university.com/tls/server.crt \
  --client-key  organizations/ordererOrganizations/university.com/orderers/orderer.university.com/tls/server.key

sleep 5
```

---

## Step 9 — Join Peer to Channel

```bash
peer channel join -b ./channel-artifacts/universitychannel.block
sleep 3

# Verify
peer channel list
# universitychannel  ✓
```

---

## Step 10 — Package Chaincode

```bash
peer lifecycle chaincode package /tmp/university.tar.gz \
  --path $HOME/hyperledger/chaincode/university \
  --lang golang \
  --label university_1.0
```

---

## Step 11 — Install Chaincode

```bash
peer lifecycle chaincode install /tmp/university.tar.gz
sleep 5

peer lifecycle chaincode queryinstalled
# Should show university_1.0 package
```

---

## Step 12 — Get Package ID

```bash
PKG_ID=$(peer lifecycle chaincode queryinstalled --output json \
  | python3 -c "
import sys, json
pkgs = json.load(sys.stdin)['installed_chaincodes']
print([p['package_id'] for p in pkgs if p['label']=='university_1.0'][0])
")
echo "Package ID: $PKG_ID"
```

---

## Step 13 — Approve Chaincode

```bash
peer lifecycle chaincode approveformyorg \
  -o localhost:7050 \
  --ordererTLSHostnameOverride orderer.university.com \
  --tls --cafile $ORDERER_CA \
  --channelID universitychannel \
  --name university \
  --version 1.0 \
  --package-id $PKG_ID \
  --sequence 1
```

---

## Step 14 — Commit Chaincode

```bash
peer lifecycle chaincode commit \
  -o localhost:7050 \
  --ordererTLSHostnameOverride orderer.university.com \
  --tls --cafile $ORDERER_CA \
  --channelID universitychannel \
  --name university \
  --version 1.0 \
  --sequence 1 \
  --peerAddresses localhost:7051 \
  --tlsRootCertFiles $CORE_PEER_TLS_ROOTCERT_FILE

sleep 5
```

---

## Step 15 — Initialize the Ledger

Seeds default roles (SuperAdmin, Registrar, Instructor) and permissions.  
**Run this exactly once. Never run it again after upgrades.**

```bash
peer chaincode invoke \
  -o localhost:7050 \
  --ordererTLSHostnameOverride orderer.university.com \
  --tls --cafile $ORDERER_CA \
  -C universitychannel \
  -n university \
  --isInit \
  -c '{"function":"InitLedger","Args":[]}' \
  --peerAddresses localhost:7051 \
  --tlsRootCertFiles $CORE_PEER_TLS_ROOTCERT_FILE
```

---

## Step 16 — Smoke Test

```bash
sleep 3

peer chaincode query -C universitychannel -n university \
  -c '{"function":"GetAllRoles","Args":[]}'
```

Expected:

```json
[
  {"name":"Instructor","description":"Grade own courses only","active":true},
  {"name":"Registrar","description":"Student, course, and enrollment management","active":true},
  {"name":"SuperAdmin","description":"Full system access — all permissions","active":true}
]
```

**If you see this — the network is fully operational.**

---

## What Just Happened

| Step | What was created |
|------|-----------------|
| 4 | X.509 certificates for peer, orderer, admin, users |
| 5 | The genesis block that defines the channel |
| 6 | 3 Docker containers: peer, orderer, CA |
| 8–9 | Channel `universitychannel` created and joined |
| 10–14 | Chaincode packaged, installed, approved, committed |
| 15 | Ledger seeded with RBAC defaults, caller made SuperAdmin |

---

## Next Steps

- See [contract-reference.md](./contract-reference.md) to start invoking functions
- See [rbac.md](./rbac.md) to assign roles to other users
- See [redeploy.md](./redeploy.md) when you change chaincode code
