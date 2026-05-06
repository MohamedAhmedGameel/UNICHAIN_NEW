package state

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
	"github.com/unichain/chaincode/university/types"
)

func courseKey(id string) string              { return "CRS~" + id }
func courseProfListKey(profID uint64) string  { return fmt.Sprintf("CRS_PROF~%d", profID) }
func courseListKey() string                   { return "CRS_LIST" }

// ─── Course CRUD ─────────────────────────────────────────────────────────────

func CreateCourse(ctx contractapi.TransactionContextInterface, id, name string, profID uint64) error {
	id = strings.TrimSpace(id)
	name = strings.TrimSpace(name)
	if id == "" || name == "" {
		return fmt.Errorf("course id and name are required")
	}
	if !IsProfessorActive(ctx, profID) {
		return fmt.Errorf("professor %d not found or inactive", profID)
	}
	// Unique ID check
	existing, _ := ctx.GetStub().GetState(courseKey(id))
	if existing != nil {
		return fmt.Errorf("course '%s' already exists", id)
	}

	c := types.Course{
		ID:          id,
		Name:        name,
		ProfessorID: profID,
		Active:      true,
	}
	data, _ := json.Marshal(c)
	if err := ctx.GetStub().PutState(courseKey(id), data); err != nil {
		return err
	}
	// Add to professor's course list
	if err := AppendStringList(ctx, courseProfListKey(profID), id); err != nil {
		return err
	}
	return AppendStringList(ctx, courseListKey(), id)
}

func UpdateCourse(ctx contractapi.TransactionContextInterface, id, name string) error {
	c, err := GetCourse(ctx, id)
	if err != nil {
		return err
	}
	if c == nil || !c.Active {
		return fmt.Errorf("course '%s' not found", id)
	}
	if name != "" {
		c.Name = name
	}
	data, _ := json.Marshal(c)
	return ctx.GetStub().PutState(courseKey(id), data)
}

func ReassignCourse(ctx contractapi.TransactionContextInterface, id string, newProfID uint64) error {
	c, err := GetCourse(ctx, id)
	if err != nil {
		return err
	}
	if c == nil || !c.Active {
		return fmt.Errorf("course '%s' not found", id)
	}
	if !IsProfessorActive(ctx, newProfID) {
		return fmt.Errorf("professor %d not found or inactive", newProfID)
	}
	if c.ProfessorID == newProfID {
		return nil // no-op
	}
	// Remove from old professor's list
	_ = RemoveStringList(ctx, courseProfListKey(c.ProfessorID), id)
	// Add to new professor's list
	_ = AppendStringList(ctx, courseProfListKey(newProfID), id)
	c.ProfessorID = newProfID
	data, _ := json.Marshal(c)
	return ctx.GetStub().PutState(courseKey(id), data)
}

func DeleteCourse(ctx contractapi.TransactionContextInterface, id string) error {
	c, err := GetCourse(ctx, id)
	if err != nil {
		return err
	}
	if c == nil || !c.Active {
		return fmt.Errorf("course '%s' not found", id)
	}

	// Cascade: unenroll all students from this course
	enrollments, _ := GetCourseEnrollments(ctx, id)
	for _, e := range enrollments {
		if e.Active {
			_ = UnenrollStudent(ctx, e.StudentID, e.CourseID, e.Semester)
		}
	}

	// Remove from professor's list
	_ = RemoveStringList(ctx, courseProfListKey(c.ProfessorID), id)
	// Remove from global list
	_ = RemoveStringList(ctx, courseListKey(), id)
	// Mark inactive
	c.Active = false
	data, _ := json.Marshal(c)
	return ctx.GetStub().PutState(courseKey(id), data)
}

func GetCourse(ctx contractapi.TransactionContextInterface, id string) (*types.Course, error) {
	data, err := ctx.GetStub().GetState(courseKey(id))
	if err != nil {
		return nil, err
	}
	if data == nil {
		return nil, nil
	}
	var c types.Course
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

func GetAllCourses(ctx contractapi.TransactionContextInterface) ([]types.Course, error) {
	ids, err := GetStringList(ctx, courseListKey())
	if err != nil {
		return nil, err
	}
	var courses []types.Course
	for _, id := range ids {
		c, err := GetCourse(ctx, id)
		if err != nil || c == nil || !c.Active {
			continue
		}
		courses = append(courses, *c)
	}
	return courses, nil
}

func GetCoursesByProfessor(ctx contractapi.TransactionContextInterface, profID uint64) ([]types.Course, error) {
	ids, err := GetStringList(ctx, courseProfListKey(profID))
	if err != nil {
		return nil, err
	}
	var courses []types.Course
	for _, id := range ids {
		c, err := GetCourse(ctx, id)
		if err != nil || c == nil || !c.Active {
			continue
		}
		courses = append(courses, *c)
	}
	return courses, nil
}

func CourseExists(ctx contractapi.TransactionContextInterface, id string) bool {
	c, err := GetCourse(ctx, id)
	return err == nil && c != nil && c.Active
}
