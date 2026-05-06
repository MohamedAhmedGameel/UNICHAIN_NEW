package state

import (
	"encoding/json"
	"fmt"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
	"github.com/unichain/chaincode/university/types"
)

// Enrollment key: ENR~{studentId}~{courseId}~{semester}
func enrollmentKey(studentID, courseID, semester string) string {
	return fmt.Sprintf("ENR~%s~%s~%s", studentID, courseID, semester)
}

// Index keys
func stuEnrListKey(studentID string) string             { return "ENR_STU~" + studentID }
func crsEnrListKey(courseID string) string               { return "ENR_CRS~" + courseID }
func stuSemesterListKey(studentID string) string         { return "ENR_SEM~" + studentID }
func stuSemEnrListKey(studentID, semester string) string { return fmt.Sprintf("ENR_SEM_ENR~%s~%s", studentID, semester) }

// ─── Enrollment CRUD ─────────────────────────────────────────────────────────

func EnrollStudent(ctx contractapi.TransactionContextInterface, studentID, courseID, semester string) error {
	if studentID == "" || courseID == "" || semester == "" {
		return fmt.Errorf("studentId, courseId, and semester are required")
	}

	// Parse studentID to uint64 for active check
	var sid uint64
	if _, err := fmt.Sscanf(studentID, "%d", &sid); err != nil {
		return fmt.Errorf("invalid studentId: %s", studentID)
	}

	if !IsStudentActive(ctx, sid) {
		return fmt.Errorf("student '%s' not found or inactive", studentID)
	}
	if !CourseExists(ctx, courseID) {
		return fmt.Errorf("course '%s' not found or inactive", courseID)
	}

	key := enrollmentKey(studentID, courseID, semester)
	existing, _ := ctx.GetStub().GetState(key)
	if existing != nil {
		var e types.EnrollmentRecord
		_ = json.Unmarshal(existing, &e)
		if e.Active {
			return fmt.Errorf("student '%s' already enrolled in '%s' for '%s'", studentID, courseID, semester)
		}
	}

	enr := types.EnrollmentRecord{
		StudentID: studentID,
		CourseID:  courseID,
		Semester:  semester,
		Mark:      0,
		Active:    true,
	}
	data, _ := json.Marshal(enr)
	if err := ctx.GetStub().PutState(key, data); err != nil {
		return err
	}

	// Maintain indexes
	_ = AppendStringList(ctx, stuEnrListKey(studentID), key)
	_ = AppendStringList(ctx, crsEnrListKey(courseID), key)
	_ = AppendStringList(ctx, stuSemEnrListKey(studentID, semester), key)

	// Track distinct semesters for student
	_ = AppendStringList(ctx, stuSemesterListKey(studentID), semester)
	return nil
}

func BatchEnroll(ctx contractapi.TransactionContextInterface, studentIDs []string, courseID, semester string) error {
	if !CourseExists(ctx, courseID) {
		return fmt.Errorf("course '%s' not found", courseID)
	}
	for _, sid := range studentIDs {
		var id uint64
		if _, err := fmt.Sscanf(sid, "%d", &id); err != nil {
			continue
		}
		if !IsStudentActive(ctx, id) {
			continue
		}
		key := enrollmentKey(sid, courseID, semester)
		existing, _ := ctx.GetStub().GetState(key)
		if existing != nil {
			var e types.EnrollmentRecord
			_ = json.Unmarshal(existing, &e)
			if e.Active {
				continue // skip already enrolled
			}
		}
		_ = EnrollStudent(ctx, sid, courseID, semester)
	}
	return nil
}

func UnenrollStudent(ctx contractapi.TransactionContextInterface, studentID, courseID, semester string) error {
	key := enrollmentKey(studentID, courseID, semester)
	data, err := ctx.GetStub().GetState(key)
	if err != nil {
		return err
	}
	if data == nil {
		return nil // already unenrolled — idempotent
	}
	var enr types.EnrollmentRecord
	if err := json.Unmarshal(data, &enr); err != nil {
		return err
	}
	if !enr.Active {
		return nil
	}
	enr.Active = false
	out, _ := json.Marshal(enr)
	if err := ctx.GetStub().PutState(key, out); err != nil {
		return err
	}
	// Remove from all indexes
	_ = RemoveStringList(ctx, stuEnrListKey(studentID), key)
	_ = RemoveStringList(ctx, crsEnrListKey(courseID), key)
	_ = RemoveStringList(ctx, stuSemEnrListKey(studentID, semester), key)
	return nil
}

func UpdateMark(ctx contractapi.TransactionContextInterface, studentID, courseID, semester string, mark int) error {
	if mark < 0 || mark > 100 {
		return fmt.Errorf("mark must be 0–100, got %d", mark)
	}
	key := enrollmentKey(studentID, courseID, semester)
	data, err := ctx.GetStub().GetState(key)
	if err != nil {
		return err
	}
	if data == nil {
		return fmt.Errorf("enrollment not found for student '%s', course '%s', semester '%s'", studentID, courseID, semester)
	}
	var enr types.EnrollmentRecord
	if err := json.Unmarshal(data, &enr); err != nil {
		return err
	}
	if !enr.Active {
		return fmt.Errorf("enrollment is not active")
	}
	enr.Mark = mark
	out, _ := json.Marshal(enr)
	return ctx.GetStub().PutState(key, out)
}

// ─── Enrollment reads ─────────────────────────────────────────────────────────

func GetStudentEnrollments(ctx contractapi.TransactionContextInterface, studentID string) ([]types.EnrollmentRecord, error) {
	return resolveEnrollmentKeys(ctx, stuEnrListKey(studentID))
}

func GetSemesterEnrollments(ctx contractapi.TransactionContextInterface, studentID, semester string) ([]types.EnrollmentRecord, error) {
	return resolveEnrollmentKeys(ctx, stuSemEnrListKey(studentID, semester))
}

func GetCourseEnrollments(ctx contractapi.TransactionContextInterface, courseID string) ([]types.EnrollmentRecord, error) {
	return resolveEnrollmentKeys(ctx, crsEnrListKey(courseID))
}

func GetStudentSemesters(ctx contractapi.TransactionContextInterface, studentID string) ([]string, error) {
	return GetStringList(ctx, stuSemesterListKey(studentID))
}

// resolveEnrollmentKeys loads all enrollment records from a key-list index.
func resolveEnrollmentKeys(ctx contractapi.TransactionContextInterface, listKey string) ([]types.EnrollmentRecord, error) {
	keys, err := GetStringList(ctx, listKey)
	if err != nil {
		return nil, err
	}
	var records []types.EnrollmentRecord
	for _, k := range keys {
		data, err := ctx.GetStub().GetState(k)
		if err != nil || data == nil {
			continue
		}
		var enr types.EnrollmentRecord
		if err := json.Unmarshal(data, &enr); err != nil {
			continue
		}
		records = append(records, enr)
	}
	return records, nil
}

// ─── GPA helpers ─────────────────────────────────────────────────────────────

// CalculateGPA returns (totalMarks, gradedCount) for client-side 4.0 conversion.
func CalculateGPA(ctx contractapi.TransactionContextInterface, studentID string) (int, int, error) {
	records, err := GetStudentEnrollments(ctx, studentID)
	if err != nil {
		return 0, 0, err
	}
	total, count := 0, 0
	for _, r := range records {
		if r.Active && r.Mark > 0 {
			total += r.Mark
			count++
		}
	}
	return total, count, nil
}
