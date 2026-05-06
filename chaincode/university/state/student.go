package state

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
	"github.com/unichain/chaincode/university/types"
)

func stuKey(id uint64) string         { return fmt.Sprintf("STU~%d", id) }
func stuAddrKey(addr string) string   { return "STU_ADDR~" + strings.ToLower(addr) }
func stuListKey() string              { return "STU_LIST" }

// ─── Student CRUD ────────────────────────────────────────────────────────────

func AddStudent(ctx contractapi.TransactionContextInterface, name string, majorID, year uint64, supervisorAddr, walletAddr string) (uint64, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return 0, fmt.Errorf("student name is required")
	}
	if majorID == 0 {
		return 0, fmt.Errorf("majorId is required")
	}
	if year == 0 {
		return 0, fmt.Errorf("year is required")
	}
	if !IsMajorActive(ctx, majorID) {
		return 0, fmt.Errorf("major %d not found or inactive", majorID)
	}

	id, err := nextID(ctx, "student")
	if err != nil {
		return 0, err
	}

	s := types.Student{
		ID:                 id,
		Name:               name,
		MajorID:            majorID,
		Year:               year,
		AcademicSupervisor: supervisorAddr,
		WalletAddress:      walletAddr,
		Active:             true,
	}
	data, _ := json.Marshal(s)
	if err := ctx.GetStub().PutState(stuKey(id), data); err != nil {
		return 0, err
	}
	// Wallet → ID index (if provided)
	if walletAddr != "" {
		idBytes, _ := json.Marshal(id)
		_ = ctx.GetStub().PutState(stuAddrKey(walletAddr), idBytes)
	}
	return id, appendUint64List(ctx, stuListKey(), id)
}

func UpdateStudent(ctx contractapi.TransactionContextInterface, id uint64, name string, majorID, year uint64, supervisorAddr, walletAddr string) error {
	s, err := GetStudent(ctx, id)
	if err != nil {
		return err
	}
	if s == nil || !s.Active {
		return fmt.Errorf("student %d not found", id)
	}
	if name != "" {
		s.Name = name
	}
	if majorID > 0 {
		s.MajorID = majorID
	}
	if year > 0 {
		s.Year = year
	}
	if supervisorAddr != "" {
		s.AcademicSupervisor = supervisorAddr
	}
	if walletAddr != "" && walletAddr != s.WalletAddress {
		// Remove old index
		if s.WalletAddress != "" {
			_ = ctx.GetStub().DelState(stuAddrKey(s.WalletAddress))
		}
		s.WalletAddress = walletAddr
		idBytes, _ := json.Marshal(id)
		_ = ctx.GetStub().PutState(stuAddrKey(walletAddr), idBytes)
	}
	data, _ := json.Marshal(s)
	return ctx.GetStub().PutState(stuKey(id), data)
}

func DeleteStudent(ctx contractapi.TransactionContextInterface, id uint64) error {
	s, err := GetStudent(ctx, id)
	if err != nil {
		return err
	}
	if s == nil || !s.Active {
		return fmt.Errorf("student %d not found", id)
	}

	// Cascade: unenroll from all active enrollments
	idStr := fmt.Sprintf("%d", id)
	enrollments, _ := GetStudentEnrollments(ctx, idStr)
	for _, e := range enrollments {
		if e.Active {
			_ = UnenrollStudent(ctx, e.StudentID, e.CourseID, e.Semester)
		}
	}

	if s.WalletAddress != "" {
		_ = ctx.GetStub().DelState(stuAddrKey(s.WalletAddress))
	}
	s.Active = false
	data, _ := json.Marshal(s)
	_ = ctx.GetStub().PutState(stuKey(id), data)
	return removeUint64List(ctx, stuListKey(), id)
}

func GetStudent(ctx contractapi.TransactionContextInterface, id uint64) (*types.Student, error) {
	data, err := ctx.GetStub().GetState(stuKey(id))
	if err != nil {
		return nil, err
	}
	if data == nil {
		return nil, nil
	}
	var s types.Student
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

func GetAllStudents(ctx contractapi.TransactionContextInterface) ([]types.Student, error) {
	ids, err := getUint64List(ctx, stuListKey())
	if err != nil {
		return nil, err
	}
	var students []types.Student
	for _, id := range ids {
		s, err := GetStudent(ctx, id)
		if err != nil || s == nil || !s.Active {
			continue
		}
		students = append(students, *s)
	}
	return students, nil
}

func IsStudentActive(ctx contractapi.TransactionContextInterface, id uint64) bool {
	s, err := GetStudent(ctx, id)
	return err == nil && s != nil && s.Active
}
