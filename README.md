# UNICHAIN

A blockchain-backed university management system built on Hyperledger Fabric 2.5.
All academic records — majors, professors, students, courses, enrollments, and grades —
are stored on a permissioned distributed ledger. Access to every operation is controlled
by a custom role-based access control system enforced inside the chaincode itself,
not at the application layer.

---

## Contents

- [Overview](#overview)
- [System Architecture](#system-architecture)
- [Technology Stack](#technology-stack)
- [Repository Structure](#repository-structure)
- [Network Layer](#network-layer)
- [API Layer](#api-layer)
  - [Application Startup](#application-startup)
  - [Authentication](#authentication)
  - [Identity Bridge](#identity-bridge)
  - [Fabric Gateway](#fabric-gateway)
  - [Wallet](#wallet)
  - [CA Client](#ca-client)
  - [Endpoints](#endpoints)
- [Chaincode Layer](#chaincode-layer)
- [Role-Based Access Control](#role-based-access-control)
- [Data Flow](#data-flow)
- [Getting Started](#getting-started)
- [Configuration Reference](#configuration-reference)
- [Development Guide](#development-guide)

---

## Overview

UNICHAIN replaces a traditional university database with an append-only blockchain
ledger. Every write — registering a student, grading a course, enrolling in a major —
is a signed transaction committed to the ledger. Every read is a query against the
current ledger state stored in CouchDB.

The system enforces the following guarantees:

- No record can be silently altered. Every change produces a transaction with a
  timestamp, a transaction ID, and the identity of the caller.
- Access to every chaincode function is gated by a permission check. A professor
  cannot grade a course they were not assigned to. A registrar cannot manage roles.
- Each API user has a corresponding cryptographic identity on the blockchain. API
  calls are signed with that identity's private key before being submitted to the peer.

---

## System Architecture

```
Client (browser / curl / Postman)
          |
          | HTTP
          v
  +-----------------+
  |    FastAPI       |   Port 8000
  |  (Python 3.12)  |
  +-----------------+
       |        |
       |        | reads/writes app user table
       |        v
       |   +----------+
       |   |  SQLite  |   unichain.db
       |   +----------+
       |
       | peer CLI subprocess (signed with user X.509 cert)
       v
  +----------------------------------------------------+
  |  Hyperledger Fabric Network (Docker)               |
  |                                                    |
  |  Fabric CA         :7054   issues X.509 certs      |
  |  Orderer           :7050   orders transactions     |
  |  Peer              :7051   hosts chaincode, ledger |
  |  CouchDB           :5984   stores ledger state     |
  |  CLI container             admin tooling           |
  +----------------------------------------------------+
```

### Two-database design

The system uses two separate stores with distinct responsibilities:

| Store | Technology | What it holds |
|-------|-----------|---------------|
| Off-chain | SQLite (async) | App users, hashed passwords, wallet label mappings |
| On-chain | Hyperledger Fabric + CouchDB | All academic domain data, roles, permissions |

The off-chain database holds only what is needed to authenticate a user and locate
their Fabric identity. All business data lives on the blockchain.

---

## Technology Stack

| Layer | Technology | Version |
|-------|-----------|---------|
| Blockchain network | Hyperledger Fabric | 2.5.7 |
| Fabric CA | fabric-ca | 1.5.7 |
| State database | CouchDB | 3.3 |
| Chaincode language | Go | 1.21 |
| Chaincode framework | fabric-contract-api-go | 1.2.2 |
| API framework | FastAPI | latest |
| API runtime | Python | 3.12 |
| API server | Uvicorn | latest |
| ORM | SQLAlchemy (async) | 2.x |
| Database driver | aiosqlite | latest |
| Auth | python-jose (JWT) + passlib (bcrypt) | latest |
| CA client | httpx (async) | latest |
| Crypto | cryptography | latest |

---

## Repository Structure

```
.
├── api/                        Python FastAPI application
│   ├── main.py                 Application entry point, router registration
│   ├── config.py               Environment-driven settings (pydantic-settings)
│   ├── requirements.txt        Python dependencies
│   ├── auth/
│   │   ├── models.py           AppUser SQLAlchemy model
│   │   ├── router.py           /auth endpoints + JWT dependency
│   │   └── service.py          Password hashing, JWT creation/verification
│   ├── db/
│   │   └── database.py         Async SQLAlchemy engine, session factory, Base
│   ├── fabric/
│   │   ├── ca_client.py        Fabric CA REST client (register + enroll)
│   │   ├── gateway.py          Peer CLI wrapper (submit + evaluate)
│   │   └── wallet.py           Filesystem identity wallet + address derivation
│   ├── identity/
│   │   ├── bridge.py           Maps AppUser to Fabric identity (lazy creation)
│   │   └── models.py           FabricIdentity SQLAlchemy model
│   ├── routers/
│   │   ├── courses.py          /courses endpoints
│   │   ├── enrollments.py      /enrollments endpoints
│   │   ├── majors.py           /majors endpoints
│   │   ├── professors.py       /professors endpoints
│   │   ├── roles.py            /roles, /permissions, /users endpoints
│   │   └── students.py         /students endpoints
│   └── services/
│       ├── fabric_svc.py       FabricService base class (submit + evaluate)
│       └── gpa.py              Client-side GPA calculation from transcript
│
├── chaincode/
│   └── university/             Go chaincode
│       ├── chaincode.go        Contract entry point, all callable functions
│       ├── go.mod / go.sum     Go module definition
│       ├── rbac/
│       │   ├── rbac.go         Role/permission/member state management
│       │   └── seed.go         InitLedger seed data
│       └── state/
│           ├── course.go       Course state CRUD
│           ├── enrollment.go   Enrollment state CRUD
│           ├── major.go        Major state CRUD
│           ├── professor.go    Professor state CRUD
│           └── student.go      Student state CRUD
│
└── network/
    ├── configtx.yaml           Channel and consortium configuration
    ├── crypto-config.yaml      Organization and node definitions
    ├── docker-compose.yml      Container definitions for all Fabric services
    ├── unichain_setup.sh       One-shot network bootstrap script
    └── scripts/
        ├── bootstrap.sh        Generate crypto, start containers, join channel
        └── deploy_chaincode.sh Package, install, approve, commit chaincode
```

---

## Network Layer

The Fabric network consists of five Docker containers defined in
`network/docker-compose.yml`.

**ca.org1.university.com** — Port 7054

Issues X.509 certificates to new users when they interact with the blockchain for
the first time. The API communicates with the CA over its REST API using admin
credentials to register a new username, then enrolls that username to receive a
signed certificate and the corresponding private key.

**orderer.university.com** — Ports 7050, 7053

Receives endorsed transaction proposals from the peer, orders them into blocks, and
delivers the blocks back to the peer for committing to the ledger. Uses the etcd-based
Raft consensus protocol. The admin port 7053 is used by `osnadmin` to join the orderer
to the channel during network setup.

**peer0.org1.university.com** — Port 7051

The single peer in this network. Holds the ledger, runs the chaincode container,
endorses transaction proposals, and commits blocks. All API calls ultimately land here
via the `peer chaincode invoke` and `peer chaincode query` CLI commands. The peer
requires access to the Docker socket at runtime in order to launch the Go chaincode
container on first invocation.

**couchdb0** — Port 5984

The state database for the peer. Stores the current world state of all chaincode keys
as JSON documents. Enables rich queries from the chaincode using CouchDB selectors in
addition to standard key range queries.

**cli**

A fabric-tools container used during deployment. The chaincode package is installed
through this container. It shares the Docker network with all other services and has
volume mounts for the crypto material, channel artifacts, and chaincode source.

---

## API Layer

The API is a single FastAPI application. It has no business logic of its own. It
translates HTTP requests into signed chaincode calls and returns the results. All
validation, authorization, and state management happen inside the chaincode.

### Application Startup

`main.py` is the application entry point. On startup it:

1. Calls `init_db()` to create SQLite tables if they do not exist.
2. Registers nine routers covering auth, majors, professors, students, courses,
   enrollments, roles, permissions, and user lookups.
3. Attaches CORS middleware.
4. Exposes `/health` and `/` endpoints for basic liveness checks.

```bash
uvicorn main:app --reload --port 8000
```

The interactive API documentation is available at `http://localhost:8000/docs`.

---

### Authentication

Authentication is handled entirely in `auth/`. It establishes who you are for the
purpose of loading your Fabric identity. Authorization — what you are allowed to do —
is enforced exclusively by the chaincode.

**Registration** (`POST /auth/register`)

Accepts an email, password, and display name. Creates an `AppUser` row in SQLite
with the password stored as a bcrypt hash. Returns a JWT on success. No Fabric
identity is created at this point. Identity provisioning happens lazily on the first
blockchain call.

**Login** (`POST /auth/login`)

Verifies the submitted password against the stored bcrypt hash. Returns a new JWT
if the credentials are valid.

**Token verification**

Every protected endpoint depends on `get_current_user`, defined in `auth/router.py`.
This dependency extracts the Bearer token from the `Authorization` header, decodes
the JWT using the configured `JWT_SECRET`, loads the matching `AppUser` from SQLite,
and injects the user object into the route handler. Requests with missing, expired,
or invalid tokens receive a 401 response before any chaincode call is attempted.

```
Authorization: Bearer <token>
```

**AppUser model** (`auth/models.py`)

| Column | Type | Description |
|--------|------|-------------|
| id | String (UUID) | Primary key |
| email | String | Unique, indexed |
| password_hash | String | bcrypt hash |
| display_name | String | Optional |
| created_at | DateTime | UTC timestamp |

---

### Identity Bridge

`identity/bridge.py` is the core integration point between the application user
table and the Fabric network. It implements lazy identity creation — a Fabric identity
is provisioned only when a user makes their first blockchain call.

**FabricIdentity model** (`identity/models.py`)

| Column | Type | Description |
|--------|------|-------------|
| id | String (UUID) | Primary key |
| app_user_id | String | Foreign key to AppUser |
| fabric_username | String | Username registered with the CA |
| msp_id | String | MSP identifier (Org1MSP) |
| wallet_label | String | Directory name under wallet/ |
| fabric_address | String | 40-char hex address derived from cert |

**`get_or_create_identity(user, db)`**

This function is called at the start of every blockchain route handler. Its logic:

1. Queries SQLite for an existing `FabricIdentity` row for the current user.
2. If found, loads the cert and key from the wallet directory and returns an
   `Identity` dataclass immediately.
3. If no mapping exists, or the wallet files are missing, it:
   - Generates a `fabric_username` from the first 16 characters of the UUID
     with hyphens removed (e.g. `user_6ba7b8109dad11d1`).
   - Generates a random hex secret for CA enrollment.
   - Calls `ca_client.register(username, secret)` to create the identity in the CA.
   - Calls `ca_client.enroll(username, secret)` to receive the signed X.509 cert
     and private key.
   - Derives the on-chain Fabric address from the certificate.
   - Writes the cert and key to the wallet via `wallet.store()`.
   - Inserts the `FabricIdentity` mapping into SQLite.
   - Returns the loaded `Identity`.

This approach means the first API call for a new user is slightly slower due to the
CA round-trip. All subsequent calls load directly from the wallet.

**`get_fabric_address_by_user_id(user_id, db)`**

A lookup-only function used by the roles router to resolve an app user ID to a Fabric
address without triggering identity creation. Returns `None` if the user has no
identity yet. The caller is responsible for deciding how to handle the `None` case.

---

### Fabric Gateway

`fabric/gateway.py` translates Python function calls into signed chaincode invocations
using the `peer` CLI binary as a subprocess.

**Design note**

The Fabric Gateway gRPC SDK is the recommended approach for production deployments.
This implementation uses the `peer` CLI instead because it requires no additional
dependency setup beyond the binaries already installed for network administration,
and it works reliably across all Fabric 2.x versions. The interface exposed to
callers is identical — replacing the `_invoke` function with a gRPC implementation
requires no changes to any router or service file.

**`submit_transaction(identity, function, *args)`**

Executes a write transaction. The transaction is endorsed by the peer, ordered by
the orderer, and committed to the ledger. The ledger state change is permanent
and auditable via `GetHistoryForKey`.

**`evaluate_transaction(identity, function, *args)`**

Executes a read-only query. The proposal is simulated on the peer only. No
transaction reaches the orderer and nothing is written to the ledger.

**Internal flow of `_invoke`**

1. Creates a temporary directory with a minimal MSP structure containing the
   calling user's certificate and private key:
   ```
   /tmp/fabric_XXXX/msp/
   ├── signcerts/cert.pem    (user X.509 cert from wallet)
   ├── keystore/key.pem      (user private key from wallet)
   └── cacerts/              (org CA certs copied from peer MSP)
   ```
2. Constructs the peer environment variables pointing to this temporary MSP:
   ```
   CORE_PEER_TLS_ENABLED=true
   CORE_PEER_LOCALMSPID=Org1MSP
   CORE_PEER_TLS_ROOTCERT_FILE=<peer TLS cert>
   CORE_PEER_MSPCONFIGPATH=<temp MSP dir>
   CORE_PEER_ADDRESS=localhost:7051
   ```
3. Builds the `peer chaincode query` or `peer chaincode invoke` command with the
   channel name, chaincode name, and function arguments serialized as JSON:
   ```json
   {"function": "AddMajor", "Args": ["CS", "Computer Science", ""]}
   ```
4. Runs the command with a 30-second timeout via `subprocess.run`.
5. Parses stdout as JSON. Returns a `{"result": ...}` wrapper if the output is not
   valid JSON. Raises `RuntimeError` with the stderr content if the exit code is
   non-zero.
6. Deletes the temporary directory in both success and timeout cases.

The orderer CA file path for invoke commands is derived programmatically by
traversing up from the configured `FABRIC_PEER_TLS_CERT` path — four levels up, then
into the orderer MSP tlscacerts directory.

---

### Wallet

`fabric/wallet.py` manages the filesystem store of user identities. Each identity
occupies a directory under the configured `wallet_path` (default `./wallet`):

```
wallet/
└── user_<uuid>/
    ├── cert.pem      X.509 certificate signed by the Fabric CA
    ├── key.pem       EC private key (SECP256R1, PKCS8, unencrypted)
    └── msp_id.txt    MSP identifier string
```

**`cert_to_address(cert_pem)`**

Derives the deterministic 40-character hex address that the chaincode uses to
identify the caller. The derivation must match the Go chaincode's
`getCallerAddress()` function exactly, or all permission checks will fail.

The derivation steps:

1. Parse the PEM certificate using the `cryptography` library.
2. Format the subject DN as an RFC 4514 string.
3. Format the issuer DN as an RFC 4514 string.
4. Concatenate as `x509::{subject}::{issuer}`.
5. Compute SHA-256 of the concatenated string.
6. Return the first 40 hexadecimal characters.

This mirrors Fabric's internal `clientIdentity.GetID()` return value, which is
what the Go chaincode hashes on the ledger side to derive the same address.

---

### CA Client

`fabric/ca_client.py` communicates with the Fabric CA REST API. A singleton
`ca_client` instance is created at module import time and shared across all requests.

**Admin token acquisition**

The first time a user needs to be registered, `_get_admin_token()` enrolls the CA
admin using basic auth (admin ID and password from settings). It generates a
throwaway EC key pair, signs a CSR with it, and posts the CSR to the CA `/enroll`
endpoint. The returned certificate PEM is cached in memory as `_admin_token` and
reused for all subsequent registration calls within the same process lifetime.

**`register(username, secret)`**

Posts to `/register` authenticated as the CA admin. Creates a new `client`-type
identity in the CA with affiliation `org1.department1`. The function is idempotent
with respect to the "already registered" error — duplicate registration attempts do
not raise an exception, which makes retries safe.

**`enroll(username, secret)`**

Posts to `/enroll` authenticated as the username being enrolled. Generates a fresh
EC key pair and CSR, submits only the CSR (public key) to the CA, and receives the
signed X.509 certificate in return. The private key is generated locally and never
transmitted. Returns the `(cert_pem, key_pem)` tuple.

---

### Endpoints

All endpoints except `/auth/register` and `/auth/login` require a valid JWT in the
`Authorization: Bearer` header. The permission column indicates what on-chain
permission the chaincode requires from the caller. `none` means the chaincode
performs no permission check and any authenticated user may call the endpoint.

#### Auth

| Method | Path | Description |
|--------|------|-------------|
| POST | /auth/register | Create account, returns JWT |
| POST | /auth/login | Verify credentials, returns JWT |
| GET | /auth/me | Return current user app profile |

#### Majors

| Method | Path | Chaincode Function | Permission |
|--------|------|--------------------|-----------|
| GET | /majors | GetAllMajors | none |
| GET | /majors/{id} | GetMajor | none |
| GET | /majors/code/{code} | GetMajorByCode | none |
| POST | /majors | AddMajor | major:create |
| PUT | /majors/{id} | UpdateMajor | major:update |
| DELETE | /majors/{id} | DeactivateMajor | major:deactivate |

#### Professors

| Method | Path | Chaincode Function | Permission |
|--------|------|--------------------|-----------|
| GET | /professors | GetAllProfessors | none |
| GET | /professors/{id} | GetProfessor | none |
| POST | /professors | AddProfessor | professor:create |
| PUT | /professors/{id} | UpdateProfessor | professor:update |
| DELETE | /professors/{id} | DeleteProfessor | professor:delete |

Creating a professor via `POST /professors` automatically grants the `Instructor`
role to that professor's Fabric address inside the same chaincode transaction.
No separate role assignment step is required.

#### Students

| Method | Path | Chaincode Function | Permission |
|--------|------|--------------------|-----------|
| GET | /students | GetAllStudents | none |
| GET | /students/{id} | GetStudent | none |
| POST | /students | AddStudent | student:create |
| PUT | /students/{id} | UpdateStudent | student:update |
| DELETE | /students/{id} | DeleteStudent | student:delete |

#### Courses

| Method | Path | Chaincode Function | Permission |
|--------|------|--------------------|-----------|
| GET | /courses | GetAllCourses | none |
| GET | /courses/{id} | GetCourse | none |
| POST | /courses | AddCourse | course:create |
| PUT | /courses/{id} | UpdateCourse | course:update |
| DELETE | /courses/{id} | DeleteCourse | course:delete |
| PUT | /courses/{id}/reassign | ReassignCourse | course:reassign |

#### Enrollments

| Method | Path | Chaincode Function | Permission |
|--------|------|--------------------|-----------|
| GET | /enrollments | GetAllEnrollments | none |
| POST | /enrollments | EnrollStudent | enrollment:enroll |
| POST | /enrollments/batch | BatchEnroll | enrollment:batch |
| DELETE | /enrollments/{id} | UnenrollStudent | enrollment:unenroll |
| PUT | /enrollments/{id}/mark | UpdateMark | mark:own or mark:any |
| GET | /enrollments/transcript/{student_id} | GetTranscript | none |
| GET | /enrollments/gpa/{student_id} | GetTranscript (computed) | none |

The GPA endpoint calls `GetTranscript` to retrieve all enrollment records for the
student and computes the cumulative GPA client-side using the standard 4.0 scale
letter-grade mapping defined in `services/gpa.py`.

The `mark:own` permission allows a professor to submit grades only for courses where
the `professor_address` field on the course record matches their own Fabric address.
The `mark:any` permission allows grading any course regardless of assignment.

#### Roles and Permissions

| Method | Path | Chaincode Function | Permission |
|--------|------|--------------------|-----------|
| GET | /roles | GetAllRoles | none |
| POST | /roles | CreateRole | role:manage |
| DELETE | /roles/{name} | DeleteRole | role:manage |
| GET | /roles/{name}/permissions | GetRolePermissions | none |
| POST | /roles/{name}/permissions | GrantPermission | role:manage |
| DELETE | /roles/{name}/permissions/{perm} | RevokePermission | role:manage |
| GET | /roles/{name}/members | GetRoleMembers | none |
| POST | /roles/{name}/members | GrantRole | role:manage |
| DELETE | /roles/{name}/members/{address} | RevokeRole | role:manage |
| GET | /permissions | GetAllPermissions | none |
| POST | /permissions | CreatePermission | role:manage |
| GET | /users/{id}/roles | GetUserRoles | none |
| GET | /users/{id}/permissions | GetUserEffectivePermissions | none |
| GET | /users/{id}/fabric-address | SQLite lookup | none |

When assigning a role via `POST /roles/{name}/members`, the request body accepts
either a `fabric_address` (40-char hex string) or an `app_user_id` (UUID). When
`app_user_id` is provided, the bridge resolves it to a Fabric address using the
`FabricIdentity` table. The target user must have already made at least one blockchain
call, as identity creation is lazy and the Fabric address does not exist in the
database until the first call triggers enrollment.

---

## Chaincode Layer

The chaincode is written in Go using the `fabric-contract-api-go` framework and
runs inside a Docker container managed by the Fabric peer. It is the single source
of truth for all business logic and access control.

All chaincode state is stored in CouchDB as key-value pairs. The keys follow a
naming convention that groups records by entity type and enables range queries:

```
MAJOR~{id}
PROFESSOR~{id}
STUDENT~{id}
COURSE~{id}
ENROLLMENT~{id}
ROLE~{name}
ROLE_MEMBER~{role}~{address}
ROLE_MEMBER_LIST~{role}
USER_ROLE~{address}~{role}
USER_ROLE_LIST~{address}
ROLE_PERM~{role}~{perm}
ROLE_PERM_LIST~{role}
COUNTER~{entity}
```

Integer IDs for domain records are managed by per-entity counter keys
stored as `COUNTER~{entity}`. Each `Add*` function reads the counter,
increments it, writes it back, and uses the new value as the record ID.

`InitLedger` seeds three roles on first deployment: `SuperAdmin`, `Registrar`,
and `Instructor`, each with a predefined permission set. It grants `SuperAdmin`
to the caller's Fabric address — meaning the deployer's identity becomes the
initial system administrator automatically.

---

## Role-Based Access Control

RBAC is enforced entirely inside the chaincode. The API performs no access control.

**Three-layer model**

```
User (Fabric address derived from X.509 cert)
    assigned to --> Role
                      grants --> Permission
```

**Seeded roles**

| Role | Description |
|------|-------------|
| SuperAdmin | Full system access. Bypasses all permission checks. |
| Registrar | Manages professors, students, courses, and enrollments. |
| Instructor | Submits grades for assigned courses only. |

**Authorization sequence for each chaincode call**

1. The peer CLI signs the transaction proposal with the caller's private key.
2. The peer extracts the X.509 certificate from the signed proposal.
3. The chaincode calls `clientIdentity.GetID()` to get the identity string,
   formatted as `x509::{subject DN}::{issuer DN}`.
4. It SHA-256 hashes that string and takes the first 40 hex characters as the
   caller's on-chain address.
5. It reads `USER_ROLE_LIST~{address}` from the ledger to find the caller's roles.
6. For each role, it reads `ROLE_PERM~{role}~{required_permission}`.
7. The call proceeds if any role grants the required permission, or if the caller
   holds `SuperAdmin` (which bypasses all permission checks unconditionally).

**The `mark:own` vs `mark:any` distinction**

When `UpdateMark` is called by a user with only `mark:own`, the chaincode fetches
the course record and compares its `professor_address` field with the caller's
derived address. If they do not match, the transaction is rejected. Users with
`mark:any` (granted to the Registrar role) bypass this ownership check.

---

## Data Flow

**First API call for a new user**

```
POST /majors  (Authorization: Bearer <token>)
    |
    +-- get_current_user()
    |       decode JWT -> user_id
    |       SELECT AppUser FROM sqlite WHERE id = user_id
    |
    +-- get_or_create_identity(user, db)
    |       SELECT FabricIdentity WHERE app_user_id = user.id  ->  not found
    |       generate fabric_username, secret
    |       POST https://localhost:7054/register  (CA admin basic auth)
    |       POST https://localhost:7054/enroll    (username:secret basic auth)
    |       receive cert_pem, key_pem
    |       wallet.store("user_<uuid>", cert_pem, key_pem)
    |       INSERT FabricIdentity -> sqlite
    |       return Identity(cert_pem, key_pem, ...)
    |
    +-- svc.submit(identity, "AddMajor", "CS", "Computer Science", "")
            gateway._invoke(identity, "AddMajor", ["CS",...], evaluate=False)
                write cert+key to /tmp/fabric_XXXX/msp/
                build command:
                  peer chaincode invoke
                    -C universitychannel -n university
                    -c '{"function":"AddMajor","Args":["CS","Computer Science",""]}'
                    --tls --cafile <orderer CA cert>
                    -o localhost:7050
                    --peerAddresses localhost:7051
                    CORE_PEER_MSPCONFIGPATH=/tmp/fabric_XXXX/msp
                subprocess.run(cmd, timeout=30)
                peer signs proposal with user's cert+key
                peer endorses, submits to orderer
                orderer orders into block, delivers to peer
                peer commits block to ledger
                return {"id": 1, "code": "CS", "name": "Computer Science", ...}
```

**Subsequent calls (identity already in wallet)**

```
GET /majors  (Authorization: Bearer <token>)
    |
    +-- get_current_user()         JWT decode + SQLite lookup
    +-- get_or_create_identity()   FabricIdentity found -> load from wallet files
    +-- svc.evaluate(identity, "GetAllMajors")
            peer chaincode query -C universitychannel -n university
              -c '{"function":"GetAllMajors","Args":[]}'
            no orderer involved, no ledger commit
            return [{...}, {...}, ...]
```

---

## Getting Started

### Requirements

- Windows 11 with WSL2 (Ubuntu 24.04)
- Docker Desktop 4.x with WSL2 backend enabled
- Hyperledger Fabric 2.5.7 binaries installed in WSL2
- Go 1.21 installed in WSL2
- Python 3.12 installed in WSL2

### Start the network

```bash
cd ~/hyperledger/network
bash unichain_setup.sh
```

This script generates crypto material, starts all Docker containers, joins the
orderer and peer to the channel, and deploys the chaincode including the
`InitLedger` call.

### Start the API

```bash
export CORE_PEER_TLS_ENABLED=true
export CORE_PEER_LOCALMSPID=Org1MSP
export CORE_PEER_ADDRESS=localhost:7051
export CORE_PEER_TLS_ROOTCERT_FILE=$HOME/hyperledger/network/organizations/peerOrganizations/org1.university.com/peers/peer0.org1.university.com/tls/ca.crt
export CORE_PEER_MSPCONFIGPATH=$HOME/hyperledger/network/organizations/peerOrganizations/org1.university.com/users/Admin@org1.university.com/msp
export ORDERER_CA=$HOME/hyperledger/network/organizations/ordererOrganizations/university.com/orderers/orderer.university.com/msp/tlscacerts/tlsca.university.com-cert.pem
export FABRIC_CFG_PATH=$HOME/fabric/config

cd ~/hyperledger/api
pip3 install --break-system-packages -r requirements.txt
uvicorn main:app --reload --port 8000
```

Open `http://localhost:8000/docs` in a browser.

### First steps after startup

```bash
# Register the first user (this user becomes SuperAdmin)
curl -s -X POST http://localhost:8000/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@uni.com","password":"Admin1234!","display_name":"Admin"}'

# Login and save token
TOKEN=$(curl -s -X POST http://localhost:8000/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@uni.com","password":"Admin1234!"}' \
  | python3 -c "import sys,json; print(json.load(sys.stdin)['access_token'])")

# Verify on-chain roles
curl -s http://localhost:8000/roles \
  -H "Authorization: Bearer $TOKEN" | python3 -m json.tool
```

The first registered user's Fabric identity is the one that makes the initial
`GetAllRoles` call. Because this user's address was not the one that called
`InitLedger` during deployment, they will not automatically hold `SuperAdmin`.
To grant SuperAdmin to the first API user, call the roles endpoint using the
deployment admin's peer CLI directly, or use the CLI container as the signing
identity before any API users exist.

---

## Configuration Reference

All settings are loaded from a `.env` file at `api/.env` via pydantic-settings.
The `CORE_PEER_*` variables must be exported as shell environment variables rather
than placed in `.env`, because they are consumed by the peer subprocess at runtime
and pydantic rejects unknown fields.

| Variable | Default | Description |
|----------|---------|-------------|
| JWT_SECRET | (required) | Secret key for signing JWT tokens. No default. |
| JWT_ALGORITHM | HS256 | JWT signing algorithm |
| JWT_EXPIRE_MINUTES | 1440 | Token lifetime in minutes (24 hours) |
| FABRIC_CHANNEL | universitychannel | Fabric channel name |
| FABRIC_CHAINCODE | university | Deployed chaincode name |
| FABRIC_MSP_ID | Org1MSP | Organization MSP identifier |
| FABRIC_PEER_ENDPOINT | localhost:7051 | Peer address for CLI subprocess |
| FABRIC_PEER_TLS_CERT | (path) | Path to peer TLS root certificate |
| FABRIC_CA_URL | https://localhost:7054 | Fabric CA base URL |
| FABRIC_CA_NAME | ca-org1 | CA name as configured in the network |
| FABRIC_CA_ADMIN | admin | CA admin username |
| FABRIC_CA_ADMIN_PW | adminpw | CA admin password |
| FABRIC_CA_TLS_CERT | (path) | Path to CA TLS certificate |
| WALLET_PATH | ./wallet | Directory for storing user identity files |
| DATABASE_URL | sqlite+aiosqlite:///./unichain.db | SQLAlchemy connection string |

---

## Development Guide

### Adding a new chaincode function to the API

1. Implement the function in `chaincode/university/chaincode.go` with the
   appropriate RBAC guard.
2. Redeploy the chaincode by incrementing the version label and sequence number:
   ```bash
   docker exec cli peer lifecycle chaincode package university_vX.tar.gz \
     --path /opt/gopath/src/github.com/hyperledger/fabric/peer/chaincode/university \
     --lang golang --label university_X.0
   # Then install, approve (--sequence N), and commit
   ```
3. Add a route handler in the appropriate router file:
   ```python
   @router.get("/{record_id}")
   async def get_record(record_id: int,
                        user=Depends(get_current_user),
                        db: AsyncSession = Depends(get_db)):
       identity = await get_or_create_identity(user, db)
       return await svc.evaluate(identity, "GetRecord", str(record_id))
   ```
   Use `svc.evaluate` for read-only chaincode functions.
   Use `svc.submit` for functions that write to the ledger.
4. Register the router in `main.py` if it is a new file.

### Testing chaincode functions directly without the API

```bash
export CORE_PEER_TLS_ENABLED=true
export CORE_PEER_LOCALMSPID=Org1MSP
export CORE_PEER_ADDRESS=localhost:7051
export CORE_PEER_TLS_ROOTCERT_FILE=$HOME/hyperledger/network/organizations/peerOrganizations/org1.university.com/peers/peer0.org1.university.com/tls/ca.crt
export CORE_PEER_MSPCONFIGPATH=$HOME/hyperledger/network/organizations/peerOrganizations/org1.university.com/users/Admin@org1.university.com/msp
export ORDERER_CA=$HOME/hyperledger/network/organizations/ordererOrganizations/university.com/orderers/orderer.university.com/msp/tlscacerts/tlsca.university.com-cert.pem
export FABRIC_CFG_PATH=$HOME/fabric/config

# Read query
peer chaincode query -C universitychannel -n university \
  -c '{"function":"GetAllRoles","Args":[]}'

# Write transaction
peer chaincode invoke -o localhost:7050 \
  --ordererTLSHostnameOverride orderer.university.com \
  --tls --cafile $ORDERER_CA \
  -C universitychannel -n university \
  -c '{"function":"AddMajor","Args":["EE","Electrical Engineering",""]}' \
  --peerAddresses localhost:7051 \
  --tlsRootCertFiles $CORE_PEER_TLS_ROOTCERT_FILE
```

### Files excluded from version control

The following paths are in `.gitignore` and must never be committed to the repository:

| Path | Reason |
|------|--------|
| `network/organizations/` | Generated crypto material and private keys |
| `network/channel-artifacts/` | Generated genesis block and channel transactions |
| `api/wallet/` | Per-user private keys |
| `api/unichain.db` | Local SQLite database |
| `api/.env` | Secrets and environment-specific configuration |