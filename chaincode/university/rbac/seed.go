package rbac

import (
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// DefaultPermissions is the full set of permission strings seeded on InitLedger.
// These are just defaults — new permissions can be created at runtime.
var DefaultPermissions = []struct {
	Name string
	Desc string
}{
	// Majors
	{"major:create", "Create academic majors"},
	{"major:update", "Update major details"},
	{"major:deactivate", "Deactivate a major"},
	// Professors
	{"professor:create", "Register a professor"},
	{"professor:update", "Update professor details"},
	{"professor:delete", "Delete a professor (cascade)"},
	// Students
	{"student:create", "Register a student"},
	{"student:update", "Update student details"},
	{"student:delete", "Delete a student (cascade)"},
	// Courses
	{"course:create", "Create a course"},
	{"course:update", "Update course details"},
	{"course:delete", "Delete a course (cascade)"},
	{"course:reassign", "Reassign course to another professor"},
	// Enrollment
	{"enrollment:enroll", "Enroll a student in a course"},
	{"enrollment:unenroll", "Unenroll a student from a course"},
	{"enrollment:batch", "Batch enroll students"},
	// Marks
	{"mark:any", "Grade any course"},
	{"mark:own", "Grade own courses only"},
	// Roles
	{"role:manage", "Create/delete roles, grant/revoke permissions and role assignments"},
	// Permissions
	{"permission:manage", "Create new permission types"},
}

// DefaultRoles defines the three initial roles and their permissions.
var DefaultRoles = []struct {
	Name        string
	Description string
	Permissions []string
}{
	{
		Name:        "SuperAdmin",
		Description: "Full system access — all permissions",
		Permissions: nil, // filled below with ALL permissions
	},
	{
		Name:        "Registrar",
		Description: "Student, course, and enrollment management",
		Permissions: []string{
			"professor:create", "professor:update",
			"student:create", "student:update", "student:delete",
			"course:create", "course:update", "course:delete", "course:reassign",
			"enrollment:enroll", "enrollment:unenroll", "enrollment:batch",
			"mark:any",
		},
	},
	{
		Name:        "Instructor",
		Description: "Grade own courses only",
		Permissions: []string{"mark:own"},
	},
}

// SeedDefaults creates all default permissions, roles, and grants SuperAdmin
// to the caller (deployer). Called once during InitLedger.
func SeedDefaults(ctx contractapi.TransactionContextInterface) error {
	callerAddr, err := GetCallerAddress(ctx)
	if err != nil {
		return err
	}

	// 1. Create all default permissions
	allPermNames := make([]string, 0, len(DefaultPermissions))
	for _, p := range DefaultPermissions {
		if err := CreatePermission(ctx, p.Name, p.Desc); err != nil {
			// Ignore "already exists" — idempotent
			_ = err
		}
		allPermNames = append(allPermNames, p.Name)
	}

	// 2. Create default roles and assign their permissions
	for _, r := range DefaultRoles {
		if err := CreateRole(ctx, r.Name, r.Description); err != nil {
			_ = err
		}
		perms := r.Permissions
		if perms == nil {
			// SuperAdmin gets ALL permissions
			perms = allPermNames
		}
		for _, perm := range perms {
			_ = grantPermToRoleUnchecked(ctx, r.Name, perm)
		}
	}

	// 3. Grant SuperAdmin to the deployer
	if err := grantRoleUnchecked(ctx, "SuperAdmin", callerAddr); err != nil {
		return err
	}

	// 4. Lock the super admin address (immutable)
	return SetSuperAdmin(ctx, callerAddr)
}
