package main

import (
	"encoding/json"
	"fmt"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"

	"github.com/unichain/chaincode/university/rbac"
	"github.com/unichain/chaincode/university/state"
	"github.com/unichain/chaincode/university/types"
)

type UniversityChaincode struct {
	contractapi.Contract
}

type GPAResult struct {
	TotalMarks  int     `json:"totalMarks"`
	GradedCount int     `json:"gradedCount"`
	Average     float64 `json:"average"`
}

func (c *UniversityChaincode) InitLedger(ctx contractapi.TransactionContextInterface) error {
	return rbac.SeedDefaults(ctx)
}

func (c *UniversityChaincode) CreateRole(ctx contractapi.TransactionContextInterface, roleName string, description string) error {
	if err := rbac.RequirePermission(ctx, "role:manage"); err != nil { return err }
	return rbac.CreateRole(ctx, roleName, description)
}
func (c *UniversityChaincode) GetRole(ctx contractapi.TransactionContextInterface, roleName string) (*types.Role, error) {
	return rbac.GetRole(ctx, roleName)
}
func (c *UniversityChaincode) GetAllRoles(ctx contractapi.TransactionContextInterface) ([]types.Role, error) {
	return rbac.GetAllRoles(ctx)
}
func (c *UniversityChaincode) DeleteRole(ctx contractapi.TransactionContextInterface, roleName string) error {
	if err := rbac.RequirePermission(ctx, "role:manage"); err != nil { return err }
	return rbac.DeleteRole(ctx, roleName)
}
func (c *UniversityChaincode) AssignRoleToUser(ctx contractapi.TransactionContextInterface, userID string, roleName string) error {
	if err := rbac.RequirePermission(ctx, "role:manage"); err != nil { return err }
	return rbac.GrantRole(ctx, roleName, userID)
}
func (c *UniversityChaincode) RevokeRoleFromUser(ctx contractapi.TransactionContextInterface, userID string, roleName string) error {
	if err := rbac.RequirePermission(ctx, "role:manage"); err != nil { return err }
	return rbac.RevokeRole(ctx, roleName, userID)
}
func (c *UniversityChaincode) GetUserRoles(ctx contractapi.TransactionContextInterface, userID string) ([]string, error) {
	return rbac.GetUserRoles(ctx, userID)
}
func (c *UniversityChaincode) GetCallerAddress(ctx contractapi.TransactionContextInterface) (string, error) {
	return rbac.GetCallerAddress(ctx)
}

func (c *UniversityChaincode) AddMajor(ctx contractapi.TransactionContextInterface, code string, name string, description string) (uint64, error) {
	if err := rbac.RequirePermission(ctx, "major:create"); err != nil { return 0, err }
	return state.AddMajor(ctx, code, name, description)
}
func (c *UniversityChaincode) UpdateMajor(ctx contractapi.TransactionContextInterface, id uint64, name string, description string) error {
	if err := rbac.RequirePermission(ctx, "major:update"); err != nil { return err }
	return state.UpdateMajor(ctx, id, name, description)
}
func (c *UniversityChaincode) DeactivateMajor(ctx contractapi.TransactionContextInterface, id uint64) error {
	if err := rbac.RequirePermission(ctx, "major:deactivate"); err != nil { return err }
	return state.DeactivateMajor(ctx, id)
}
func (c *UniversityChaincode) GetMajor(ctx contractapi.TransactionContextInterface, id uint64) (*types.Major, error) {
	return state.GetMajor(ctx, id)
}
func (c *UniversityChaincode) GetMajorByCode(ctx contractapi.TransactionContextInterface, code string) (*types.Major, error) {
	return state.GetMajorByCode(ctx, code)
}
func (c *UniversityChaincode) GetAllMajors(ctx contractapi.TransactionContextInterface) ([]types.Major, error) {
	return state.GetAllMajors(ctx)
}

func (c *UniversityChaincode) AddProfessor(ctx contractapi.TransactionContextInterface, name string, department string, address string) (uint64, error) {
	if err := rbac.RequirePermission(ctx, "professor:create"); err != nil { return 0, err }
	return state.AddProfessor(ctx, name, department, address)
}
func (c *UniversityChaincode) UpdateProfessor(ctx contractapi.TransactionContextInterface, id uint64, name string, department string, newAddress string) error {
	if err := rbac.RequirePermission(ctx, "professor:update"); err != nil { return err }
	return state.UpdateProfessor(ctx, id, name, department, newAddress)
}
func (c *UniversityChaincode) DeleteProfessor(ctx contractapi.TransactionContextInterface, id uint64) error {
	if err := rbac.RequirePermission(ctx, "professor:delete"); err != nil { return err }
	return state.DeleteProfessor(ctx, id)
}
func (c *UniversityChaincode) GetProfessor(ctx contractapi.TransactionContextInterface, id uint64) (*types.Professor, error) {
	return state.GetProfessor(ctx, id)
}
func (c *UniversityChaincode) GetProfessorByAddress(ctx contractapi.TransactionContextInterface, address string) (*types.Professor, error) {
	return state.GetProfessorByAddress(ctx, address)
}
func (c *UniversityChaincode) GetAllProfessors(ctx contractapi.TransactionContextInterface) ([]types.Professor, error) {
	return state.GetAllProfessors(ctx)
}

func (c *UniversityChaincode) AddStudent(ctx contractapi.TransactionContextInterface, name string, majorID uint64, year uint64, supervisorAddr string, walletAddr string) (uint64, error) {
	if err := rbac.RequirePermission(ctx, "student:create"); err != nil { return 0, err }
	return state.AddStudent(ctx, name, majorID, year, supervisorAddr, walletAddr)
}
func (c *UniversityChaincode) UpdateStudent(ctx contractapi.TransactionContextInterface, id uint64, name string, majorID uint64, year uint64, supervisorAddr string, walletAddr string) error {
	if err := rbac.RequirePermission(ctx, "student:update"); err != nil { return err }
	return state.UpdateStudent(ctx, id, name, majorID, year, supervisorAddr, walletAddr)
}
func (c *UniversityChaincode) DeleteStudent(ctx contractapi.TransactionContextInterface, id uint64) error {
	if err := rbac.RequirePermission(ctx, "student:delete"); err != nil { return err }
	return state.DeleteStudent(ctx, id)
}
func (c *UniversityChaincode) GetStudent(ctx contractapi.TransactionContextInterface, id uint64) (*types.Student, error) {
	return state.GetStudent(ctx, id)
}
func (c *UniversityChaincode) GetAllStudents(ctx contractapi.TransactionContextInterface) ([]types.Student, error) {
	return state.GetAllStudents(ctx)
}

func (c *UniversityChaincode) CreateCourse(ctx contractapi.TransactionContextInterface, id string, name string, professorID uint64) error {
	if err := rbac.RequirePermission(ctx, "course:create"); err != nil { return err }
	return state.CreateCourse(ctx, id, name, professorID)
}
func (c *UniversityChaincode) UpdateCourse(ctx contractapi.TransactionContextInterface, id string, name string) error {
	if err := rbac.RequirePermission(ctx, "course:update"); err != nil { return err }
	return state.UpdateCourse(ctx, id, name)
}
func (c *UniversityChaincode) ReassignCourse(ctx contractapi.TransactionContextInterface, id string, newProfessorID uint64) error {
	if err := rbac.RequirePermission(ctx, "course:reassign"); err != nil { return err }
	return state.ReassignCourse(ctx, id, newProfessorID)
}
func (c *UniversityChaincode) DeleteCourse(ctx contractapi.TransactionContextInterface, id string) error {
	if err := rbac.RequirePermission(ctx, "course:delete"); err != nil { return err }
	return state.DeleteCourse(ctx, id)
}
func (c *UniversityChaincode) GetCourse(ctx contractapi.TransactionContextInterface, id string) (*types.Course, error) {
	return state.GetCourse(ctx, id)
}
func (c *UniversityChaincode) GetAllCourses(ctx contractapi.TransactionContextInterface) ([]types.Course, error) {
	return state.GetAllCourses(ctx)
}
func (c *UniversityChaincode) GetCoursesByProfessor(ctx contractapi.TransactionContextInterface, professorID uint64) ([]types.Course, error) {
	return state.GetCoursesByProfessor(ctx, professorID)
}

func (c *UniversityChaincode) EnrollStudent(ctx contractapi.TransactionContextInterface, studentID string, courseID string, semester string) error {
	if err := rbac.RequirePermission(ctx, "enrollment:enroll"); err != nil { return err }
	return state.EnrollStudent(ctx, studentID, courseID, semester)
}
func (c *UniversityChaincode) BatchEnroll(ctx contractapi.TransactionContextInterface, studentIDsJSON string, courseID string, semester string) error {
	if err := rbac.RequirePermission(ctx, "enrollment:batch"); err != nil { return err }
	var ids []string
	if err := json.Unmarshal([]byte(studentIDsJSON), &ids); err != nil {
		return fmt.Errorf("invalid studentIds JSON: %w", err)
	}
	return state.BatchEnroll(ctx, ids, courseID, semester)
}
func (c *UniversityChaincode) UnenrollStudent(ctx contractapi.TransactionContextInterface, studentID string, courseID string, semester string) error {
	if err := rbac.RequirePermission(ctx, "enrollment:unenroll"); err != nil { return err }
	return state.UnenrollStudent(ctx, studentID, courseID, semester)
}
func (c *UniversityChaincode) GetStudentEnrollments(ctx contractapi.TransactionContextInterface, studentID string) ([]types.EnrollmentRecord, error) {
	return state.GetStudentEnrollments(ctx, studentID)
}
func (c *UniversityChaincode) GetSemesterEnrollments(ctx contractapi.TransactionContextInterface, studentID string, semester string) ([]types.EnrollmentRecord, error) {
	return state.GetSemesterEnrollments(ctx, studentID, semester)
}
func (c *UniversityChaincode) GetCourseEnrollments(ctx contractapi.TransactionContextInterface, courseID string) ([]types.EnrollmentRecord, error) {
	return state.GetCourseEnrollments(ctx, courseID)
}
func (c *UniversityChaincode) GetStudentSemesters(ctx contractapi.TransactionContextInterface, studentID string) ([]string, error) {
	return state.GetStudentSemesters(ctx, studentID)
}

func (c *UniversityChaincode) UpdateMark(ctx contractapi.TransactionContextInterface, studentID string, courseID string, semester string, mark int) error {
	addr, err := rbac.GetCallerAddress(ctx)
	if err != nil { return err }
	if rbac.HasPermission(ctx, addr, "mark:any") {
		return state.UpdateMark(ctx, studentID, courseID, semester, mark)
	}
	if rbac.HasPermission(ctx, addr, "mark:own") {
		course, err := state.GetCourse(ctx, courseID)
		if err != nil || course == nil { return fmt.Errorf("course '%s' not found", courseID) }
		prof, err := state.GetProfessorByAddress(ctx, addr)
		if err != nil || prof == nil { return fmt.Errorf("no professor record for caller") }
		if course.ProfessorID != prof.ID { return fmt.Errorf("forbidden: not instructor of '%s'", courseID) }
		return state.UpdateMark(ctx, studentID, courseID, semester, mark)
	}
	return fmt.Errorf("forbidden: missing mark:any or mark:own")
}

func (c *UniversityChaincode) GetStudentGPA(ctx contractapi.TransactionContextInterface, studentID string) (*GPAResult, error) {
	total, count, err := state.CalculateGPA(ctx, studentID)
	if err != nil { return nil, err }
	avg := 0.0
	if count > 0 { avg = float64(total) / float64(count) }
	return &GPAResult{TotalMarks: total, GradedCount: count, Average: avg}, nil
}
