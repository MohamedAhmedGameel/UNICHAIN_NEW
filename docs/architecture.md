# Architecture

---

## Network Topology

```
Windows 11
└── Docker Desktop (WSL2 backend)
    └── WSL2 Ubuntu
        ├── peer binary (fabric-samples/bin/peer)
        │
        └── Docker containers
            ├── peer0.org1.university.com   :7051
            ├── orderer.university.com      :7050 / :7053
            └── ca.org1.university.com      :7054

            [auto-created on first invoke]
            └── dev-peer0-university_1.x    (chaincode process)
```

---

## Container Roles

| Container | Image | Ports | Role |
|-----------|-------|-------|------|
| `peer0.org1.university.com` | `fabric-peer:2.5` | 7051, 9444 | Endorses transactions, maintains ledger |
| `orderer.university.com` | `fabric-orderer:2.5` | 7050, 7053, 9443 | Orders and distributes blocks to peers |
| `ca.org1.university.com` | `fabric-ca:1.5` | 7054, 17054 | Issues X.509 identities for the org |
| `dev-peer0-university_1.x` | Auto-built Go binary | — | Runs chaincode logic, created on first invoke |

---

## Transaction Flow

```
Client (peer CLI)
    │
    │  1. Submit proposal
    ▼
peer0.org1.university.com
    │
    │  2. Forward to chaincode container
    ▼
dev-peer0-university (chaincode)
    │
    │  3. Execute function, read/write world state
    │  4. Return endorsement
    ▼
peer0.org1.university.com
    │
    │  5. Forward endorsed tx to orderer
    ▼
orderer.university.com
    │
    │  6. Order into block, deliver back to peer
    ▼
peer0.org1.university.com
    │
    │  7. Validate and commit block to ledger
    ▼
Ledger (LevelDB)
```

---

## Chaincode Internal Structure

```
chaincode.go          ← All public contract methods (entry points)
    │
    ├── rbac/
    │   ├── rbac.go   ← Identity, role/permission CRUD, access guards
    │   └── seed.go   ← Default data seeded on InitLedger
    │
    ├── state/
    │   ├── major.go      ← Major CRUD, counter helper, list helpers
    │   ├── professor.go  ← Professor CRUD
    │   ├── student.go    ← Student CRUD
    │   ├── course.go     ← Course CRUD
    │   └── enrollment.go ← Enrollment CRUD, marks, GPA calculation
    │
    └── types/
        └── models.go  ← All struct definitions (Major, Student, Course, etc.)
```

---

## Identity Model

Fabric does not have wallets or private keys in the chaincode layer.  
Identity is derived from the caller's X.509 certificate at transaction time:

```
X.509 Certificate (from MSP)
    │
    ▼
ctx.GetClientIdentity().GetID()
    │  returns: "x509::CN=Admin@org1..."  (full subject string)
    ▼
SHA256 hash
    │
    ▼
First 40 hex characters = Caller Address
    │
    ▼
Used as the key for all RBAC lookups
```

This means "address" in UniChain == a deterministic hash of the admin's X.509 cert.  
The Admin identity that runs `InitLedger` is permanently made SuperAdmin.

---

## State Database Key Schema

All data is stored as JSON values under string keys in LevelDB (the peer's state database).

### RBAC Keys

| Key Pattern | Value | Description |
|-------------|-------|-------------|
| `ROLE~{name}` | `Role` JSON | Role definition |
| `ROLE_MEMBERS~{role}~{addr}` | `"true"` | Membership marker |
| `ROLE_MEMBER_LIST~{role}` | `StringList` JSON | All members of a role |
| `USER_ROLES~{addr}~{role}` | `"true"` | Assignment marker |
| `USER_ROLE_LIST~{addr}` | `StringList` JSON | All roles of a user |
| `ROLE_PERMISSION~{role}~{perm}` | `"true"` | Permission grant marker |
| `ROLE_PERM_LIST~{role}` | `StringList` JSON | All permissions of a role |
| `PERMISSION~{name}` | `Permission` JSON | Permission definition |
| `SUPER_ADMIN` | address string | Immutable SuperAdmin address |

### Entity Keys

| Key Pattern | Value | Description |
|-------------|-------|-------------|
| `CTR~{entity}` | `Counter` JSON | Auto-increment counter per entity type |
| `MAJOR~{id}` | `Major` JSON | Major by numeric ID |
| `MAJOR_CODE~{code}` | uint64 JSON | Major ID indexed by code string |
| `MAJOR_LIST` | `Uint64List` JSON | All major IDs |
| `PROF~{id}` | `Professor` JSON | Professor by numeric ID |
| `PROF_ADDR~{addr}` | uint64 JSON | Professor ID indexed by address |
| `PROF_LIST` | `Uint64List` JSON | All professor IDs |
| `STU~{id}` | `Student` JSON | Student by numeric ID |
| `STU_ADDR~{addr}` | uint64 JSON | Student ID indexed by wallet address |
| `STU_LIST` | `Uint64List` JSON | All student IDs |
| `CRS~{id}` | `Course` JSON | Course by string ID (e.g. "CS101") |
| `CRS_PROF~{profId}` | `StringList` JSON | Course IDs assigned to a professor |
| `CRS_LIST` | `StringList` JSON | All course IDs |
| `ENR~{stuId}~{crsId}~{semester}` | `EnrollmentRecord` JSON | Enrollment record |
| `ENR_STU~{stuId}` | `StringList` JSON | All enrollment keys for a student |
| `ENR_CRS~{crsId}` | `StringList` JSON | All enrollment keys for a course |
| `ENR_SEM~{stuId}` | `StringList` JSON | All semesters a student enrolled in |
| `ENR_SEM_ENR~{stuId}~{sem}` | `StringList` JSON | Enrollment keys per student per semester |

---

## Data Types (models.go)

```go
type Major struct {
    ID          uint64 `json:"id"`
    Code        string `json:"code"`
    Name        string `json:"name"`
    Description string `json:"description"`
    Active      bool   `json:"active"`
}

type Professor struct {
    ID               uint64 `json:"id"`
    ProfessorAddress string `json:"professorAddress"`
    Name             string `json:"name"`
    Department       string `json:"department"`
    Active           bool   `json:"active"`
}

type Student struct {
    ID                 uint64 `json:"id"`
    Name               string `json:"name"`
    MajorID            uint64 `json:"majorId"`
    Year               uint64 `json:"year"`
    AcademicSupervisor string `json:"academicSupervisor"`
    WalletAddress      string `json:"walletAddress"`
    Active             bool   `json:"active"`
}

type Course struct {
    ID          string `json:"id"`
    Name        string `json:"name"`
    ProfessorID uint64 `json:"professorId"`
    Active      bool   `json:"active"`
}

type EnrollmentRecord struct {
    StudentID string `json:"studentId"`
    CourseID  string `json:"courseId"`
    Semester  string `json:"semester"`
    Mark      int    `json:"mark"`   // 0 = not yet graded
    Active    bool   `json:"active"`
}

type Role struct {
    Name        string `json:"name"`
    Description string `json:"description"`
    Active      bool   `json:"active"`
}
```

---

## Cascade Rules

| Action | Cascades to |
|--------|-------------|
| `DeleteProfessor` | Deletes all courses owned by that professor |
| `DeleteCourse` | Unenrolls all students enrolled in that course |
| `DeleteStudent` | Unenrolls student from all active enrollments |
| `DeleteRole` | Removes all role members and permission grants |
| `DeactivateMajor` | No cascade — students keep their majorId |
