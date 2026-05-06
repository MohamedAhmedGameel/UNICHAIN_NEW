
package rbac

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"unicode/utf8"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
	"github.com/unichain/chaincode/university/types"
)

// ─── Key helpers ─────────────────────────────────────────────────────────────

func roleKey(name string) string                  { return "ROLE~" + name }
func roleMemberKey(role, addr string) string       { return "ROLE_MEMBERS~" + role + "~" + addr }
func roleMemberListKey(role string) string          { return "ROLE_MEMBER_LIST~" + role }
func userRoleKey(addr, role string) string          { return "USER_ROLES~" + addr + "~" + role }
func userRoleListKey(addr string) string            { return "USER_ROLE_LIST~" + addr }
func permissionKey(perm string) string              { return "PERMISSION~" + perm }
func rolePermKey(role, perm string) string          { return "ROLE_PERMISSION~" + role + "~" + perm }
func rolePermListKey(role string) string            { return "ROLE_PERM_LIST~" + role }

const superAdminKey = "SUPER_ADMIN"

// ─── Caller identity ─────────────────────────────────────────────────────────

// GetCallerAddress extracts a deterministic 40-char hex address from the
// caller's X.509 certificate subject. This is the Fabric equivalent of
// Ethereum's msg.sender.
func GetCallerAddress(ctx contractapi.TransactionContextInterface) (string, error) {
	id, err := ctx.GetClientIdentity().GetID()
	if err != nil {
		return "", fmt.Errorf("failed to get client identity: %w", err)
	}
	hash := sha256.Sum256([]byte(id))
	return fmt.Sprintf("%x", hash)[:40], nil
}

// ─── Super Admin ─────────────────────────────────────────────────────────────

func SetSuperAdmin(ctx contractapi.TransactionContextInterface, addr string) error {
	return ctx.GetStub().PutState(superAdminKey, []byte(addr))
}

func GetSuperAdmin(ctx contractapi.TransactionContextInterface) (string, error) {
	val, err := ctx.GetStub().GetState(superAdminKey)
	if err != nil {
		return "", err
	}
	if val == nil {
		return "", nil
	}
	return string(val), nil
}

func IsSuperAdmin(ctx contractapi.TransactionContextInterface, addr string) bool {
	sa, err := GetSuperAdmin(ctx)
	if err != nil || sa == "" {
		return false
	}
	return sa == addr
}

// ─── Permission CRUD ─────────────────────────────────────────────────────────

func CreatePermission(ctx contractapi.TransactionContextInterface, name, description string) error {
	key := permissionKey(name)
	existing, _ := ctx.GetStub().GetState(key)
	if existing != nil {
		return fmt.Errorf("permission '%s' already exists", name)
	}
	p := types.Permission{Name: name, Description: description}
	data, _ := json.Marshal(p)
	return ctx.GetStub().PutState(key, data)
}

func GetPermission(ctx contractapi.TransactionContextInterface, name string) (*types.Permission, error) {
	data, err := ctx.GetStub().GetState(permissionKey(name))
	if err != nil {
		return nil, err
	}
	if data == nil {
		return nil, nil
	}
	var p types.Permission
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

func GetAllPermissions(ctx contractapi.TransactionContextInterface) ([]types.Permission, error) {
	iter, err := ctx.GetStub().GetStateByRange("PERMISSION~", "PERMISSION")
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	var perms []types.Permission
	for iter.HasNext() {
		kv, err := iter.Next()
		if err != nil {
			return nil, err
		}
		var p types.Permission
		if err := json.Unmarshal(kv.Value, &p); err != nil {
			continue
		}
		perms = append(perms, p)
	}
	return perms, nil
}

// ─── Role CRUD ───────────────────────────────────────────────────────────────

func CreateRole(ctx contractapi.TransactionContextInterface, name, description string) error {
	key := roleKey(name)
	existing, _ := ctx.GetStub().GetState(key)
	if existing != nil {
		return fmt.Errorf("role '%s' already exists", name)
	}
	r := types.Role{Name: name, Description: description, Active: true}
	data, _ := json.Marshal(r)
	return ctx.GetStub().PutState(key, data)
}


func DeleteRole(ctx contractapi.TransactionContextInterface, name string) error {
	// Remove all members from this role first
	members, _ := getStringList(ctx, roleMemberListKey(name))
	for _, addr := range members {
		_ = ctx.GetStub().DelState(roleMemberKey(name, addr))
		_ = ctx.GetStub().DelState(userRoleKey(addr, name))
		removeFromStringList(ctx, userRoleListKey(addr), name)
	}
	_ = ctx.GetStub().DelState(roleMemberListKey(name))

	// Remove all permission mappings
	perms, _ := getStringList(ctx, rolePermListKey(name))
	for _, p := range perms {
		_ = ctx.GetStub().DelState(rolePermKey(name, p))
	}
	_ = ctx.GetStub().DelState(rolePermListKey(name))

	return ctx.GetStub().DelState(roleKey(name))
}

func GetRole(ctx contractapi.TransactionContextInterface, name string) (*types.Role, error) {
	data, err := ctx.GetStub().GetState(roleKey(name))
	if err != nil {
		return nil, err
	}
	if data == nil {
		return nil, nil
	}
	var r types.Role
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, err
	}
	return &r, nil
}

func GetAllRoles(ctx contractapi.TransactionContextInterface) ([]types.Role, error) {
	iter, err := ctx.GetStub().GetStateByRange("ROLE~", "ROLE")
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	var roles []types.Role

	for iter.HasNext() {
		kv, err := iter.Next()
		if err != nil {
			continue
		}

		// 🔥 CRITICAL: ensure data is valid UTF-8 BEFORE using it
		if !utf8.Valid(kv.Value) {
			continue
		}

		var role types.Role
		if err := json.Unmarshal(kv.Value, &role); err != nil {
			continue
		}

		// 🔥 EXTRA SAFETY: validate string fields
		if !utf8.ValidString(role.Name) {
			continue
		}

		roles = append(roles, role)
	}

	return roles, nil
}

// ─── Role ↔ Permission ──────────────────────────────────────────────────────

func GrantPermissionToRole(ctx contractapi.TransactionContextInterface, role, perm string) error {
	r, _ := GetRole(ctx, role)
	if r == nil {
		return fmt.Errorf("role '%s' not found", role)
	}
	p, _ := GetPermission(ctx, perm)
	if p == nil {
		return fmt.Errorf("permission '%s' not found", perm)
	}
	if err := ctx.GetStub().PutState(rolePermKey(role, perm), []byte("true")); err != nil {
		return err
	}
	return appendToStringList(ctx, rolePermListKey(role), perm)
}

func RevokePermissionFromRole(ctx contractapi.TransactionContextInterface, role, perm string) error {
	_ = ctx.GetStub().DelState(rolePermKey(role, perm))
	return removeFromStringList(ctx, rolePermListKey(role), perm)
}

func GetRolePermissions(ctx contractapi.TransactionContextInterface, role string) ([]string, error) {
	return getStringList(ctx, rolePermListKey(role))
}

// ─── Role ↔ User ─────────────────────────────────────────────────────────────

func GrantRole(ctx contractapi.TransactionContextInterface, role, addr string) error {
	r, _ := GetRole(ctx, role)
	if r == nil {
		return fmt.Errorf("role '%s' not found", role)
	}
	// Check if already assigned
	val, _ := ctx.GetStub().GetState(roleMemberKey(role, addr))
	if val != nil {
		return nil // idempotent
	}
	if err := ctx.GetStub().PutState(roleMemberKey(role, addr), []byte("true")); err != nil {
		return err
	}
	if err := appendToStringList(ctx, roleMemberListKey(role), addr); err != nil {
		return err
	}
	if err := ctx.GetStub().PutState(userRoleKey(addr, role), []byte("true")); err != nil {
		return err
	}
	return appendToStringList(ctx, userRoleListKey(addr), role)
}

func RevokeRole(ctx contractapi.TransactionContextInterface, role, addr string) error {
	// Cannot revoke super admin
	sa, _ := GetSuperAdmin(ctx)
	if sa == addr {
		return fmt.Errorf("cannot revoke roles from super admin")
	}
	_ = ctx.GetStub().DelState(roleMemberKey(role, addr))
	_ = ctx.GetStub().DelState(userRoleKey(addr, role))
	_ = removeFromStringList(ctx, roleMemberListKey(role), addr)
	return removeFromStringList(ctx, userRoleListKey(addr), role)
}

func GetRoleMembers(ctx contractapi.TransactionContextInterface, role string) ([]string, error) {
	return getStringList(ctx, roleMemberListKey(role))
}

func GetUserRoles(ctx contractapi.TransactionContextInterface, addr string) ([]string, error) {
	return getStringList(ctx, userRoleListKey(addr))
}

// ─── Permission check ────────────────────────────────────────────────────────

// HasPermission checks if an address holds any role that grants the given permission.
func HasPermission(ctx contractapi.TransactionContextInterface, addr, perm string) bool {
	// Super admin always has all permissions
	if IsSuperAdmin(ctx, addr) {
		return true
	}
	roles, err := GetUserRoles(ctx, addr)
	if err != nil || len(roles) == 0 {
		return false
	}
	for _, role := range roles {
		val, _ := ctx.GetStub().GetState(rolePermKey(role, perm))
		if val != nil && string(val) == "true" {
			return true
		}
	}
	return false
}

// RequirePermission is the chaincode guard — call at the top of every write function.
func RequirePermission(ctx contractapi.TransactionContextInterface, perm string) error {
	addr, err := GetCallerAddress(ctx)
	if err != nil {
		return err
	}
	if !HasPermission(ctx, addr, perm) {
		return fmt.Errorf("forbidden: missing permission '%s'", perm)
	}
	return nil
}

// ─── String list helpers (JSON array in state) ───────────────────────────────

func getStringList(ctx contractapi.TransactionContextInterface, key string) ([]string, error) {
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

func putStringList(ctx contractapi.TransactionContextInterface, key string, items []string) error {
	data, _ := json.Marshal(types.StringList{Items: items})
	return ctx.GetStub().PutState(key, data)
}

func appendToStringList(ctx contractapi.TransactionContextInterface, key, item string) error {
	items, _ := getStringList(ctx, key)
	for _, existing := range items {
		if existing == item {
			return nil // already present
		}
	}
	items = append(items, item)
	return putStringList(ctx, key, items)
}

func removeFromStringList(ctx contractapi.TransactionContextInterface, key, item string) error {
	items, _ := getStringList(ctx, key)
	for i, v := range items {
		if v == item {
			items[i] = items[len(items)-1]
			items = items[:len(items)-1]
			return putStringList(ctx, key, items)
		}
	}
	return nil
}

// grantRoleUnchecked grants a role without verifying the role exists in state.
// Required during seeding because GetState cannot see PutState writes made
// earlier in the same transaction when CouchDB is used as state database.
func grantRoleUnchecked(ctx contractapi.TransactionContextInterface, role, addr string) error {
	if err := ctx.GetStub().PutState(roleMemberKey(role, addr), []byte("true")); err != nil {
		return err
	}
	if err := appendToStringList(ctx, roleMemberListKey(role), addr); err != nil {
		return err
	}
	if err := ctx.GetStub().PutState(userRoleKey(addr, role), []byte("true")); err != nil {
		return err
	}
	return appendToStringList(ctx, userRoleListKey(addr), role)
}

// grantPermToRoleUnchecked grants a permission to a role without verifying
// either exists in state. Same CouchDB phantom-read workaround as above.
func grantPermToRoleUnchecked(ctx contractapi.TransactionContextInterface, role, perm string) error {
	if err := ctx.GetStub().PutState(rolePermKey(role, perm), []byte("true")); err != nil {
		return err
	}
	return appendToStringList(ctx, rolePermListKey(role), perm)
}
