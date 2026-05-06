# RBAC — Roles & Permissions

---

## How Identity Works

UniChain doesn't use passwords or API keys. Identity comes from the X.509 certificate embedded in the Fabric MSP (Membership Service Provider).

When you submit a transaction, the chaincode hashes your certificate to a 40-character hex address:

```
Your X.509 cert subject
    ↓
SHA256 hash
    ↓
First 40 hex characters = Your "address"
    ↓
Used as key for all role/permission lookups
```

The Admin identity in `CORE_PEER_MSPCONFIGPATH` is whoever you are when you run commands. The identity that ran `InitLedger` is permanently registered as SuperAdmin.

### Find Your Address

```bash
peer chaincode query -C universitychannel -n university \
  -c '{"function":"GetCallerAddress","Args":[]}'
```

---

## Default Roles

Seeded automatically by `InitLedger`:

### SuperAdmin
- Has all permissions
- Granted to whoever ran `InitLedger` (the deployer)
- The address is locked permanently with `SetSuperAdmin`
- Cannot be revoked

### Registrar
- Manages professors, students, courses, enrollment
- Can grade any course (`mark:any`)

### Instructor
- Can only grade their own courses (`mark:own`)
- The caller's address must match the professor's `professorAddress` field

---

## Full Permission List

| Permission | Description |
|------------|-------------|
| `major:create` | Create academic majors |
| `major:update` | Update major details |
| `major:deactivate` | Deactivate a major |
| `professor:create` | Register a professor |
| `professor:update` | Update professor details |
| `professor:delete` | Delete a professor |
| `student:create` | Register a student |
| `student:update` | Update student details |
| `student:delete` | Delete a student |
| `course:create` | Create a course |
| `course:update` | Update course name |
| `course:delete` | Delete a course |
| `course:reassign` | Move course to another professor |
| `enrollment:enroll` | Enroll a student |
| `enrollment:unenroll` | Unenroll a student |
| `enrollment:batch` | Batch enroll multiple students |
| `mark:any` | Grade any course |
| `mark:own` | Grade only your own courses |
| `role:manage` | Create/delete roles, assign permissions and members |
| `permission:manage` | Create new permission types |

---

## Role → Permission Matrix

| Permission | SuperAdmin | Registrar | Instructor |
|------------|:---:|:---:|:---:|
| major:create | ✓ | | |
| major:update | ✓ | | |
| major:deactivate | ✓ | | |
| professor:create | ✓ | ✓ | |
| professor:update | ✓ | ✓ | |
| professor:delete | ✓ | | |
| student:create | ✓ | ✓ | |
| student:update | ✓ | ✓ | |
| student:delete | ✓ | ✓ | |
| course:create | ✓ | ✓ | |
| course:update | ✓ | ✓ | |
| course:delete | ✓ | ✓ | |
| course:reassign | ✓ | ✓ | |
| enrollment:enroll | ✓ | ✓ | |
| enrollment:unenroll | ✓ | ✓ | |
| enrollment:batch | ✓ | ✓ | |
| mark:any | ✓ | ✓ | |
| mark:own | ✓ | | ✓ |
| role:manage | ✓ | | |
| permission:manage | ✓ | | |

---

## Assigning Roles

### Step 1 — Get the user's address

The user must run this with their own identity:

```bash
peer chaincode query -C universitychannel -n university \
  -c '{"function":"GetCallerAddress","Args":[]}'
# Returns: "a3f2bc8d1e4f5a6b7c8d9e0f1a2b3c4d5e6f7a8b"
```

### Step 2 — SuperAdmin assigns the role

```bash
peer chaincode invoke \
  -o localhost:7050 --ordererTLSHostnameOverride orderer.university.com \
  --tls --cafile $ORDERER_CA \
  -C universitychannel -n university \
  -c '{"function":"AssignRoleToUser","Args":["a3f2bc8d1e4f5a6b7c8d9e0f1a2b3c4d5e6f7a8b","Registrar"]}' \
  --peerAddresses localhost:7051 \
  --tlsRootCertFiles $CORE_PEER_TLS_ROOTCERT_FILE
```

### Step 3 — Verify

```bash
peer chaincode query -C universitychannel -n university \
  -c '{"function":"GetUserRoles","Args":["a3f2bc8d1e4f5a6b7c8d9e0f1a2b3c4d5e6f7a8b"]}'
# Returns: ["Registrar"]
```

---

## Revoking Roles

```bash
peer chaincode invoke ... \
  -c '{"function":"RevokeRoleFromUser","Args":["<address>","Registrar"]}'
```

> SuperAdmin cannot be revoked from their own address.

---

## Creating Custom Roles

```bash
# 1. Create the role
peer chaincode invoke ... -c '{"function":"CreateRole","Args":["Dean","Dean of the faculty"]}'

# 2. Assign it to a user
peer chaincode invoke ... -c '{"function":"AssignRoleToUser","Args":["<address>","Dean"]}'
```

Custom roles have no permissions by default. Currently permissions can only be granted programmatically (via `GrantPermissionToRole` in rbac.go). Adding this as a contract method is a future enhancement.

---

## Instructor Mark:Own Flow

For an instructor to grade their own courses:

1. Register the instructor as a professor with their actual caller address:
```bash
# Get instructor's address first
peer chaincode query ... -c '{"function":"GetCallerAddress","Args":[]}'

# Register them as a professor using that exact address
peer chaincode invoke ... -c '{"function":"AddProfessor","Args":["Dr. Jones","Physics","<their-address>"]}'
```

2. Assign them the Instructor role:
```bash
peer chaincode invoke ... -c '{"function":"AssignRoleToUser","Args":["<their-address>","Instructor"]}'
```

3. Assign courses to that professor ID.

4. When they invoke `UpdateMark`, the chaincode:
   - Checks they have `mark:own`
   - Looks up the course's `professorId`
   - Looks up the professor by their caller address
   - Confirms the IDs match
   - Only then allows the grade

---

## Security Notes

- All permission checks happen inside the chaincode — they cannot be bypassed from the outside
- The SuperAdmin address is written to state during `InitLedger` and never changes
- There is no "password reset" — if the Admin MSP cert is lost, SuperAdmin access is gone
- All transactions are signed by the submitter's X.509 cert and recorded on the immutable ledger
