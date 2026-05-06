# Redeploy — Upgrading the Chaincode

When you change chaincode code, you must go through the full package → install → approve → commit cycle again with an incremented sequence number.

**Your ledger data is preserved across upgrades.** You do NOT re-run `InitLedger`.

---

## Check Current Sequence

Always check before redeploying:

```bash
peer lifecycle chaincode querycommitted \
  --channelID universitychannel --name university
```

Note the `Sequence` value. Your next deploy must use `Sequence + 1`.

---

## Step 1 — Make Your Code Changes

Edit files inside WSL2. If editing on Windows, copy to WSL2 first:

```bash
cp /mnt/c/Users/YourName/path/to/chaincode.go \
   ~/hyperledger/chaincode/university/chaincode.go

# Fix line endings
sed -i 's/\r//' ~/hyperledger/chaincode/university/chaincode.go
```

---

## Step 2 — Verify It Compiles

**Never skip this step.**

```bash
cd ~/hyperledger/chaincode/university
go build ./...
# Zero output = good. Any output = fix the error before continuing.
```

---

## Step 3 — Set Variables

```bash
# Set the new version and sequence
NEW_VERSION="1.3"     # increment the version label
NEW_SEQUENCE=4        # current sequence + 1

# Set env vars if not already set
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

## Step 4 — Package

```bash
peer lifecycle chaincode package /tmp/university_${NEW_VERSION}.tar.gz \
  --path $HOME/hyperledger/chaincode/university \
  --lang golang \
  --label university_${NEW_VERSION}
```

---

## Step 5 — Install

```bash
peer lifecycle chaincode install /tmp/university_${NEW_VERSION}.tar.gz
sleep 5
```

---

## Step 6 — Get Package ID

```bash
PKG_ID=$(peer lifecycle chaincode queryinstalled --output json \
  | python3 -c "
import sys, json
pkgs = json.load(sys.stdin)['installed_chaincodes']
label = 'university_${NEW_VERSION}'
print([p['package_id'] for p in pkgs if p['label'] == label][0])
")
echo "Package ID: $PKG_ID"
```

---

## Step 7 — Approve

```bash
peer lifecycle chaincode approveformyorg \
  -o localhost:7050 \
  --ordererTLSHostnameOverride orderer.university.com \
  --tls --cafile $ORDERER_CA \
  --channelID universitychannel \
  --name university \
  --version ${NEW_VERSION} \
  --package-id $PKG_ID \
  --sequence ${NEW_SEQUENCE}
```

---

## Step 8 — Commit

```bash
peer lifecycle chaincode commit \
  -o localhost:7050 \
  --ordererTLSHostnameOverride orderer.university.com \
  --tls --cafile $ORDERER_CA \
  --channelID universitychannel \
  --name university \
  --version ${NEW_VERSION} \
  --sequence ${NEW_SEQUENCE} \
  --peerAddresses localhost:7051 \
  --tlsRootCertFiles $CORE_PEER_TLS_ROOTCERT_FILE
```

---

## Step 9 — Verify

```bash
sleep 5

# Check new version is committed
peer lifecycle chaincode querycommitted \
  --channelID universitychannel --name university

# Test a query to confirm chaincode responds
peer chaincode query -C universitychannel -n university \
  -c '{"function":"GetAllRoles","Args":[]}'
```

---

## Sequence Tracking

Keep a record so you never lose track:

| Deploy | Version | Sequence | Notes |
|--------|---------|----------|-------|
| Initial deploy | 1.0 | 1 | With --init-required |
| UTF-8 fix | 1.1 | 2 | Fixed \xff range keys |
| Full state wiring | 1.2 | 3 | Added all contract methods |
| Your next change | 1.3 | 4 | |

---

## Important Rules

- **Never reuse a sequence number** — it will be rejected
- **Never re-run InitLedger** after an upgrade — it will attempt to recreate already-existing state
- **Data persists** — all majors, students, courses, enrollments survive upgrades
- **`--init-required` is gone** after sequence 1 — do not add it to upgrade deploys
- If you see `committed with status (INVALID)`, check the sequence number and approve/commit again with the correct one
