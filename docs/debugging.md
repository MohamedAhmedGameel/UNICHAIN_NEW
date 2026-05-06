# Debugging

---

## First: Identify Which Layer Failed

```
Client command → Peer → Chaincode container → Ledger
```

| Error location | How to tell |
|----------------|-------------|
| Client/env | Error before any INFO lines appear |
| Peer | `endorsement failure` or `connection refused` |
| Chaincode | `error in simulation` or `chaincode stream terminated` |
| Orderer | `failed to send` or `context deadline exceeded` |

---

## Check Container Status

```bash
docker ps -a --format "table {{.Names}}\t{{.Status}}\t{{.Image}}"
```

All three must be `Up`:
- `peer0.org1.university.com`
- `orderer.university.com`
- `ca.org1.university.com`

Check chaincode container (created automatically after first invoke):

```bash
docker ps -a | grep "dev-peer"
# Should be "Up X minutes" not "Exited"
```

---

## Read Peer Logs

```bash
# Last 50 lines
docker logs peer0.org1.university.com 2>&1 | tail -50

# Filter for errors only
docker logs peer0.org1.university.com 2>&1 | grep -i "error\|panic\|fail\|warn" | tail -30

# Follow live
docker logs -f peer0.org1.university.com 2>&1
```

---

## Read Chaincode Container Logs

This is the most useful log for runtime errors. The chaincode process logs panics and fmt.Println output here.

```bash
# Get the container name
CC=$(docker ps -a --format '{{.Names}}' | grep "dev-peer.*university")
echo "Container: $CC"

# Read logs
docker logs $CC 2>&1 | tail -50

# Filter for panics
docker logs $CC 2>&1 | grep -i "panic\|error\|fatal"
```

---

## Read Orderer Logs

```bash
docker logs orderer.university.com 2>&1 | tail -30
docker logs orderer.university.com 2>&1 | grep -i "error\|fail" | tail -20
```

---

## Check Installed Chaincode

```bash
peer lifecycle chaincode queryinstalled
```

Shows all installed packages and their labels/IDs.

---

## Check Committed Chaincode

```bash
peer lifecycle chaincode querycommitted \
  --channelID universitychannel --name university
```

Shows: version, sequence, init-required status, endorsement policy.

---

## Check Approval Status

Useful when commit is failing:

```bash
peer lifecycle chaincode checkcommitreadiness \
  --channelID universitychannel \
  --name university \
  --version 1.2 \
  --sequence 3
```

Expected output:
```
Approvals: {Org1MSP: true}
```

If `false`, re-run the `approveformyorg` step.

---

## Inspect a Transaction

```bash
# Get a recent block
peer channel fetch newest /tmp/latest.block \
  -c universitychannel \
  -o localhost:7050 \
  --ordererTLSHostnameOverride orderer.university.com \
  --tls --cafile $ORDERER_CA

# Decode it (requires configtxlator)
configtxlator proto_decode \
  --input /tmp/latest.block \
  --type common.Block \
  | python3 -m json.tool | head -80
```

---

## Test Peer Connectivity

```bash
# Basic peer reachability
peer lifecycle chaincode queryinstalled

# If this errors, the peer is unreachable:
# - Check CORE_PEER_ADDRESS=localhost:7051
# - Check CORE_PEER_TLS_ROOTCERT_FILE path exists
# - Check the peer container is running
```

---

## Test Orderer Connectivity

```bash
# List channels on orderer
osnadmin channel list \
  -o localhost:7053 \
  --ca-file   organizations/ordererOrganizations/university.com/orderers/orderer.university.com/tls/ca.crt \
  --client-cert organizations/ordererOrganizations/university.com/orderers/orderer.university.com/tls/server.crt \
  --client-key  organizations/ordererOrganizations/university.com/orderers/orderer.university.com/tls/server.key
```

---

## Verify Environment Variables Are Set

```bash
echo "FABRIC_CFG_PATH:  $FABRIC_CFG_PATH"
echo "PEER_ADDRESS:     $CORE_PEER_ADDRESS"
echo "LOCALMSPID:       $CORE_PEER_LOCALMSPID"
echo "MSP_PATH exists:  $(ls $CORE_PEER_MSPCONFIGPATH 2>/dev/null && echo YES || echo NO)"
echo "TLS_CERT exists:  $(ls $CORE_PEER_TLS_ROOTCERT_FILE 2>/dev/null && echo YES || echo NO)"
echo "ORDERER_CA exists:$(ls $ORDERER_CA 2>/dev/null && echo YES || echo NO)"
echo "peer binary:      $(which peer)"
```

All should show values and files should exist.

---

## Force Restart Chaincode Container

If the chaincode container is stuck or crashed, kill it and the peer will restart it on the next invoke:

```bash
CC=$(docker ps -a --format '{{.Names}}' | grep "dev-peer.*university")
docker rm -f $CC
# Next peer invoke will rebuild and restart it automatically
```

---

## Nuclear Option — Full Reset

If nothing works and you want to start completely fresh:

```bash
cd ~/hyperledger/network

# Stop everything
docker compose down -v

# Remove chaincode containers and images
docker rm -f $(docker ps -aq --filter "name=dev-peer") 2>/dev/null || true
docker rmi $(docker images --format '{{.Repository}}:{{.Tag}}' | grep "dev-peer") 2>/dev/null || true

# Remove generated artifacts
rm -rf organizations/peerOrganizations
rm -rf organizations/ordererOrganizations
rm -rf channel-artifacts/
```

Then follow [start-from-scratch.md](./start-from-scratch.md).
