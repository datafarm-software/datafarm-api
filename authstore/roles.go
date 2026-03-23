package authstore

import "slices"

type Role int

func (r Role) Int() int       { return int(r) }
func (r Role) String() string { return RoleNames[r] }

const (
	// DemoViewer  Role = 4
	// Viewer      Role = 0
	// TeamManager Role = 1
	// Distributor Role = 2
	User        Role = 0
	NetworkUser Role = 5
	Admin       Role = 3
)

var RoleRank = map[Role]int{
	// DemoViewer:  1,
	// Viewer:      2,
	// TeamManager: 3,
	// Distributor: 4,
	User:        2,
	NetworkUser: 5,
	Admin:       6,
}

var RoleNames = map[Role]string{
	User:        "user",
	NetworkUser: "network-user",
	Admin:       "admin",
}

var AllRoles = []Role{User, NetworkUser, Admin}

type Permission string

const (
	GetAnyCompany Permission = "get-any-company"
	GetAnyNetwork Permission = "get-any-network"
)

var rolePermissions = map[Role][]Permission{
	NetworkUser: {
		GetAnyCompany,
	},
	Admin: {
		GetAnyNetwork,
	},
}

func InitRoles() error {
	rolePermissions[NetworkUser] = append(rolePermissions[NetworkUser], rolePermissions[User]...)
	rolePermissions[Admin] = append(rolePermissions[Admin], rolePermissions[NetworkUser]...)
	return nil
}

func RoleAtLeast(userRole Role, minRequired Role) bool {
	return RoleRank[userRole] >= RoleRank[minRequired]
}

func HasPermission(role Role, permission Permission) bool {
	perms, ok := rolePermissions[role]
	if !ok {
		return false
	}
	return slices.Contains(perms, permission)
}
