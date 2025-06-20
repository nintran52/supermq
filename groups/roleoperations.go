// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package groups

import (
	"github.com/absmach/supermq/pkg/roles"
	"github.com/absmach/supermq/pkg/svcutil"
)

// Internal Operations

const (
	OpViewGroup svcutil.Operation = iota
	OpUpdateGroup
	OpUpdateGroupTags
	OpEnableGroup
	OpDisableGroup
	OpRetrieveGroupHierarchy
	OpAddParentGroup
	OpRemoveParentGroup
	OpAddChildrenGroups
	OpRemoveChildrenGroups
	OpRemoveAllChildrenGroups
	OpListChildrenGroups
	OpDeleteGroup
	OpCreateGroup
	OpListGroups
	OpListUserGroups
)

var expectedOperations = []svcutil.Operation{
	OpViewGroup,
	OpUpdateGroup,
	OpUpdateGroupTags,
	OpEnableGroup,
	OpDisableGroup,
	OpRetrieveGroupHierarchy,
	OpAddParentGroup,
	OpRemoveParentGroup,
	OpAddChildrenGroups,
	OpRemoveChildrenGroups,
	OpRemoveAllChildrenGroups,
	OpListChildrenGroups,
	OpDeleteGroup,
}

var OperationNames = []string{
	"OpViewGroup",
	"OpUpdateGroup",
	"OpUpdateGroupTags",
	"OpEnableGroup",
	"OpDisableGroup",
	"OpRetrieveGroupHierarchy",
	"OpAddParentGroup",
	"OpRemoveParentGroup",
	"OpAddChildrenGroups",
	"OpRemoveChildrenGroups",
	"OpRemoveAllChildrenGroups",
	"OpListChildrenGroups",
	"OpDeleteGroup",
	"OpCreateGroup",
	"OpListGroups",
	"OpListUserGroups",
}

func NewOperationPerm() svcutil.OperationPerm {
	return svcutil.NewOperationPerm(expectedOperations, OperationNames)
}

// External Operations.
const (
	DomainOpCreateGroup svcutil.ExternalOperation = iota
	DomainOpListGroups
	UserOpListGroups
	ClientsOpListGroups
	ChannelsOpListGroups
)

var expectedExternalOperations = []svcutil.ExternalOperation{
	DomainOpCreateGroup,
	DomainOpListGroups,
	UserOpListGroups,
	ClientsOpListGroups,
	ChannelsOpListGroups,
}

var externalOperationNames = []string{
	"DomainOpCreateGroup",
	"DomainOpListGroups",
	"UserOpListGroups",
	"ClientsOpListGroups",
	"ChannelsOpListGroups",
}

func NewExternalOperationPerm() svcutil.ExternalOperationPerm {
	return svcutil.NewExternalOperationPerm(expectedExternalOperations, externalOperationNames)
}

// Below codes should moved out of service, may be can be kept in `cmd/<svc>/main.go`

const (
	updatePermission    = "update_permission"
	readPermission      = "read_permission"
	deletePermission    = "delete_permission"
	setChildPermission  = "set_child_permission"
	setParentPermission = "set_parent_permission"

	manageRolePermission      = "manage_role_permission"
	addRoleUsersPermission    = "add_role_users_permission"
	removeRoleUsersPermission = "remove_role_users_permission"
	viewRoleUsersPermission   = "view_role_users_permission"
)

func NewOperationPermissionMap() map[svcutil.Operation]svcutil.Permission {
	opPerm := map[svcutil.Operation]svcutil.Permission{
		OpViewGroup:               readPermission,
		OpUpdateGroup:             updatePermission,
		OpUpdateGroupTags:         updatePermission,
		OpEnableGroup:             updatePermission,
		OpDisableGroup:            updatePermission,
		OpRetrieveGroupHierarchy:  readPermission,
		OpAddParentGroup:          setParentPermission,
		OpRemoveParentGroup:       setParentPermission,
		OpAddChildrenGroups:       setChildPermission,
		OpRemoveChildrenGroups:    setChildPermission,
		OpRemoveAllChildrenGroups: setChildPermission,
		OpListChildrenGroups:      readPermission,
		OpDeleteGroup:             deletePermission,
	}
	return opPerm
}

func NewRolesOperationPermissionMap() map[svcutil.Operation]svcutil.Permission {
	opPerm := map[svcutil.Operation]svcutil.Permission{
		roles.OpAddRole:                manageRolePermission,
		roles.OpRemoveRole:             manageRolePermission,
		roles.OpUpdateRoleName:         manageRolePermission,
		roles.OpRetrieveRole:           manageRolePermission,
		roles.OpRetrieveAllRoles:       manageRolePermission,
		roles.OpRoleAddActions:         manageRolePermission,
		roles.OpRoleListActions:        manageRolePermission,
		roles.OpRoleCheckActionsExists: manageRolePermission,
		roles.OpRoleRemoveActions:      manageRolePermission,
		roles.OpRoleRemoveAllActions:   manageRolePermission,
		roles.OpRoleAddMembers:         addRoleUsersPermission,
		roles.OpRoleListMembers:        viewRoleUsersPermission,
		roles.OpRoleCheckMembersExists: viewRoleUsersPermission,
		roles.OpRoleRemoveMembers:      removeRoleUsersPermission,
		roles.OpRoleRemoveAllMembers:   manageRolePermission,
	}
	return opPerm
}

const (
	// External Permissions for the domain.
	domainCreateGroupPermission = "group_create_permission"
	domainListGroupPermission   = "membership"
	userListGroupsPermission    = "membership"
	clientListGroupPermission   = "read_permission"
	chanelListGroupPermission   = "read_permission"
)

func NewExternalOperationPermissionMap() map[svcutil.ExternalOperation]svcutil.Permission {
	extOpPerm := map[svcutil.ExternalOperation]svcutil.Permission{
		DomainOpCreateGroup:  domainCreateGroupPermission,
		DomainOpListGroups:   domainListGroupPermission,
		UserOpListGroups:     userListGroupsPermission,
		ClientsOpListGroups:  clientListGroupPermission,
		ChannelsOpListGroups: chanelListGroupPermission,
	}
	return extOpPerm
}
