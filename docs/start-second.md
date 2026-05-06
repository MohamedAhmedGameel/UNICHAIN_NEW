# Second Start

**Every time after the first setup.** Use this when the network already exists  
(crypto material and channel-artifacts are already generated) and you just need to start it back up.

If this is your first time, go to [start-from-scratch.md](./start-from-scratch.md) instead.

---

## Quick Check — Which Situation Are You In?

```bash
# Check 1: Does crypto material exist?
ls ~/hyperledger/network/organizations/peerOrganizations/ 2>/dev/null && echo "EXISTS" || echo "MISSING"

# Check 2: Does the genesis block exist?
ls ~/hyperledger/network/channel-artifacts/universitychannel.block 2>/dev/null && echo "EXISTS" || echo "MISSING"

# Check 3: Are containers already running?
docker ps --format "{{.Names}}" | grep -E "peer|orderer|ca"
```

| Result | Action |
|--------|--------|
| Both MISSING | Go to [start-from-scratch.md](./start-from-scratch.md) |
| Both EXIST, containers running | Just set env vars → jump to [Step 3](#step3) |
| Both EXIST, containers stopped | Start from [Step 1](#step1) below |
| One MISSING | Something is corrupted — go to [start-from-scratch.md](./start-from-scratch.md) |

---

## Step 1 — Open Docker Desktop on Windows <a name="step1"></a>

Before touching WSL2, make sure Docker Desktop is running on Windows.  
Look for the Docker whale icon in the system tray. If it's not there, launch Docker Desktop from the Start Menu and wait for it to say "Engine running".

---

## Step 2 — Start the Containers

Open WSL2 terminal and run:

```bash
cd ~/hyperledger/network
docker compose up -d

echo "Waiting for containers..."
sleep 15

docker ps --format "table {{.Names}}\t{{.Status}}"
```

Expected:

```
NAMES                           STATUS
peer0.org1.university.com       Up 15 seconds
orderer.university.com          Up 15 seconds
ca.org1.university.com          Up 15 seconds
```

If any container fails to start, check its logs:

```bash
docker logs peer0.org1.university.com 2>&1 | tail -30
docker logs orderer.university.com 2>&1 | tail -30
```

---

## Step 3 — Set Environment Variables <a name="step3"></a>

**Required every new terminal session** unless you added them to `~/.bashrc`.

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

> **Tip:** Add these to `~/.bashrc` once so you never have to paste them again:
> ```bash
> nano ~/.bashrc   # paste the exports at the bottom, save
> source ~/.bashrc
> ```

---

## Step 4 — Verify the Network is Ready

```bash
# Check peer is reachable
peer lifecycle chaincode queryinstalled
# Should list installed chaincode packages

# Check chaincode is committed on channel
peer lifecycle chaincode querycommitted \
  --channelID universitychannel --name university
# Should show: university, version 1.2, sequence 3 (or your latest)
```

---

## Step 5 — Verify Chaincode is Responding

```bash
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

**If you see this — you're fully connected and ready to use the system.**

---

## Step 6 — Resume Working

Your ledger state is fully preserved. Everything you created before (majors, students, courses, enrollments) is still there.

```bash
# Check what's on the ledger
peer chaincode query -C universitychannel -n university -c '{"function":"GetAllMajors","Args":[]}'
peer chaincode query -C universitychannel -n university -c '{"function":"GetAllStudents","Args":[]}'
peer chaincode query -C universitychannel -n university -c '{"function":"GetAllCourses","Args":[]}'
```

See [contract-reference.md](./contract-reference.md) for all available commands.

---

## Stopping the Network

When you're done for the day:

```bash
cd ~/hyperledger/network
docker compose down
```

This stops containers but **preserves all ledger data**. Next time just do Steps 1–5 above.

---

## Wiping Everything and Starting Over

Only do this if you want to reset the ledger to zero:

```bash
cd ~/hyperledger/network

# Stop containers and delete volumes (ledger data)
docker compose down -v

# Remove generated crypto and channel artifacts
rm -rf organizations/peerOrganizations
rm -rf organizations/ordererOrganizations
rm -rf channel-artifacts/

# Remove old chaincode containers and images
docker rm -f $(docker ps -aq --filter "name=dev-peer") 2>/dev/null || true
docker rmi $(docker images --format '{{.Repository}}:{{.Tag}}' | grep "dev-peer") 2>/dev/null || true
```

Then follow [start-from-scratch.md](./start-from-scratch.md) from the beginning.

---

## Troubleshooting Second Start

### Chaincode container not starting

```bash
# Check if it's crashed
docker ps -a | grep "dev-peer"

# Read its logs
docker logs $(docker ps -a --format '{{.Names}}' | grep "dev-peer.*university") 2>&1 | tail -40
```

### `connection refused` on peer commands

The peer container isn't up yet. Wait a few more seconds after `docker compose up -d` and retry.

### `context deadline exceeded`

TLS cert path is wrong or container is unreachable. Re-run the env var block from Step 3.

### State is empty after restart

This should not happen with `docker compose down` (without `-v`). If it does, the Docker volume was deleted. You'll need to go through [start-from-scratch.md](./start-from-scratch.md) and redeploy the chaincode, then reinitialize with `InitLedger`.
