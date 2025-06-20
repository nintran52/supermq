// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package groups

import (
	"context"
	"fmt"
	"time"

	"github.com/absmach/supermq"
	grpcChannelsV1 "github.com/absmach/supermq/api/grpc/channels/v1"
	grpcClientsV1 "github.com/absmach/supermq/api/grpc/clients/v1"
	apiutil "github.com/absmach/supermq/api/http/util"
	smqauthn "github.com/absmach/supermq/pkg/authn"
	"github.com/absmach/supermq/pkg/errors"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	"github.com/absmach/supermq/pkg/policies"
	"github.com/absmach/supermq/pkg/roles"
)

var ErrGroupIDs = errors.New("invalid group ids")

type service struct {
	repo       Repository
	policy     policies.Service
	idProvider supermq.IDProvider
	channels   grpcChannelsV1.ChannelsServiceClient
	clients    grpcClientsV1.ClientsServiceClient

	roles.ProvisionManageService
}

// NewService returns a new groups service implementation.
func NewService(repo Repository, policy policies.Service, idp supermq.IDProvider, channels grpcChannelsV1.ChannelsServiceClient, clients grpcClientsV1.ClientsServiceClient, sidProvider supermq.IDProvider, availableActions []roles.Action, builtInRoles map[roles.BuiltInRoleName][]roles.Action) (Service, error) {
	rpms, err := roles.NewProvisionManageService(policies.GroupType, repo, policy, sidProvider, availableActions, builtInRoles)
	if err != nil {
		return service{}, err
	}
	return service{
		repo:                   repo,
		policy:                 policy,
		idProvider:             idp,
		channels:               channels,
		clients:                clients,
		ProvisionManageService: rpms,
	}, nil
}

func (svc service) CreateGroup(ctx context.Context, session smqauthn.Session, g Group) (retGr Group, retRps []roles.RoleProvision, retErr error) {
	groupID, err := svc.idProvider.ID()
	if err != nil {
		return Group{}, []roles.RoleProvision{}, err
	}
	if g.Status != EnabledStatus && g.Status != DisabledStatus {
		return Group{}, []roles.RoleProvision{}, svcerr.ErrInvalidStatus
	}

	g.ID = groupID
	g.CreatedAt = time.Now().UTC()
	g.Domain = session.DomainID

	saved, err := svc.repo.Save(ctx, g)
	if err != nil {
		return Group{}, []roles.RoleProvision{}, errors.Wrap(svcerr.ErrCreateEntity, err)
	}

	defer func() {
		if retErr != nil {
			if errRollback := svc.repo.Delete(ctx, saved.ID); errRollback != nil {
				retErr = errors.Wrap(retErr, errors.Wrap(apiutil.ErrRollbackTx, errRollback))
			}
		}
	}()

	oprs := []policies.Policy{}

	oprs = append(oprs, policies.Policy{
		Domain:      session.DomainID,
		SubjectType: policies.DomainType,
		Subject:     session.DomainID,
		Relation:    policies.DomainRelation,
		ObjectType:  policies.GroupType,
		Object:      saved.ID,
	})
	if saved.Parent != "" {
		oprs = append(oprs, policies.Policy{
			Domain:      session.DomainID,
			SubjectType: policies.GroupType,
			Subject:     saved.Parent,
			Relation:    policies.ParentGroupRelation,
			ObjectType:  policies.GroupType,
			ObjectKind:  policies.NewGroupKind,
			Object:      saved.ID,
		})
	}
	newBuiltInRoleMembers := map[roles.BuiltInRoleName][]roles.Member{
		BuiltInRoleAdmin: {roles.Member(session.UserID)},
	}
	nrps, err := svc.AddNewEntitiesRoles(ctx, session.DomainID, session.UserID, []string{saved.ID}, oprs, newBuiltInRoleMembers)
	if err != nil {
		return Group{}, []roles.RoleProvision{}, errors.Wrap(svcerr.ErrAddPolicies, err)
	}

	return saved, nrps, nil
}

func (svc service) ViewGroup(ctx context.Context, session smqauthn.Session, id string, withRoles bool) (Group, error) {
	var group Group
	var err error
	switch withRoles {
	case true:
		group, err = svc.repo.RetrieveByIDWithRoles(ctx, id, session.UserID)
	default:
		group, err = svc.repo.RetrieveByID(ctx, id)
	}
	if err != nil {
		return Group{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}

	return group, nil
}

func (svc service) ListGroups(ctx context.Context, session smqauthn.Session, gm PageMeta) (Page, error) {
	switch session.SuperAdmin {
	case true:
		gm.DomainID = session.DomainID
		page, err := svc.repo.RetrieveAll(ctx, gm)
		if err != nil {
			return Page{}, errors.Wrap(svcerr.ErrViewEntity, err)
		}
		return page, nil
	default:
		page, err := svc.repo.RetrieveUserGroups(ctx, session.DomainID, session.UserID, gm)
		if err != nil {
			return Page{}, errors.Wrap(svcerr.ErrViewEntity, err)
		}
		return page, nil
	}
}

func (svc service) ListUserGroups(ctx context.Context, session smqauthn.Session, userID string, pm PageMeta) (Page, error) {
	page, err := svc.repo.RetrieveUserGroups(ctx, session.DomainID, userID, pm)
	if err != nil {
		return Page{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}
	return page, nil
}

func (svc service) UpdateGroup(ctx context.Context, session smqauthn.Session, g Group) (Group, error) {
	g.UpdatedAt = time.Now().UTC()
	g.UpdatedBy = session.UserID

	group, err := svc.repo.Update(ctx, g)
	if err != nil {
		return Group{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}
	return group, nil
}

func (svc service) UpdateGroupTags(ctx context.Context, session smqauthn.Session, g Group) (Group, error) {
	group := Group{
		ID:        g.ID,
		Tags:      g.Tags,
		UpdatedAt: time.Now(),
		UpdatedBy: session.UserID,
	}
	group, err := svc.repo.UpdateTags(ctx, group)
	if err != nil {
		return Group{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}
	return group, nil
}

func (svc service) EnableGroup(ctx context.Context, session smqauthn.Session, id string) (Group, error) {
	group := Group{
		ID:        id,
		Status:    EnabledStatus,
		UpdatedAt: time.Now().UTC(),
	}
	group, err := svc.changeGroupStatus(ctx, session, group)
	if err != nil {
		return Group{}, err
	}
	return group, nil
}

func (svc service) DisableGroup(ctx context.Context, session smqauthn.Session, id string) (Group, error) {
	group := Group{
		ID:        id,
		Status:    DisabledStatus,
		UpdatedAt: time.Now().UTC(),
	}
	group, err := svc.changeGroupStatus(ctx, session, group)
	if err != nil {
		return Group{}, err
	}
	return group, nil
}

func (svc service) RetrieveGroupHierarchy(ctx context.Context, session smqauthn.Session, id string, hm HierarchyPageMeta) (HierarchyPage, error) {
	hp, err := svc.repo.RetrieveHierarchy(ctx, id, hm)
	if err != nil {
		return HierarchyPage{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}
	hids := svc.getGroupIDs(hp.Groups)
	ids, err := svc.filterAllowedGroupIDsOfUserID(ctx, session.DomainUserID, "read_permission", hids)
	if err != nil {
		return HierarchyPage{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}
	hp.Groups = svc.allowedGroups(hp.Groups, ids)
	return hp, nil
}

func (svc service) allowedGroups(gps []Group, ids []string) []Group {
	aIDs := make(map[string]struct{}, len(ids))

	for _, id := range ids {
		aIDs[id] = struct{}{}
	}

	aGroups := []Group{}
	for _, g := range gps {
		ag := g
		if _, ok := aIDs[g.ID]; !ok {
			ag = Group{ID: "xxxx-xxxx-xxxx-xxxx", Level: g.Level}
		}
		aGroups = append(aGroups, ag)
	}
	return aGroups
}

func (svc service) getGroupIDs(gps []Group) []string {
	hids := []string{}
	for _, g := range gps {
		hids = append(hids, g.ID)
		if len(g.Children) > 0 {
			children := make([]Group, len(g.Children))
			for i, child := range g.Children {
				children[i] = *child
			}
			cids := svc.getGroupIDs(children)
			hids = append(hids, cids...)
		}
	}
	return hids
}

func (svc service) AddParentGroup(ctx context.Context, session smqauthn.Session, id, parentID string) (retErr error) {
	group, err := svc.repo.RetrieveByID(ctx, id)
	if err != nil {
		return errors.Wrap(svcerr.ErrViewEntity, err)
	}

	var pols []policies.Policy
	if group.Parent != "" {
		return errors.Wrap(svcerr.ErrConflict, fmt.Errorf("%s group already have parent", group.ID))
	}
	pols = append(pols, policies.Policy{
		Domain:      session.DomainID,
		SubjectType: policies.GroupType,
		Subject:     parentID,
		Relation:    policies.ParentGroupRelation,
		ObjectType:  policies.GroupType,
		Object:      group.ID,
	})

	if err := svc.policy.AddPolicies(ctx, pols); err != nil {
		return errors.Wrap(svcerr.ErrAddPolicies, err)
	}
	defer func() {
		if retErr != nil {
			if errRollback := svc.policy.DeletePolicies(ctx, pols); errRollback != nil {
				retErr = errors.Wrap(retErr, errors.Wrap(apiutil.ErrRollbackTx, errRollback))
			}
		}
	}()

	if err := svc.repo.AssignParentGroup(ctx, parentID, group.ID); err != nil {
		return err
	}
	return nil
}

func (svc service) RemoveParentGroup(ctx context.Context, session smqauthn.Session, id string) (retErr error) {
	group, err := svc.repo.RetrieveByID(ctx, id)
	if err != nil {
		return errors.Wrap(svcerr.ErrViewEntity, err)
	}

	if group.Parent != "" {
		var pols []policies.Policy
		pols = append(pols, policies.Policy{
			Domain:      session.DomainID,
			SubjectType: policies.GroupType,
			Subject:     group.Parent,
			Relation:    policies.ParentGroupRelation,
			ObjectType:  policies.GroupType,
			Object:      group.ID,
		})

		if err := svc.policy.DeletePolicies(ctx, pols); err != nil {
			return errors.Wrap(svcerr.ErrDeletePolicies, err)
		}
		defer func() {
			if retErr != nil {
				if errRollback := svc.policy.AddPolicies(ctx, pols); errRollback != nil {
					retErr = errors.Wrap(retErr, errors.Wrap(apiutil.ErrRollbackTx, errRollback))
				}
			}
		}()
		if err := svc.repo.UnassignParentGroup(ctx, group.Parent, group.ID); err != nil {
			return errors.Wrap(svcerr.ErrRemoveEntity, err)
		}

		return nil
	}

	return nil
}

func (svc service) AddChildrenGroups(ctx context.Context, session smqauthn.Session, parentGroupID string, childrenGroupIDs []string) (retErr error) {
	childrenGroupsPage, err := svc.repo.RetrieveByIDs(ctx, PageMeta{Limit: 1<<63 - 1}, childrenGroupIDs...)
	if err != nil {
		return errors.Wrap(svcerr.ErrViewEntity, err)
	}
	if len(childrenGroupsPage.Groups) == 0 {
		return ErrGroupIDs
	}

	for _, childGroup := range childrenGroupsPage.Groups {
		if childGroup.Parent != "" {
			return errors.Wrap(svcerr.ErrConflict, fmt.Errorf("%s group already have parent", childGroup.ID))
		}
	}

	var pols []policies.Policy
	for _, childGroup := range childrenGroupsPage.Groups {
		pols = append(pols, policies.Policy{
			Domain:      session.DomainID,
			SubjectType: policies.GroupType,
			Subject:     parentGroupID,
			Relation:    policies.ParentGroupRelation,
			ObjectType:  policies.GroupType,
			Object:      childGroup.ID,
		})
	}

	if err := svc.policy.AddPolicies(ctx, pols); err != nil {
		return errors.Wrap(svcerr.ErrAddPolicies, err)
	}
	defer func() {
		if retErr != nil {
			if errRollback := svc.policy.DeletePolicies(ctx, pols); errRollback != nil {
				retErr = errors.Wrap(retErr, errors.Wrap(apiutil.ErrRollbackTx, errRollback))
			}
		}
	}()
	if err = svc.repo.AssignParentGroup(ctx, parentGroupID, childrenGroupIDs...); err != nil {
		return errors.Wrap(svcerr.ErrUpdateEntity, err)
	}

	return nil
}

func (svc service) RemoveChildrenGroups(ctx context.Context, session smqauthn.Session, parentGroupID string, childrenGroupIDs []string) (retErr error) {
	childrenGroupsPage, err := svc.repo.RetrieveByIDs(ctx, PageMeta{Limit: 1<<63 - 1}, childrenGroupIDs...)
	if err != nil {
		return errors.Wrap(svcerr.ErrViewEntity, err)
	}
	if len(childrenGroupsPage.Groups) == 0 {
		return ErrGroupIDs
	}

	var pols []policies.Policy

	for _, group := range childrenGroupsPage.Groups {
		if group.Parent != "" && group.Parent != parentGroupID {
			return errors.Wrap(svcerr.ErrConflict, fmt.Errorf("%s group doesn't have same parent", group.ID))
		}
		pols = append(pols, policies.Policy{
			Domain:      session.DomainID,
			SubjectType: policies.GroupType,
			Subject:     parentGroupID,
			Relation:    policies.ParentGroupRelation,
			ObjectType:  policies.GroupType,
			Object:      group.ID,
		})
	}

	if err := svc.policy.DeletePolicies(ctx, pols); err != nil {
		return errors.Wrap(svcerr.ErrDeletePolicies, err)
	}
	defer func() {
		if retErr != nil {
			if errRollback := svc.policy.AddPolicies(ctx, pols); errRollback != nil {
				retErr = errors.Wrap(retErr, errors.Wrap(apiutil.ErrRollbackTx, errRollback))
			}
		}
	}()
	if err := svc.repo.UnassignParentGroup(ctx, parentGroupID, childrenGroupIDs...); err != nil {
		return errors.Wrap(svcerr.ErrUpdateEntity, err)
	}

	return nil
}

func (svc service) RemoveAllChildrenGroups(ctx context.Context, session smqauthn.Session, id string) error {
	pol := policies.Policy{
		Domain:      session.DomainID,
		SubjectType: policies.GroupType,
		Subject:     id,
		Relation:    policies.ParentGroupRelation,
		ObjectType:  policies.GroupType,
	}

	if err := svc.policy.DeletePolicyFilter(ctx, pol); err != nil {
		return errors.Wrap(svcerr.ErrDeletePolicies, err)
	}
	if err := svc.repo.UnassignAllChildrenGroups(ctx, id); err != nil {
		return errors.Wrap(svcerr.ErrRemoveEntity, err)
	}

	return nil
}

func (svc service) ListChildrenGroups(ctx context.Context, session smqauthn.Session, id string, startLevel, endLevel int64, pm PageMeta) (Page, error) {
	page, err := svc.repo.RetrieveChildrenGroups(ctx, session.DomainID, session.UserID, id, startLevel, endLevel, pm)
	if err != nil {
		return Page{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}
	return page, nil
}

func (svc service) DeleteGroup(ctx context.Context, session smqauthn.Session, id string) error {
	if _, err := svc.channels.UnsetParentGroupFromChannels(ctx, &grpcChannelsV1.UnsetParentGroupFromChannelsReq{ParentGroupId: id}); err != nil {
		return errors.Wrap(svcerr.ErrRemoveEntity, err)
	}

	if _, err := svc.clients.UnsetParentGroupFromClient(ctx, &grpcClientsV1.UnsetParentGroupFromClientReq{ParentGroupId: id}); err != nil {
		return errors.Wrap(svcerr.ErrRemoveEntity, err)
	}

	g, err := svc.repo.ChangeStatus(ctx, Group{ID: id, Status: DeletedStatus})
	if err != nil {
		return errors.Wrap(svcerr.ErrRemoveEntity, err)
	}

	filterDeletePolicies := []policies.Policy{
		{
			SubjectType: policies.GroupType,
			Subject:     id,
		},
		{
			ObjectType: policies.GroupType,
			Object:     id,
		},
	}
	deletePolicies := []policies.Policy{
		{
			SubjectType: policies.DomainType,
			Subject:     session.DomainID,
			Relation:    policies.DomainRelation,
			ObjectType:  policies.GroupType,
			Object:      id,
		},
	}
	if g.Parent != "" {
		deletePolicies = append(deletePolicies, policies.Policy{
			Domain:      session.DomainID,
			SubjectType: policies.GroupType,
			Subject:     g.Parent,
			Relation:    policies.ParentGroupRelation,
			ObjectType:  policies.GroupType,
			Object:      id,
		})
	}
	if err := svc.RemoveEntitiesRoles(ctx, session.DomainID, session.DomainUserID, []string{id}, filterDeletePolicies, deletePolicies); err != nil {
		return errors.Wrap(svcerr.ErrDeletePolicies, err)
	}

	if err := svc.repo.Delete(ctx, id); err != nil {
		return err
	}

	return nil
}

func (svc service) filterAllowedGroupIDsOfUserID(ctx context.Context, userID, permission string, groupIDs []string) ([]string, error) {
	var ids []string
	allowedIDs, err := svc.listAllGroupsOfUserID(ctx, userID, permission)
	if err != nil {
		return []string{}, err
	}

	for _, gid := range groupIDs {
		for _, id := range allowedIDs {
			if id == gid {
				ids = append(ids, id)
			}
		}
	}
	return ids, nil
}

func (svc service) listAllGroupsOfUserID(ctx context.Context, userID, permission string) ([]string, error) {
	allowedIDs, err := svc.policy.ListAllObjects(ctx, policies.Policy{
		SubjectType: policies.UserType,
		Subject:     userID,
		Permission:  permission,
		ObjectType:  policies.GroupType,
	})
	if err != nil {
		return []string{}, err
	}
	return allowedIDs.Policies, nil
}

func (svc service) changeGroupStatus(ctx context.Context, session smqauthn.Session, group Group) (Group, error) {
	dbGroup, err := svc.repo.RetrieveByID(ctx, group.ID)
	if err != nil {
		return Group{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}
	if dbGroup.Status == group.Status {
		return Group{}, errors.ErrStatusAlreadyAssigned
	}

	group.UpdatedBy = session.UserID
	return svc.repo.ChangeStatus(ctx, group)
}
