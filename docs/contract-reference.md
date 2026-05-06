# Contract Reference

All callable chaincode functions with exact commands.

---

## Command Templates

```bash
# INVOKE — writes to ledger, goes through orderer (~2–3 seconds)
peer chaincode invoke \
  -o localhost:7050 --ordererTLSHostnameOverride orderer.university.com \
  --tls --cafile $ORDERER_CA \
  -C universitychannel -n university \
  -c '{"function":"FUNCTION_NAME","Args":["arg1","arg2"]}' \
  --peerAddresses localhost:7051 \
  --tlsRootCertFiles $CORE_PEER_TLS_ROOTCERT_FILE

# QUERY — reads only, hits peer directly (instant)
peer chaincode query \
  -C universitychannel -n university \
  -c '{"function":"FUNCTION_NAME","Args":["arg1"]}'
```

> All arguments must be strings. Pass numbers as `"1"`, booleans as `"true"`.

---

## INIT

### `InitLedger`
Seeds default roles, permissions, and makes the caller SuperAdmin.  
**Run once only — never run again after upgrades.**

```bash
peer chaincode invoke \
  -o localhost:7050 --ordererTLSHostnameOverride orderer.university.com \
  --tls --cafile $ORDERER_CA \
  -C universitychannel -n university --isInit \
  -c '{"function":"InitLedger","Args":[]}' \
  --peerAddresses localhost:7051 \
  --tlsRootCertFiles $CORE_PEER_TLS_ROOTCERT_FILE
```

---

## ROLES

### `CreateRole` — Invoke
Requires: `role:manage`

| Arg | Type | Example |
|-----|------|---------|
| roleName | string | `"Dean"` |
| description | string | `"Dean of faculty"` |

```bash
peer chaincode invoke ... -c '{"function":"CreateRole","Args":["Dean","Dean of faculty"]}'
```

---

### `GetRole` — Query

| Arg | Type | Example |
|-----|------|---------|
| roleName | string | `"Registrar"` |

```bash
peer chaincode query ... -c '{"function":"GetRole","Args":["Registrar"]}'
```

Returns:
```json
{"name":"Registrar","description":"Student, course, and enrollment management","active":true}
```

---

### `GetAllRoles` — Query

```bash
peer chaincode query ... -c '{"function":"GetAllRoles","Args":[]}'
```

---

### `DeleteRole` — Invoke
Requires: `role:manage`  
Cascades: removes all member assignments and permission grants.

```bash
peer chaincode invoke ... -c '{"function":"DeleteRole","Args":["Dean"]}'
```

---

### `AssignRoleToUser` — Invoke
Requires: `role:manage`

| Arg | Type | Example |
|-----|------|---------|
| userID | address string | `"a3f2bc..."` |
| roleName | string | `"Registrar"` |

```bash
# First get the user's address
peer chaincode query ... -c '{"function":"GetCallerAddress","Args":[]}'

# Then assign
peer chaincode invoke ... -c '{"function":"AssignRoleToUser","Args":["<address>","Registrar"]}'
```

---

### `RevokeRoleFromUser` — Invoke
Requires: `role:manage`

```bash
peer chaincode invoke ... -c '{"function":"RevokeRoleFromUser","Args":["<address>","Registrar"]}'
```

---

### `GetUserRoles` — Query

```bash
peer chaincode query ... -c '{"function":"GetUserRoles","Args":["<address>"]}'
```

Returns: `["SuperAdmin","Registrar"]`

---

### `GetCallerAddress` — Query
Returns the deterministic address of whoever is submitting the transaction.

```bash
peer chaincode query ... -c '{"function":"GetCallerAddress","Args":[]}'
```

Returns: `"a3f2bc8d1e4f5a6b7c8d9e0f1a2b3c4d5e6f7a8b"`

---

## MAJORS

### `AddMajor` — Invoke
Requires: `major:create`  
Returns: assigned numeric ID.

| Arg | Type | Example |
|-----|------|---------|
| code | string (uppercase) | `"CS"` |
| name | string | `"Computer Science"` |
| description | string | `"Core CS program"` |

```bash
peer chaincode invoke ... -c '{"function":"AddMajor","Args":["CS","Computer Science","Core CS program"]}'
peer chaincode invoke ... -c '{"function":"AddMajor","Args":["EE","Electrical Engineering","Power systems"]}'
peer chaincode invoke ... -c '{"function":"AddMajor","Args":["ME","Mechanical Engineering",""]}'
```

---

### `UpdateMajor` — Invoke
Requires: `major:update`  
Pass empty string `""` for fields you don't want to change.

| Arg | Type | Example |
|-----|------|---------|
| id | uint64 as string | `"1"` |
| name | string | `"CS & AI"` |
| description | string | `"Updated"` |

```bash
peer chaincode invoke ... -c '{"function":"UpdateMajor","Args":["1","CS & Artificial Intelligence",""]}'
```

---

### `DeactivateMajor` — Invoke
Requires: `major:deactivate`  
Sets `active: false`. Excluded from `GetAllMajors` after this.

```bash
peer chaincode invoke ... -c '{"function":"DeactivateMajor","Args":["1"]}'
```

---

### `GetMajor` — Query

```bash
peer chaincode query ... -c '{"function":"GetMajor","Args":["1"]}'
```

---

### `GetMajorByCode` — Query

```bash
peer chaincode query ... -c '{"function":"GetMajorByCode","Args":["CS"]}'
```

---

### `GetAllMajors` — Query
Returns only active majors.

```bash
peer chaincode query ... -c '{"function":"GetAllMajors","Args":[]}'
```

---

## PROFESSORS

### `AddProfessor` — Invoke
Requires: `professor:create`  
Returns: assigned numeric ID.

| Arg | Type | Example |
|-----|------|---------|
| name | string | `"Dr. Smith"` |
| department | string | `"Computer Science"` |
| address | string | any unique identifier |

```bash
peer chaincode invoke ... -c '{"function":"AddProfessor","Args":["Dr. Smith","Computer Science","0xprof1"]}'
```

---

### `UpdateProfessor` — Invoke
Requires: `professor:update`  
Pass empty string `""` for fields you don't want to change.

```bash
peer chaincode invoke ... -c '{"function":"UpdateProfessor","Args":["1","Dr. John Smith","CS & AI",""]}'
```

---

### `DeleteProfessor` — Invoke
Requires: `professor:delete`  
Cascades: deletes all courses owned by this professor (which cascades to unenroll students).

```bash
peer chaincode invoke ... -c '{"function":"DeleteProfessor","Args":["1"]}'
```

---

### `GetProfessor` — Query

```bash
peer chaincode query ... -c '{"function":"GetProfessor","Args":["1"]}'
```

---

### `GetProfessorByAddress` — Query

```bash
peer chaincode query ... -c '{"function":"GetProfessorByAddress","Args":["0xprof1"]}'
```

---

### `GetAllProfessors` — Query

```bash
peer chaincode query ... -c '{"function":"GetAllProfessors","Args":[]}'
```

---

## STUDENTS

### `AddStudent` — Invoke
Requires: `student:create`  
Returns: assigned numeric ID.

| Arg | Type | Example |
|-----|------|---------|
| name | string | `"Alice"` |
| majorId | uint64 as string | `"1"` |
| year | uint64 as string | `"2"` |
| supervisorAddr | string | professor's address or `""` |
| walletAddr | string | student's unique address or `""` |

```bash
peer chaincode invoke ... -c '{"function":"AddStudent","Args":["Alice","1","2","0xprof1","0xalice"]}'
peer chaincode invoke ... -c '{"function":"AddStudent","Args":["Bob","1","1","",""]}'
```

---

### `UpdateStudent` — Invoke
Requires: `student:update`  
Pass `"0"` for numeric fields and `""` for string fields you don't want to change.

```bash
peer chaincode invoke ... -c '{"function":"UpdateStudent","Args":["1","Alice Smith","0","3","",""]}'
```

---

### `DeleteStudent` — Invoke
Requires: `student:delete`  
Cascades: unenrolls from all active enrollments.

```bash
peer chaincode invoke ... -c '{"function":"DeleteStudent","Args":["1"]}'
```

---

### `GetStudent` — Query

```bash
peer chaincode query ... -c '{"function":"GetStudent","Args":["1"]}'
```

---

### `GetAllStudents` — Query

```bash
peer chaincode query ... -c '{"function":"GetAllStudents","Args":[]}'
```

---

## COURSES

### `CreateCourse` — Invoke
Requires: `course:create`  
Course ID is a string like `"CS101"` — you define it, it must be unique.

| Arg | Type | Example |
|-----|------|---------|
| id | string | `"CS101"` |
| name | string | `"Intro to Programming"` |
| professorId | uint64 as string | `"1"` |

```bash
peer chaincode invoke ... -c '{"function":"CreateCourse","Args":["CS101","Intro to Programming","1"]}'
peer chaincode invoke ... -c '{"function":"CreateCourse","Args":["CS201","Data Structures","1"]}'
peer chaincode invoke ... -c '{"function":"CreateCourse","Args":["EE101","Circuit Theory","2"]}'
```

---

### `UpdateCourse` — Invoke
Requires: `course:update`  
Only updates the name. To change professor, use `ReassignCourse`.

```bash
peer chaincode invoke ... -c '{"function":"UpdateCourse","Args":["CS101","Introduction to Programming & Algorithms"]}'
```

---

### `ReassignCourse` — Invoke
Requires: `course:reassign`

```bash
peer chaincode invoke ... -c '{"function":"ReassignCourse","Args":["CS101","2"]}'
```

---

### `DeleteCourse` — Invoke
Requires: `course:delete`  
Cascades: unenrolls all students enrolled in this course.

```bash
peer chaincode invoke ... -c '{"function":"DeleteCourse","Args":["CS101"]}'
```

---

### `GetCourse` — Query

```bash
peer chaincode query ... -c '{"function":"GetCourse","Args":["CS101"]}'
```

---

### `GetAllCourses` — Query

```bash
peer chaincode query ... -c '{"function":"GetAllCourses","Args":[]}'
```

---

### `GetCoursesByProfessor` — Query

```bash
peer chaincode query ... -c '{"function":"GetCoursesByProfessor","Args":["1"]}'
```

---

## ENROLLMENT

### `EnrollStudent` — Invoke
Requires: `enrollment:enroll`

| Arg | Type | Example |
|-----|------|---------|
| studentId | numeric string | `"1"` |
| courseId | string | `"CS101"` |
| semester | string | `"2024-Fall"` |

```bash
peer chaincode invoke ... -c '{"function":"EnrollStudent","Args":["1","CS101","2024-Fall"]}'
```

---

### `BatchEnroll` — Invoke
Requires: `enrollment:batch`  
First arg is a JSON array of student ID strings.

```bash
peer chaincode invoke ... -c '{"function":"BatchEnroll","Args":["[\"1\",\"2\",\"3\"]","CS101","2024-Fall"]}'
```

---

### `UnenrollStudent` — Invoke
Requires: `enrollment:unenroll`

```bash
peer chaincode invoke ... -c '{"function":"UnenrollStudent","Args":["1","CS101","2024-Fall"]}'
```

---

### `GetStudentEnrollments` — Query

```bash
peer chaincode query ... -c '{"function":"GetStudentEnrollments","Args":["1"]}'
```

---

### `GetSemesterEnrollments` — Query

```bash
peer chaincode query ... -c '{"function":"GetSemesterEnrollments","Args":["1","2024-Fall"]}'
```

---

### `GetCourseEnrollments` — Query

```bash
peer chaincode query ... -c '{"function":"GetCourseEnrollments","Args":["CS101"]}'
```

---

### `GetStudentSemesters` — Query
Returns a list of all distinct semesters the student has been enrolled in.

```bash
peer chaincode query ... -c '{"function":"GetStudentSemesters","Args":["1"]}'
```

Returns: `["2024-Fall","2025-Spring"]`

---

## MARKS & GPA

### `UpdateMark` — Invoke
Mark must be 0–100.

Requires: `mark:any` (Registrar, SuperAdmin) OR `mark:own` (Instructor grading their own course only).

| Arg | Type | Example |
|-----|------|---------|
| studentId | numeric string | `"1"` |
| courseId | string | `"CS101"` |
| semester | string | `"2024-Fall"` |
| mark | int as string | `"87"` |

```bash
peer chaincode invoke ... -c '{"function":"UpdateMark","Args":["1","CS101","2024-Fall","87"]}'
```

---

### `GetStudentGPA` — Query
Returns total marks, count of graded courses, and numeric average.

```bash
peer chaincode query ... -c '{"function":"GetStudentGPA","Args":["1"]}'
```

Returns:
```json
{"totalMarks":174,"gradedCount":2,"average":87}
```

---

## Full Example Workflow

```bash
# 1. Add a major
peer chaincode invoke ... -c '{"function":"AddMajor","Args":["CS","Computer Science","Core CS"]}'

# 2. Add a professor
peer chaincode invoke ... -c '{"function":"AddProfessor","Args":["Dr. Smith","CS","0xprof1"]}'

# 3. Add students
peer chaincode invoke ... -c '{"function":"AddStudent","Args":["Alice","1","1","0xprof1","0xalice"]}'
peer chaincode invoke ... -c '{"function":"AddStudent","Args":["Bob","1","1","0xprof1","0xbob"]}'

# 4. Create a course
peer chaincode invoke ... -c '{"function":"CreateCourse","Args":["CS101","Intro to Programming","1"]}'

# 5. Enroll students
peer chaincode invoke ... -c '{"function":"EnrollStudent","Args":["1","CS101","2024-Fall"]}'
peer chaincode invoke ... -c '{"function":"EnrollStudent","Args":["2","CS101","2024-Fall"]}'

# 6. Assign marks
peer chaincode invoke ... -c '{"function":"UpdateMark","Args":["1","CS101","2024-Fall","90"]}'
peer chaincode invoke ... -c '{"function":"UpdateMark","Args":["2","CS101","2024-Fall","75"]}'

# 7. Check GPA
peer chaincode query ... -c '{"function":"GetStudentGPA","Args":["1"]}'
peer chaincode query ... -c '{"function":"GetStudentGPA","Args":["2"]}'

# 8. See all enrollments for course
peer chaincode query ... -c '{"function":"GetCourseEnrollments","Args":["CS101"]}'
```
