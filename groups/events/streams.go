// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"context"

	"github.com/absmach/supermq/groups"
	"github.com/absmach/supermq/pkg/authn"
	"github.com/absmach/supermq/pkg/events"
	"github.com/absmach/supermq/pkg/events/store"
	"github.com/absmach/supermq/pkg/roles"
	rmEvents "github.com/absmach/supermq/pkg/roles/rolemanager/events"
	"github.com/go-chi/chi/v5/middleware"
)

const (
	supermqPrefix           = "supermq."
	createStream            = supermqPrefix + groupCreate
	updateStream            = supermqPrefix + groupUpdate
	updateTagsStream        = supermqPrefix + groupUpdateTags
	enableStream            = supermqPrefix + groupEnable
	disableStream           = supermqPrefix + groupDisable
	viewStream              = supermqPrefix + groupView
	listStream              = supermqPrefix + groupList
	listUserGroupsStream    = supermqPrefix + groupListUserGroups
	removeStream            = supermqPrefix + groupRemove
	retrieveHierarchyStream = supermqPrefix + groupRetrieveGroupHierarchy
	addParentStream         = supermqPrefix + groupAddParentGroup
	removeParentStream      = supermqPrefix + groupRemoveParentGroup
	viewParentStream        = supermqPrefix + groupViewParentGroup
	addChildrenStream       = supermqPrefix + groupAddChildrenGroups
	removeChildrenStream    = supermqPrefix + groupRemoveChildrenGroups
	removeAllChildrenStream = supermqPrefix + groupRemoveAllChildrenGroups
	listChildrenStream      = supermqPrefix + groupListChildrenGroups
)

var _ groups.Service = (*eventStore)(nil)

type eventStore struct {
	events.Publisher
	svc groups.Service
	rmEvents.RoleManagerEventStore
}

// NewEventStoreMiddleware returns wrapper around clients service that sends
// events to event store.
func New(ctx context.Context, svc groups.Service, url string) (groups.Service, error) {
	publisher, err := store.NewPublisher(ctx, url)
	if err != nil {
		return nil, err
	}
	rmes := rmEvents.NewRoleManagerEventStore("groups", groupPrefix, svc, publisher)

	return &eventStore{
		svc:                   svc,
		Publisher:             publisher,
		RoleManagerEventStore: rmes,
	}, nil
}

func (es eventStore) CreateGroup(ctx context.Context, session authn.Session, group groups.Group) (groups.Group, []roles.RoleProvision, error) {
	group, rps, err := es.svc.CreateGroup(ctx, session, group)
	if err != nil {
		return group, rps, err
	}

	event := createGroupEvent{
		Group:            group,
		rolesProvisioned: rps,
		Session:          session,
		requestID:        middleware.GetReqID(ctx),
	}

	if err := es.Publish(ctx, createStream, event); err != nil {
		return group, rps, err
	}

	return group, rps, nil
}

func (es eventStore) UpdateGroup(ctx context.Context, session authn.Session, group groups.Group) (groups.Group, error) {
	group, err := es.svc.UpdateGroup(ctx, session, group)
	if err != nil {
		return group, err
	}

	event := updateGroupEvent{
		Group:     group,
		Session:   session,
		operation: groupUpdate,
		requestID: middleware.GetReqID(ctx),
	}

	if err := es.Publish(ctx, updateStream, event); err != nil {
		return group, err
	}

	return group, nil
}

func (es *eventStore) UpdateGroupTags(ctx context.Context, session authn.Session, g groups.Group) (groups.Group, error) {
	g, err := es.svc.UpdateGroupTags(ctx, session, g)
	if err != nil {
		return g, err
	}

	event := updateGroupEvent{
		Group:     g,
		Session:   session,
		operation: groupUpdateTags,
		requestID: middleware.GetReqID(ctx),
	}
	if err := es.Publish(ctx, updateTagsStream, event); err != nil {
		return g, err
	}

	return g, nil
}

func (es eventStore) ViewGroup(ctx context.Context, session authn.Session, id string, withRoles bool) (groups.Group, error) {
	group, err := es.svc.ViewGroup(ctx, session, id, withRoles)
	if err != nil {
		return group, err
	}
	event := viewGroupEvent{
		group,
		session,
		middleware.GetReqID(ctx),
	}

	if err := es.Publish(ctx, viewStream, event); err != nil {
		return group, err
	}

	return group, nil
}

func (es eventStore) ListGroups(ctx context.Context, session authn.Session, pm groups.PageMeta) (groups.Page, error) {
	gp, err := es.svc.ListGroups(ctx, session, pm)
	if err != nil {
		return gp, err
	}
	event := listGroupEvent{
		PageMeta:   pm,
		domainID:   session.DomainID,
		userID:     session.UserID,
		tokenType:  session.Type.String(),
		superAdmin: session.SuperAdmin,
		requestID:  middleware.GetReqID(ctx),
	}

	if err := es.Publish(ctx, listStream, event); err != nil {
		return gp, err
	}

	return gp, nil
}

func (es eventStore) ListUserGroups(ctx context.Context, session authn.Session, userID string, pm groups.PageMeta) (groups.Page, error) {
	gp, err := es.svc.ListUserGroups(ctx, session, userID, pm)
	if err != nil {
		return gp, err
	}
	event := listUserGroupEvent{
		userID:     userID,
		PageMeta:   pm,
		domainID:   session.DomainID,
		tokenType:  session.Type.String(),
		superAdmin: session.SuperAdmin,
		requestID:  middleware.GetReqID(ctx),
	}

	if err := es.Publish(ctx, listUserGroupsStream, event); err != nil {
		return gp, err
	}

	return gp, nil
}

func (es eventStore) EnableGroup(ctx context.Context, session authn.Session, id string) (groups.Group, error) {
	group, err := es.svc.EnableGroup(ctx, session, id)
	if err != nil {
		return group, err
	}

	return es.changeStatus(ctx, session, groupEnable, enableStream, group)
}

func (es eventStore) DisableGroup(ctx context.Context, session authn.Session, id string) (groups.Group, error) {
	group, err := es.svc.DisableGroup(ctx, session, id)
	if err != nil {
		return group, err
	}

	return es.changeStatus(ctx, session, groupDisable, disableStream, group)
}

func (es eventStore) changeStatus(ctx context.Context, session authn.Session, operation, stream string, group groups.Group) (groups.Group, error) {
	event := changeGroupStatusEvent{
		id:        group.ID,
		operation: operation,
		updatedAt: group.UpdatedAt,
		updatedBy: group.UpdatedBy,
		status:    group.Status.String(),
		Session:   session,
		requestID: middleware.GetReqID(ctx),
	}

	if err := es.Publish(ctx, stream, event); err != nil {
		return group, err
	}

	return group, nil
}

func (es eventStore) DeleteGroup(ctx context.Context, session authn.Session, id string) error {
	if err := es.svc.DeleteGroup(ctx, session, id); err != nil {
		return err
	}
	if err := es.Publish(ctx, removeStream, deleteGroupEvent{
		id:        id,
		Session:   session,
		requestID: middleware.GetReqID(ctx),
	}); err != nil {
		return err
	}
	return nil
}

func (es eventStore) RetrieveGroupHierarchy(ctx context.Context, session authn.Session, id string, hm groups.HierarchyPageMeta) (groups.HierarchyPage, error) {
	g, err := es.svc.RetrieveGroupHierarchy(ctx, session, id, hm)
	if err != nil {
		return g, err
	}
	if err := es.Publish(ctx, retrieveHierarchyStream, retrieveGroupHierarchyEvent{id: id, Session: session, HierarchyPageMeta: hm, requestID: middleware.GetReqID(ctx)}); err != nil {
		return g, err
	}
	return g, nil
}

func (es eventStore) AddParentGroup(ctx context.Context, session authn.Session, id, parentID string) error {
	if err := es.svc.AddParentGroup(ctx, session, id, parentID); err != nil {
		return err
	}
	if err := es.Publish(ctx, addParentStream, addParentGroupEvent{id: id, parentID: parentID, Session: session, requestID: middleware.GetReqID(ctx)}); err != nil {
		return err
	}
	return nil
}

func (es eventStore) RemoveParentGroup(ctx context.Context, session authn.Session, id string) error {
	if err := es.svc.RemoveParentGroup(ctx, session, id); err != nil {
		return err
	}
	if err := es.Publish(ctx, removeParentStream, removeParentGroupEvent{id: id, Session: session, requestID: middleware.GetReqID(ctx)}); err != nil {
		return err
	}
	return nil
}

func (es eventStore) AddChildrenGroups(ctx context.Context, session authn.Session, id string, childrenGroupIDs []string) error {
	if err := es.svc.AddChildrenGroups(ctx, session, id, childrenGroupIDs); err != nil {
		return err
	}
	if err := es.Publish(ctx, addChildrenStream, addChildrenGroupsEvent{id: id, Session: session, childrenIDs: childrenGroupIDs, requestID: middleware.GetReqID(ctx)}); err != nil {
		return err
	}
	return nil
}

func (es eventStore) RemoveChildrenGroups(ctx context.Context, session authn.Session, id string, childrenGroupIDs []string) error {
	if err := es.svc.RemoveChildrenGroups(ctx, session, id, childrenGroupIDs); err != nil {
		return err
	}
	if err := es.Publish(ctx, removeChildrenStream, removeChildrenGroupsEvent{id: id, Session: session, childrenIDs: childrenGroupIDs, requestID: middleware.GetReqID(ctx)}); err != nil {
		return err
	}

	return nil
}

func (es eventStore) RemoveAllChildrenGroups(ctx context.Context, session authn.Session, id string) error {
	if err := es.svc.RemoveAllChildrenGroups(ctx, session, id); err != nil {
		return err
	}
	if err := es.Publish(ctx, removeAllChildrenStream, removeAllChildrenGroupsEvent{id: id, Session: session, requestID: middleware.GetReqID(ctx)}); err != nil {
		return err
	}
	return nil
}

func (es eventStore) ListChildrenGroups(ctx context.Context, session authn.Session, id string, startLevel, endLevel int64, pm groups.PageMeta) (groups.Page, error) {
	g, err := es.svc.ListChildrenGroups(ctx, session, id, startLevel, endLevel, pm)
	if err != nil {
		return g, err
	}
	if err := es.Publish(ctx, listChildrenStream, listChildrenGroupsEvent{
		id:         id,
		domainID:   session.DomainID,
		startLevel: startLevel,
		endLevel:   endLevel,
		PageMeta:   pm,
		userID:     session.UserID,
		tokenType:  session.Type.String(),
		superAdmin: session.SuperAdmin,
		requestID:  middleware.GetReqID(ctx),
	}); err != nil {
		return g, err
	}
	return g, nil
}
