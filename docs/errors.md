# Common Errors & Fixes

---

## Environment & Setup Errors

---

### `FABRIC_CFG_PATH does not exist`

```
Fatal error when initializing core config: FABRIC_CFG_PATH /home/.../config does not exist
```

**Cause:** The env var isn't set, or points to a wrong path.

**Fix:**
```bash
export FABRIC_CFG_PATH=$HOME/hyperledger/network/fabric-samples/config
ls $FABRIC_CFG_PATH/core.yaml   # must exist
```

---

### `command not found: peer`

**Cause:** Fabric binaries aren't in PATH.

**Fix:**
```bash
export PATH=$PATH:$HOME/hyperledger/network/fabric-samples/bin
which peer   # must return a path
```

---

### `go.mod file not found`

```
go: go.mod file not found in current directory or any parent directory
```

**Cause:** Running Go commands from the wrong directory.

**Fix:**
```bash
cd ~/hyperledger/chaincode/university
go build ./...
```

---

### `cannot use Windows Go installation`

Go binary at `/mnt/c/...` won't compile Linux chaincode correctly.

**Fix:** Install Go inside WSL2:
```bash
sudo rm -rf /usr/local/go
wget https://go.dev/dl/go1.21.13.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.13.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc
which go   # must be /usr/local/go/bin/go
```

---

### `Docker not accessible`

```
permission denied while trying to connect to the Docker daemon socket
```

**Fix:**
```bash
# Option 1: Add yourself to docker group
sudo usermod -aG docker $USER
newgrp docker

# Option 2: Make sure Docker Desktop is running on Windows
# Look for the whale icon in the system tray
```

---

## Chaincode Install / Deploy Errors

---

### `chaincode install failed: string field contains invalid UTF-8`

```
error in simulation: transaction returned with failure:
failed to marshal response: string field contains invalid UTF-8
```

**Cause:** `\xff` byte used as the upper bound in `GetStateByRange`. Protobuf cannot serialize it.

**Fix:** In `rbac/rbac.go`, replace any `\xff` range ends:
```go
// Wrong
ctx.GetStub().GetStateByRange("ROLE~", "ROLE~\xff")

// Correct
ctx.GetStub().GetStateByRange("ROLE~", "ROLE\x7f")
```

Then redeploy (see [redeploy.md](./redeploy.md)).

---

### `committed with status (INVALID)`

**Cause 1:** Wrong sequence number.
```bash
# Check current sequence
peer lifecycle chaincode querycommitted \
  --channelID universitychannel --name university

# Re-approve and re-commit with sequence+1
```

**Cause 2:** Package ID doesn't match what was approved.
```bash
# Get actual installed package ID
peer lifecycle chaincode queryinstalled --output json

# Re-approve with the correct package ID
```

---

### `chaincode with name 'university' already exists`

The chaincode is already committed at that sequence. Increment the sequence number:
```bash
# Approve and commit with sequence+1
```

---

### `error endorsing chaincode: endorsement failure`

**Cause 1:** Chaincode container not running.
```bash
docker ps -a | grep "dev-peer"
# If Exited: check logs
docker logs $(docker ps -a --format '{{.Names}}' | grep "dev-peer.*university") 2>&1 | tail -30
```

**Cause 2:** Peer is unreachable.
```bash
# Verify address
echo $CORE_PEER_ADDRESS   # must be localhost:7051
docker ps | grep peer0    # must be Up
```

---

## Runtime Errors

---

### `chaincode stream terminated`

```
error in simulation: failed to execute transaction ...: error sending: chaincode stream terminated
```

**Cause:** The chaincode process panicked and crashed.

**Fix:** Read the chaincode container logs immediately:
```bash
docker logs $(docker ps -a --format '{{.Names}}' | grep "dev-peer.*university") 2>&1 | tail -40
```

The panic message will tell you exactly what crashed.

---

### `panic: failed to marshal message: string field contains invalid UTF-8`

Same as the install error above but happening at runtime. A `GetStateByRange` call is using `\xff`.

**Fix:** Replace all `\xff` with `\x7f` in range end keys and redeploy.

---

### `forbidden: missing permission 'X'`

```
Error: endorsement failure during invoke. response: status:500
message:"forbidden: missing permission 'major:create'"
```

**Cause:** The caller's address doesn't have the required permission.

**Fix:**
```bash
# Get your address
peer chaincode query ... -c '{"function":"GetCallerAddress","Args":[]}'

# Check what roles you have
peer chaincode query ... -c '{"function":"GetUserRoles","Args":["<your-address>"]}'

# SuperAdmin assigns you the right role
peer chaincode invoke ... -c '{"function":"AssignRoleToUser","Args":["<your-address>","Registrar"]}'
```

---

### `major X not found or inactive`

**Cause:** Trying to add a student with a `majorId` that doesn't exist or was deactivated.

**Fix:** Create the major first:
```bash
peer chaincode query ... -c '{"function":"GetAllMajors","Args":[]}'
# If empty, add one:
peer chaincode invoke ... -c '{"function":"AddMajor","Args":["CS","Computer Science",""]}'
```

---

### `professor X not found or inactive`

**Cause:** Trying to create a course with a `professorId` that doesn't exist.

**Fix:**
```bash
peer chaincode query ... -c '{"function":"GetAllProfessors","Args":[]}'
peer chaincode invoke ... -c '{"function":"AddProfessor","Args":["Dr. Smith","CS","0xaddr"]}'
```

---

### `student 'X' already enrolled in 'Y' for 'Z'`

The enrollment already exists and is active. This is expected behavior — it's idempotent protection.

---

### `course 'X' already exists`

Course IDs are unique strings (like "CS101"). Either use a different ID or delete the existing course first.

---

### `mark must be 0–100`

```
Error: mark must be 0–100, got 150
```

All marks must be integers between 0 and 100 inclusive.

---

### `no professor record found for caller address`

An Instructor trying to grade with `mark:own` but their caller address isn't registered as a professor.

**Fix:** The professor must be registered with their exact caller address:
```bash
# Instructor gets their address
peer chaincode query ... -c '{"function":"GetCallerAddress","Args":[]}'

# Admin registers them as professor using that exact address
peer chaincode invoke ... -c '{"function":"AddProfessor","Args":["Dr. Jones","Physics","<exact-caller-address>"]}'
```

---

### `forbidden: not instructor of 'CS101'`

An Instructor with `mark:own` trying to grade a course they're not assigned to.

**Fix:** Either reassign the course to them, or use an identity with `mark:any`.

---

## Network / Connection Errors

---

### `connection refused` / `dial tcp ... connect: connection refused`

**Cause:** Docker containers aren't running.

**Fix:**
```bash
# Check if running
docker ps | grep -E "peer|orderer"

# Start if stopped
cd ~/hyperledger/network
docker compose up -d
sleep 15
```

---

### `context deadline exceeded`

**Cause 1:** Orderer not reachable. Check the orderer container is up and `$ORDERER_CA` path is correct.

**Cause 2:** TLS cert mismatch — the cert file doesn't match the actual TLS cert in use.

**Fix:**
```bash
ls $ORDERER_CA   # must exist
docker ps | grep orderer   # must be Up
```

---

### `No such container: cli`

**Cause:** The old deploy scripts use `docker exec cli` but there's no cli container.

**Fix:** Don't use those scripts. Run `peer` commands directly from WSL2 as shown in [contract-reference.md](./contract-reference.md). The `peer` binary is at `~/hyperledger/network/fabric-samples/bin/peer`.

---

### `x509: certificate signed by unknown authority`

**Cause:** Wrong TLS cert file — you're pointing to the wrong CA.

**Fix:** Verify the cert paths:
```bash
# Peer TLS cert
ls ~/hyperledger/network/organizations/peerOrganizations/org1.university.com/peers/peer0.org1.university.com/tls/ca.crt

# Orderer TLS cert
ls ~/hyperledger/network/organizations/ordererOrganizations/university.com/orderers/orderer.university.com/msp/tlscacerts/tlsca.university.com-cert.pem
```

Both must exist. If not, crypto material needs to be regenerated (see [start-from-scratch.md](./start-from-scratch.md)).
