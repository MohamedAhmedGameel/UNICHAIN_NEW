package state

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
	"github.com/unichain/chaincode/university/types"
)

// ─── Key helpers ─────────────────────────────────────────────────────────────

func majorKey(id uint64) string        { return fmt.Sprintf("MAJOR~%d", id) }
func majorCodeKey(code string) string  { return "MAJOR_CODE~" + strings.ToUpper(code) }
func majorListKey() string             { return "MAJOR_LIST" }
func counterKey(entity string) string  { return "CTR~" + entity }

// ─── Counter ─────────────────────────────────────────────────────────────────

func nextID(ctx contractapi.TransactionContextInterface, entity string) (uint64, error) {
	key := counterKey(entity)
	data, err := ctx.GetStub().GetState(key)
	if err != nil {
		return 0, err
	}
	var ctr types.Counter
	if data != nil {
		if err := json.Unmarshal(data, &ctr); err != nil {
			return 0, err
		}
	} else {
		ctr = types.Counter{Entity: entity, NextID: 1}
	}
	id := ctr.NextID
	ctr.NextID++
	out, _ := json.Marshal(ctr)
	if err := ctx.GetStub().PutState(key, out); err != nil {
		return 0, err
	}
	return id, nil
}

// ─── Major CRUD ──────────────────────────────────────────────────────────────

func AddMajor(ctx contractapi.TransactionContextInterface, code, name, description string) (uint64, error) {
	code = strings.ToUpper(strings.TrimSpace(code))
	name = strings.TrimSpace(name)
	if code == "" || name == "" {
		return 0, fmt.Errorf("code and name are required")
	}
	// Check unique code
	existing, _ := ctx.GetStub().GetState(majorCodeKey(code))
	if existing != nil {
		return 0, fmt.Errorf("major code '%s' already exists", code)
	}

	id, err := nextID(ctx, "major")
	if err != nil {
		return 0, err
	}

	m := types.Major{
		ID:          id,
		Code:        code,
		Name:        name,
		Description: description,
		Active:      true,
	}
	data, _ := json.Marshal(m)
	if err := ctx.GetStub().PutState(majorKey(id), data); err != nil {
		return 0, err
	}
	// Code → ID index
	idBytes, _ := json.Marshal(id)
	if err := ctx.GetStub().PutState(majorCodeKey(code), idBytes); err != nil {
		return 0, err
	}
	// Append to list
	if err := appendUint64List(ctx, majorListKey(), id); err != nil {
		return 0, err
	}
	return id, nil
}

func UpdateMajor(ctx contractapi.TransactionContextInterface, id uint64, name, description string) error {
	m, err := GetMajor(ctx, id)
	if err != nil {
		return err
	}
	if m == nil || !m.Active {
		return fmt.Errorf("major %d not found", id)
	}
	if name != "" {
		m.Name = name
	}
	if description != "" {
		m.Description = description
	}
	data, _ := json.Marshal(m)
	return ctx.GetStub().PutState(majorKey(id), data)
}

func DeactivateMajor(ctx contractapi.TransactionContextInterface, id uint64) error {
	m, err := GetMajor(ctx, id)
	if err != nil {
		return err
	}
	if m == nil || !m.Active {
		return fmt.Errorf("major %d not found", id)
	}
	m.Active = false
	data, _ := json.Marshal(m)
	return ctx.GetStub().PutState(majorKey(id), data)
}

func GetMajor(ctx contractapi.TransactionContextInterface, id uint64) (*types.Major, error) {
	data, err := ctx.GetStub().GetState(majorKey(id))
	if err != nil {
		return nil, err
	}
	if data == nil {
		return nil, nil
	}
	var m types.Major
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

func GetMajorByCode(ctx contractapi.TransactionContextInterface, code string) (*types.Major, error) {
	code = strings.ToUpper(strings.TrimSpace(code))
	idData, err := ctx.GetStub().GetState(majorCodeKey(code))
	if err != nil {
		return nil, err
	}
	if idData == nil {
		return nil, fmt.Errorf("major code '%s' not found", code)
	}
	var id uint64
	if err := json.Unmarshal(idData, &id); err != nil {
		return nil, err
	}
	m, err := GetMajor(ctx, id)
	if err != nil {
		return nil, err
	}
	if m == nil || !m.Active {
		return nil, fmt.Errorf("major code '%s' not found or inactive", code)
	}
	return m, nil
}

func GetAllMajors(ctx contractapi.TransactionContextInterface) ([]types.Major, error) {
	ids, err := getUint64List(ctx, majorListKey())
	if err != nil {
		return nil, err
	}
	var majors []types.Major
	for _, id := range ids {
		m, err := GetMajor(ctx, id)
		if err != nil || m == nil {
			continue
		}
		if m.Active {
			majors = append(majors, *m)
		}
	}
	return majors, nil
}

func IsMajorActive(ctx contractapi.TransactionContextInterface, id uint64) bool {
	m, err := GetMajor(ctx, id)
	return err == nil && m != nil && m.Active
}

// ─── Uint64 list helpers ─────────────────────────────────────────────────────

func getUint64List(ctx contractapi.TransactionContextInterface, key string) ([]uint64, error) {
	data, err := ctx.GetStub().GetState(key)
	if err != nil {
		return nil, err
	}
	if data == nil {
		return []uint64{}, nil
	}
	var list types.Uint64List
	if err := json.Unmarshal(data, &list); err != nil {
		return nil, err
	}
	return list.Items, nil
}

func putUint64List(ctx contractapi.TransactionContextInterface, key string, items []uint64) error {
	data, _ := json.Marshal(types.Uint64List{Items: items})
	return ctx.GetStub().PutState(key, data)
}

func appendUint64List(ctx contractapi.TransactionContextInterface, key string, item uint64) error {
	items, _ := getUint64List(ctx, key)
	items = append(items, item)
	return putUint64List(ctx, key, items)
}

func removeUint64List(ctx contractapi.TransactionContextInterface, key string, item uint64) error {
	items, _ := getUint64List(ctx, key)
	for i, v := range items {
		if v == item {
			items[i] = items[len(items)-1]
			items = items[:len(items)-1]
			return putUint64List(ctx, key, items)
		}
	}
	return nil
}

// ─── String list helpers (shared with other state files) ─────────────────────

func GetStringList(ctx contractapi.TransactionContextInterface, key string) ([]string, error) {
	data, err := ctx.GetStub().GetState(key)
	if err != nil {
		return nil, err
	}
	if data == nil {
		return []string{}, nil
	}
	var list types.StringList
	if err := json.Unmarshal(data, &list); err != nil {
		return nil, err
	}
	return list.Items, nil
}

func PutStringList(ctx contractapi.TransactionContextInterface, key string, items []string) error {
	data, _ := json.Marshal(types.StringList{Items: items})
	return ctx.GetStub().PutState(key, data)
}

func AppendStringList(ctx contractapi.TransactionContextInterface, key, item string) error {
	items, _ := GetStringList(ctx, key)
	for _, v := range items {
		if v == item {
			return nil
		}
	}
	items = append(items, item)
	return PutStringList(ctx, key, items)
}

func RemoveStringList(ctx contractapi.TransactionContextInterface, key, item string) error {
	items, _ := GetStringList(ctx, key)
	for i, v := range items {
		if v == item {
			items[i] = items[len(items)-1]
			items = items[:len(items)-1]
			return PutStringList(ctx, key, items)
		}
	}
	return nil
}
