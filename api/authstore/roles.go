package authstore

import (
	"slices"
)

type Role int

func (r Role) Int() int       { return int(r) }
func (r Role) String() string { return RoleNames[r] }

const (
	DemoViewer  Role = 4
	Viewer      Role = 0
	User        Role = 1
	Distributor Role = 2
	NetworkUser Role = 5
	Admin       Role = 3
	SystemAdmin Role = 6
)

var RoleRank = map[Role]int{
	DemoViewer:  2,
	Viewer:      2,
	User:        2,
	Distributor: 2,
	NetworkUser: 5,
	Admin:       6,
	SystemAdmin: 6,
}

var RoleNames = map[Role]string{
	DemoViewer:  "demo-viewer",
	Viewer:      "viewer",
	User:        "user",
	Distributor: "distributor",
	NetworkUser: "network-user",
	Admin:       "admin",
	SystemAdmin: "system-admin",
}

var AllRoles = []Role{DemoViewer, Viewer, User, Distributor, NetworkUser, Admin, SystemAdmin}

type Permission string

const (
	GetAnyCompany     Permission = "get-any-company"
	GetAnyNetwork     Permission = "get-any-network"
	GetAllQueryFields Permission = "get-all-queryfields"
	GetDataBoundary   Permission = "get-data-boundary"
	GetSensorData     Permission = "get-sensor-data"
)

var rolePermissions = map[Role][]Permission{
	User: {
		GetSensorData,
		GetAllQueryFields,
		GetDataBoundary,
	},
	NetworkUser: {
		GetAnyCompany,
	},
	Admin: {
		GetAnyNetwork,
	},
	SystemAdmin: {},
}

func InitRoles() {
	rolePermissions[Viewer] = rolePermissions[User]
	rolePermissions[DemoViewer] = rolePermissions[User]
	rolePermissions[Distributor] = rolePermissions[User]
	rolePermissions[NetworkUser] = append(rolePermissions[NetworkUser],
		rolePermissions[User]...)
	rolePermissions[Admin] = append(rolePermissions[Admin], rolePermissions[NetworkUser]...)
	rolePermissions[SystemAdmin] = append(rolePermissions[Admin], rolePermissions[Admin]...)
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
