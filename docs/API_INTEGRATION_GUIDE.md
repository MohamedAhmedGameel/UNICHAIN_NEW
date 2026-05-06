# UNICHAIN — API Integration Guide
### For Newcomers: How the Blockchain Connects to the Web API

---

## Table of Contents

1. [How It All Fits Together](#1-how-it-all-fits-together)
2. [Prerequisites](#2-prerequisites)
3. [Environment Setup](#3-environment-setup)
4. [Starting the Full Stack](#4-starting-the-full-stack)
5. [How the API Works Internally](#5-how-the-api-works-internally)
6. [API Reference](#6-api-reference)
7. [Step-by-Step First Use](#7-step-by-step-first-use)
8. [Understanding Roles and Permissions](#8-understanding-roles-and-permissions)
9. [Common Errors and Fixes](#9-common-errors-and-fixes)
10. [Development Tips](#10-development-tips)

---

## 1. How It All Fits Together

```
Browser / curl / Postman
        │
        ▼
  FastAPI (Python)          :8000
  ─────────────────────────────────────────────────────
  auth/router.py     → Register / Login → JWT token
  identity/bridge.py → Maps your JWT user → Fabric X.509 cert
  fabric/gateway.py  → Calls `peer` CLI subprocess → Fabric network
        │
        ▼
  Fabric Network (Docker)
  ├── Fabric CA          :7054   Issues X.509 identities
  ├── Orderer            :7050   Orders transactions
  ├── Peer               :7051   Runs chaincode, stores ledger
  ├── CouchDB            :5984   Stores chaincode state
  └── Chaincode (Go)             Business logic on-chain
```

### The identity bridge — key concept

Every API user has **two identities**:

| Layer | What it is | Stored in |
|---|---|---|
| App user | Email + hashed password | SQLite (`unichain.db`) |
| Fabric identity | X.509 cert + private key | Filesystem wallet (`wallet/`) |

When a user calls any blockchain endpoint for the first time, the API automatically:
1. Registers a new username with Fabric CA
2. Enrolls to get an X.509 certificate + private key
3. Stores cert + key in the local `wallet/` directory
4. Records the mapping in SQLite

Every subsequent chaincode call signs the proposal with that user's private key,
so the chaincode sees the correct caller identity and enforces RBAC accordingly.

---

## 2. Prerequisites

Before starting the API, the Fabric network must be fully running.
Verify this first:

```bash
docker ps --format "table {{.Names}}\t{{.Status}}"
```

You must see all 5 containers showing `Up`:

```
NAMES                           STATUS
cli                             Up X minutes
peer0.org1.university.com       Up X minutes
orderer.university.com          Up X minutes
ca.org1.university.com          Up X minutes
couchdb0                        Up X minutes
```

If any container is missing or shows `Exited`, run the network setup first.
See `network/unichain_setup.sh`.

---

## 3. Environment Setup

### 3.1 — Create the `.env` file

```bash
cd ~/hyperledger/api

cat > .env << EOF
# ── JWT ──────────────────────────────────────────────────────────
# Change this to a long random string before any real deployment
JWT_SECRET=$(python3 -c "import secrets; print(secrets.token_hex(32))")
JWT_ALGORITHM=HS256
JWT_EXPIRE_MINUTES=1440

# ── Fabric CA ────────────────────────────────────────────────────
FABRIC_CA_URL=https://localhost:7054
FABRIC_CA_TLS_CERT=$HOME/hyperledger/network/organizations/fabric-ca/org1/tls-cert.pem
FABRIC_CA_ADMIN=admin
FABRIC_CA_ADMIN_PW=adminpw
FABRIC_CA_NAME=ca-org1

# ── Peer ─────────────────────────────────────────────────────────
FABRIC_PEER_ENDPOINT=localhost:7051
FABRIC_PEER_TLS_CERT=$HOME/hyperledger/network/organizations/peerOrganizations/org1.university.com/peers/peer0.org1.university.com/tls/ca.crt
FABRIC_CHANNEL=universitychannel
FABRIC_CHAINCODE=university
FABRIC_MSP_ID=Org1MSP

# ── Optional ─────────────────────────────────────────────────────
DATABASE_URL=sqlite+aiosqlite:///./unichain.db
WALLET_DIR=./wallet
EOF

echo "✓ .env created"
```

### 3.2 — Export peer subprocess environment variables

The `gateway.py` calls the `peer` binary as a subprocess.
These vars must be in the shell environment when you start uvicorn:

```bash
export CORE_PEER_TLS_ENABLED=true
export CORE_PEER_LOCALMSPID=Org1MSP
export CORE_PEER_ADDRESS=localhost:7051
export CORE_PEER_TLS_ROOTCERT_FILE=$HOME/hyperledger/network/organizations/peerOrganizations/org1.university.com/peers/peer0.org1.university.com/tls/ca.crt
export CORE_PEER_MSPCONFIGPATH=$HOME/hyperledger/network/organizations/peerOrganizations/org1.university.com/users/Admin@org1.university.com/msp
export ORDERER_CA=$HOME/hyperledger/network/organizations/ordererOrganizations/university.com/orderers/orderer.university.com/msp/tlscacerts/tlsca.university.com-cert.pem
export FABRIC_CFG_PATH=$HOME/fabric/config
```

> 💡 Add these exports to your `~/.bashrc` to avoid re-exporting every session.

### 3.3 — Install Python dependencies

```bash
cd ~/hyperledger/api

# Remove hfc (abandoned SDK — not used by the code)
sed -i '/^hfc/d' requirements.txt

pip3 install --break-system-packages -r requirements.txt
```

### 3.4 — Ensure peer binary is in PATH

```bash
source ~/.bashrc
which peer     # must print: /home/youruser/fabric/bin/peer
```

If blank: `echo 'export PATH=$PATH:$HOME/fabric/bin' >> ~/.bashrc && source ~/.bashrc`

---

## 4. Starting the Full Stack

Open **two Ubuntu terminals**:

### Terminal 1 — Fabric network (already running if you followed setup)

```bash
# Verify containers are up
docker ps --format "{{.Names}}: {{.Status}}"

# If they are down, restart with:
cd ~/hyperledger/network
docker compose up -d
sleep 15
docker exec -u root peer0.org1.university.com chmod 666 /var/run/docker.sock
```

### Terminal 2 — Python API

```bash
# Export peer env vars (every new terminal session)
export CORE_PEER_TLS_ENABLED=true
export CORE_PEER_LOCALMSPID=Org1MSP
export CORE_PEER_ADDRESS=localhost:7051
export CORE_PEER_TLS_ROOTCERT_FILE=$HOME/hyperledger/network/organizations/peerOrganizations/org1.university.com/peers/peer0.org1.university.com/tls/ca.crt
export CORE_PEER_MSPCONFIGPATH=$HOME/hyperledger/network/organizations/peerOrganizations/org1.university.com/users/Admin@org1.university.com/msp
export ORDERER_CA=$HOME/hyperledger/network/organizations/ordererOrganizations/university.com/orderers/orderer.university.com/msp/tlscacerts/tlsca.university.com-cert.pem
export FABRIC_CFG_PATH=$HOME/fabric/config

cd ~/hyperledger/api
uvicorn main:app --reload --port 8000
```

**Open in your Windows browser:** http://localhost:8000/docs

---

## 5. How the API Works Internally

### Registration flow

```
POST /auth/register
    │
    ├─► Create AppUser in SQLite (email + bcrypt password)
    ├─► Generate JWT token
    └─► Return token

(Fabric identity is NOT created yet — lazy creation on first blockchain call)
```

### First blockchain call flow

```
GET /roles  (with Authorization: Bearer <token>)
    │
    ├─► Decode JWT → get user_id
    ├─► Look up AppUser in SQLite
    ├─► identity/bridge.py: no Fabric identity found →
    │       ├─► Register username with Fabric CA (REST API)
    │       ├─► Enroll → receive X.509 cert + private key
    │       ├─► Store cert+key in wallet/ directory
    │       └─► Save mapping (app_user_id ↔ wallet_label) in SQLite
    │
    ├─► fabric/gateway.py:
    │       ├─► Write user cert+key to a temp MSP directory
    │       ├─► Run: peer chaincode query -C universitychannel -n university ...
    │       └─► Parse JSON response
    │
    └─► Return JSON to client
```

### Subsequent calls

```
(Identity already in wallet)
GET /majors
    │
    ├─► Decode JWT → AppUser → load Identity from wallet
    ├─► peer chaincode query ... (signed with user's cert)
    └─► Return JSON
```

---

## 6. API Reference

All endpoints except `/auth/register` and `/auth/login` require:
```
Authorization: Bearer <your_jwt_token>
```

### Auth

| Method | Endpoint | Body | Description |
|--------|----------|------|-------------|
| POST | `/auth/register` | `{email, password, display_name}` | Create account + get token |
| POST | `/auth/login` | `{email, password}` | Login + get token |
| GET | `/auth/me` | — | Current user info |

### Roles & Permissions

| Method | Endpoint | Permission Required | Description |
|--------|----------|-------------------|-------------|
| GET | `/roles` | any | List all roles |
| POST | `/roles` | `role:manage` | Create a role |
| DELETE | `/roles/{name}` | `role:manage` | Delete a role |
| GET | `/roles/{name}/permissions` | any | List role permissions |
| POST | `/roles/{name}/permissions` | `role:manage` | Grant permission to role |
| DELETE | `/roles/{name}/permissions/{perm}` | `role:manage` | Revoke permission |
| GET | `/roles/{name}/members` | any | List role members |
| POST | `/roles/{name}/members` | `role:manage` | Assign role to user |
| DELETE | `/roles/{name}/members` | `role:manage` | Remove role from user |

### Majors

| Method | Endpoint | Permission Required | Description |
|--------|----------|-------------------|-------------|
| GET | `/majors` | any | List all majors |
| GET | `/majors/{id}` | any | Get major by ID |
| GET | `/majors/code/{code}` | any | Get major by code |
| POST | `/majors` | `major:create` | Create major |
| PUT | `/majors/{id}` | `major:update` | Update major |
| DELETE | `/majors/{id}` | `major:deactivate` | Deactivate major |

### Professors

| Method | Endpoint | Permission Required | Description |
|--------|----------|-------------------|-------------|
| GET | `/professors` | any | List all professors |
| GET | `/professors/{id}` | any | Get professor |
| POST | `/professors` | `professor:create` | Add professor (auto-grants Instructor role) |
| PUT | `/professors/{id}` | `professor:update` | Update professor |
| DELETE | `/professors/{id}` | `professor:delete` | Remove professor |

### Students

| Method | Endpoint | Permission Required | Description |
|--------|----------|-------------------|-------------|
| GET | `/students` | any | List all students |
| GET | `/students/{id}` | any | Get student |
| POST | `/students` | `student:create` | Register student |
| PUT | `/students/{id}` | `student:update` | Update student |
| DELETE | `/students/{id}` | `student:delete` | Remove student |

### Courses

| Method | Endpoint | Permission Required | Description |
|--------|----------|-------------------|-------------|
| GET | `/courses` | any | List all courses |
| GET | `/courses/{id}` | any | Get course |
| POST | `/courses` | `course:create` | Create course |
| PUT | `/courses/{id}` | `course:update` | Update course |
| DELETE | `/courses/{id}` | `course:delete` | Delete course |
| PUT | `/courses/{id}/reassign` | `course:reassign` | Reassign to professor |

### Enrollments

| Method | Endpoint | Permission Required | Description |
|--------|----------|-------------------|-------------|
| GET | `/enrollments` | any | List all enrollments |
| POST | `/enrollments` | `enrollment:enroll` | Enroll student in course |
| DELETE | `/enrollments/{id}` | `enrollment:unenroll` | Unenroll student |
| POST | `/enrollments/batch` | `enrollment:batch` | Batch enroll |
| PUT | `/enrollments/{id}/mark` | `mark:own` or `mark:any` | Submit grade |

---

## 7. Step-by-Step First Use

### Step 1 — Register the first admin user

```bash
curl -s -X POST http://localhost:8000/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@unichain.com","password":"Admin1234!","display_name":"Admin"}' \
  | python3 -m json.tool
```

Response:
```json
{"access_token": "eyJ...", "token_type": "bearer"}
```

> ⚠️ The **first registered user** automatically becomes SuperAdmin on the blockchain
> (because their Fabric identity is the one that called InitLedger during deployment).
> Subsequent registered users start with no roles.

### Step 2 — Save the token

```bash
TOKEN=$(curl -s -X POST http://localhost:8000/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@unichain.com","password":"Admin1234!"}' \
  | python3 -c "import sys,json; print(json.load(sys.stdin)['access_token'])")

echo "Token: $TOKEN"
```

### Step 3 — Verify your blockchain roles

```bash
curl -s http://localhost:8000/roles \
  -H "Authorization: Bearer $TOKEN" | python3 -m json.tool
```

Should return `SuperAdmin`, `Registrar`, `Instructor`.

### Step 4 — Create a major

```bash
curl -s -X POST http://localhost:8000/majors \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"code":"CS","name":"Computer Science","description":"Core CS program"}' \
  | python3 -m json.tool
```

### Step 5 — Register a second user and grant them Registrar role

```bash
# Register second user
TOKEN2=$(curl -s -X POST http://localhost:8000/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"registrar@unichain.com","password":"Pass1234!","display_name":"Registrar"}' \
  | python3 -c "import sys,json; print(json.load(sys.stdin)['access_token'])")

# Get their user ID
USER2_ID=$(curl -s http://localhost:8000/auth/me \
  -H "Authorization: Bearer $TOKEN2" \
  | python3 -c "import sys,json; print(json.load(sys.stdin)['id'])")

# Admin grants Registrar role using their app user ID
curl -s -X POST "http://localhost:8000/roles/Registrar/members" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"app_user_id\": \"$USER2_ID\"}" \
  | python3 -m json.tool
```

---

## 8. Understanding Roles and Permissions

### Seeded roles (created by InitLedger)

| Role | Permissions | Typical user |
|------|-------------|--------------|
| `SuperAdmin` | All permissions | System deployer |
| `Registrar` | professor:create, student:*, course:*, enrollment:*, mark:any | Admin staff |
| `Instructor` | mark:own | Professors |

### Permission model

Permissions are stored on-chain as strings. The chaincode checks them per-call.
`SuperAdmin` bypasses all permission checks (hardcoded in chaincode).

Adding a professor via `POST /professors` **automatically grants** the `Instructor`
role to that professor's Fabric address — no manual role assignment needed.

### How RBAC flows through the stack

```
API call → gateway.py builds peer CLI command with user's cert/key
         → peer sends endorsed proposal to peer container
         → chaincode extracts caller's X.509 subject → SHA-256 → 40-char address
         → rbac.HasPermission(ctx, callerAddress, "major:create")
         → checks USER_ROLES~{address} → ROLE_PERMISSION~{role}~{perm}
         → approve or reject
```

---

## 9. Common Errors and Fixes

### `"Chaincode call failed: ... forbidden: missing permission"`

Your user does not have the required role. Ask a SuperAdmin to grant it:
```bash
# SuperAdmin grants role to user
curl -X POST http://localhost:8000/roles/Registrar/members \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"app_user_id": "target-user-uuid"}'
```

### `"CA admin enroll failed: 401"`

The Fabric CA admin credentials in `.env` are wrong.
Default values are `admin` / `adminpw`. Check `FABRIC_CA_ADMIN` and `FABRIC_CA_ADMIN_PW`.

### `"Chaincode call failed: ... peer: command not found"`

The `peer` binary is not in PATH. In your API terminal:
```bash
export PATH=$PATH:$HOME/fabric/bin
which peer   # must print a path
```

### `"Chaincode call timed out"`

The Fabric network is slow or a container is down. Check:
```bash
docker ps   # all 5 must be Up
```

### `pydantic_core.ValidationError: Extra inputs are not permitted`

You added `CORE_PEER_*` variables to `.env`. Move them to shell exports instead —
pydantic rejects unknown fields in `.env`. See Section 3.2.

### `"CA enroll failed: 404"`

`FABRIC_CA_URL` or `FABRIC_CA_NAME` in `.env` is wrong.
Default CA name is `ca-org1`. Check `docker logs ca.org1.university.com`.

---

## 10. Development Tips

### Adding a new chaincode function to the API

1. Add the Go function to `chaincode/university/chaincode.go`
2. Redeploy the chaincode (increment version and sequence):
   ```bash
   # From ~/hyperledger/network
   docker exec cli peer lifecycle chaincode package university_vX.tar.gz \
     --path /opt/.../chaincode/university --lang golang --label university_X.0
   # install → approve → commit (increment --sequence)
   ```
3. Add a new FastAPI route in `api/routers/`:
   ```python
   @router.get("/my-endpoint")
   async def my_endpoint(user=Depends(get_current_user), db=Depends(get_db)):
       identity = await get_or_create_identity(user, db)
       return await svc.evaluate(identity, "MyChaincodeFn", "arg1")
   ```
   - Use `svc.evaluate()` for read queries (no ledger commit)
   - Use `svc.submit()` for write transactions (committed to ledger)

### Testing chaincode directly without the API

```bash
# Set env vars (from network dir)
export CORE_PEER_TLS_ENABLED=true
export CORE_PEER_LOCALMSPID=Org1MSP
export CORE_PEER_ADDRESS=localhost:7051
export CORE_PEER_TLS_ROOTCERT_FILE=$HOME/hyperledger/network/organizations/peerOrganizations/org1.university.com/peers/peer0.org1.university.com/tls/ca.crt
export CORE_PEER_MSPCONFIGPATH=$HOME/hyperledger/network/organizations/peerOrganizations/org1.university.com/users/Admin@org1.university.com/msp
export ORDERER_CA=$HOME/hyperledger/network/organizations/ordererOrganizations/university.com/orderers/orderer.university.com/msp/tlscacerts/tlsca.university.com-cert.pem
export FABRIC_CFG_PATH=$HOME/fabric/config

# Query
peer chaincode query -C universitychannel -n university \
  -c '{"function":"GetAllMajors","Args":[]}'

# Invoke
peer chaincode invoke -o localhost:7050 \
  --ordererTLSHostnameOverride orderer.university.com \
  --tls --cafile $ORDERER_CA \
  -C universitychannel -n university \
  -c '{"function":"AddMajor","Args":["EE","Electrical Engineering",""]}' \
  --peerAddresses localhost:7051 --tlsRootCertFiles $CORE_PEER_TLS_ROOTCERT_FILE
```

### Wallet location

User identities are stored at `api/wallet/<wallet_label>/`:
```
wallet/
└── user_<uuid>/
    ├── cert.pem      X.509 certificate (public)
    └── key.pem       Private key (keep secret)
```

Never commit the `wallet/` directory to git — it is in `.gitignore`.

### Database location

The SQLite database is at `api/unichain.db`. It stores:
- `app_users` — email, hashed password, display name
- `fabric_identities` — mapping from app user → Fabric wallet label + address

Never commit `unichain.db` to git — it is in `.gitignore`.
