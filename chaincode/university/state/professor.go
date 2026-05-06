package state

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
	"github.com/unichain/chaincode/university/types"
)

func profKey(id uint64) string         { return fmt.Sprintf("PROF~%d", id) }
func profAddrKey(addr string) string   { return "PROF_ADDR~" + strings.ToLower(addr) }
func profListKey() string              { return "PROF_LIST" }

// ─── Professor CRUD ──────────────────────────────────────────────────────────

func AddProfessor(ctx contractapi.TransactionContextInterface, name, department, address string) (uint64, error) {
	name = strings.TrimSpace(name)
	address = strings.TrimSpace(address)
	if name == "" {
		return 0, fmt.Errorf("professor name is required")
	}
	if address == "" {
		return 0, fmt.Errorf("professor address is required")
	}

	// Unique address check
	existing, _ := ctx.GetStub().GetState(profAddrKey(address))
	if existing != nil {
		return 0, fmt.Errorf("professor address '%s' already registered", address)
	}

	id, err := nextID(ctx, "professor")
	if err != nil {
		return 0, err
	}

	p := types.Professor{
		ID:               id,
		ProfessorAddress: address,
		Name:             name,
		Department:       department,
		Active:           true,
	}
	data, _ := json.Marshal(p)
	if err := ctx.GetStub().PutState(profKey(id), data); err != nil {
		return 0, err
	}
	// Address → ID index
	idBytes, _ := json.Marshal(id)
	if err := ctx.GetStub().PutState(profAddrKey(address), idBytes); err != nil {
		return 0, err
	}
	if err := appendUint64List(ctx, profListKey(), id); err != nil {
		return 0, err
	}
	return id, nil
}

func UpdateProfessor(ctx contractapi.TransactionContextInterface, id uint64, name, department, newAddress string) error {
	p, err := GetProfessor(ctx, id)
	if err != nil {
		return err
	}
	if p == nil || !p.Active {
		return fmt.Errorf("professor %d not found", id)
	}
	if name != "" {
		p.Name = name
	}
	if department != "" {
		p.Department = department
	}
	if newAddress != "" && newAddress != p.ProfessorAddress {
		// Check new address isn't taken
		existing, _ := ctx.GetStub().GetState(profAddrKey(newAddress))
		if existing != nil {
			return fmt.Errorf("address '%s' already registered", newAddress)
		}
		// Remove old index
		_ = ctx.GetStub().DelState(profAddrKey(p.ProfessorAddress))
		p.ProfessorAddress = newAddress
		idBytes, _ := json.Marshal(id)
		_ = ctx.GetStub().PutState(profAddrKey(newAddress), idBytes)
	}
	data, _ := json.Marshal(p)
	return ctx.GetStub().PutState(profKey(id), data)
}

func DeleteProfessor(ctx contractapi.TransactionContextInterface, id uint64) error {
	p, err := GetProfessor(ctx, id)
	if err != nil {
		return err
	}
	if p == nil || !p.Active {
		return fmt.Errorf("professor %d not found", id)
	}

	// Cascade: delete all courses owned by this professor
	courses, _ := GetCoursesByProfessor(ctx, id)
	for _, c := range courses {
		_ = DeleteCourse(ctx, c.ID)
	}

	// Remove address index
	_ = ctx.GetStub().DelState(profAddrKey(p.ProfessorAddress))
	// Mark inactive
	p.Active = false
	data, _ := json.Marshal(p)
	_ = ctx.GetStub().PutState(profKey(id), data)
	// Remove from list
	return removeUint64List(ctx, profListKey(), id)
}

func GetProfessor(ctx contractapi.TransactionContextInterface, id uint64) (*types.Professor, error) {
	data, err := ctx.GetStub().GetState(profKey(id))
	if err != nil {
		return nil, err
	}
	if data == nil {
		return nil, nil
	}
	var p types.Professor
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

func GetProfessorByAddress(ctx contractapi.TransactionContextInterface, addr string) (*types.Professor, error) {
	idData, err := ctx.GetStub().GetState(profAddrKey(addr))
	if err != nil || idData == nil {
		return nil, fmt.Errorf("professor with address '%s' not found", addr)
	}
	var id uint64
	if err := json.Unmarshal(idData, &id); err != nil {
		return nil, err
	}
	return GetProfessor(ctx, id)
}

func GetAllProfessors(ctx contractapi.TransactionContextInterface) ([]types.Professor, error) {
	ids, err := getUint64List(ctx, profListKey())
	if err != nil {
		return nil, err
	}
	var profs []types.Professor
	for _, id := range ids {
		p, err := GetProfessor(ctx, id)
		if err != nil || p == nil || !p.Active {
			continue
		}
		profs = append(profs, *p)
	}
	return profs, nil
}

func IsProfessorActive(ctx contractapi.TransactionContextInterface, id uint64) bool {
	p, err := GetProfessor(ctx, id)
	return err == nil && p != nil && p.Active
}
