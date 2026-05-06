package types

// ─── Counter ─────────────────────────────────────────────────────────────────

// Counter tracks auto-increment IDs per entity type.
type Counter struct {
	Entity string `json:"entity"`
	NextID uint64 `json:"nextId"`
}

// ─── Major ───────────────────────────────────────────────────────────────────

type Major struct {
	ID          uint64 `json:"id"`
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Active      bool   `json:"active"`
}

// ─── Professor ───────────────────────────────────────────────────────────────

type Professor struct {
	ID               uint64 `json:"id"`
	ProfessorAddress string `json:"professorAddress"` // cert-derived address
	Name             string `json:"name"`
	Department       string `json:"department"`
	Active           bool   `json:"active"`
}

// ─── Student ─────────────────────────────────────────────────────────────────

type Student struct {
	ID                  uint64 `json:"id"`
	Name                string `json:"name"`
	MajorID             uint64 `json:"majorId"`
	Year                uint64 `json:"year"`
	AcademicSupervisor  string `json:"academicSupervisor"` // professor's address
	WalletAddress       string `json:"walletAddress"`       // student's own address
	Active              bool   `json:"active"`
}

// ─── Course ──────────────────────────────────────────────────────────────────

type Course struct {
	ID          string `json:"id"` // string key like "CS101"
	Name        string `json:"name"`
	ProfessorID uint64 `json:"professorId"`
	Active      bool   `json:"active"`
}

// ─── Enrollment ──────────────────────────────────────────────────────────────

type EnrollmentRecord struct {
	StudentID string `json:"studentId"`
	CourseID  string `json:"courseId"`
	Semester  string `json:"semester"`
	Mark      int    `json:"mark"` // 0–100; 0 = not yet graded
	Active    bool   `json:"active"`
}

// ─── RBAC ────────────────────────────────────────────────────────────────────

type Role struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Active      bool   `json:"active"`
}

type Permission struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// ─── Lists (stored as JSON arrays in state) ──────────────────────────────────

type StringList struct {
	Items []string `json:"items"`
}

type Uint64List struct {
	Items []uint64 `json:"items"`
}
