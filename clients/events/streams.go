// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"context"

	"github.com/absmach/supermq/clients"
	"github.com/absmach/supermq/pkg/authn"
	"github.com/absmach/supermq/pkg/events"
	"github.com/absmach/supermq/pkg/events/store"
	"github.com/absmach/supermq/pkg/roles"
	rmEvents "github.com/absmach/supermq/pkg/roles/rolemanager/events"
	"github.com/go-chi/chi/v5/middleware"
)

const (
	supermqPrefix      = "supermq."
	createStream       = supermqPrefix + clientCreate
	updateStream       = supermqPrefix + clientUpdate
	updateTagsStream   = supermqPrefix + clientUpdateTags
	updateSecretStream = supermqPrefix + clientUpdateSecret
	enableStream       = supermqPrefix + clientEnable
	disableStream      = supermqPrefix + clientDisable
	removeStream       = supermqPrefix + clientRemove
	viewStream         = supermqPrefix + clientView
	viewPermsStream    = supermqPrefix + clientViewPerms
	listStream         = supermqPrefix + clientList
	listByUserStream   = supermqPrefix + clientListByUser
	listByGroupStream  = supermqPrefix + clientListByGroup
	identifyStream     = supermqPrefix + clientIdentify
	authorizeStream    = supermqPrefix + clientAuthorize
	setParentStream    = supermqPrefix + clientSetParent
	removeParentStream = supermqPrefix + clientRemoveParent
)

var _ clients.Service = (*eventStore)(nil)

type eventStore struct {
	events.Publisher
	svc clients.Service
	rmEvents.RoleManagerEventStore
}

// NewEventStoreMiddleware returns wrapper around clients service that sends
// events to event store.
func NewEventStoreMiddleware(ctx context.Context, svc clients.Service, url string) (clients.Service, error) {
	publisher, err := store.NewPublisher(ctx, url)
	if err != nil {
		return nil, err
	}
	res := rmEvents.NewRoleManagerEventStore("clients", clientPrefix, svc, publisher)

	return &eventStore{
		svc:                   svc,
		Publisher:             publisher,
		RoleManagerEventStore: res,
	}, nil
}

func (es *eventStore) CreateClients(ctx context.Context, session authn.Session, clients ...clients.Client) ([]clients.Client, []roles.RoleProvision, error) {
	clis, rps, err := es.svc.CreateClients(ctx, session, clients...)
	if err != nil {
		return clis, rps, err
	}

	for _, cli := range clis {
		event := createClientEvent{
			Client:           cli,
			rolesProvisioned: rps,
			Session:          session,
			requestID:        middleware.GetReqID(ctx),
		}
		if err := es.Publish(ctx, createStream, event); err != nil {
			return clis, rps, err
		}
	}

	return clis, rps, nil
}

func (es *eventStore) Update(ctx context.Context, session authn.Session, client clients.Client) (clients.Client, error) {
	cli, err := es.svc.Update(ctx, session, client)
	if err != nil {
		return cli, err
	}

	return es.update(ctx, session, clientUpdate, updateStream, cli)
}

func (es *eventStore) UpdateTags(ctx context.Context, session authn.Session, client clients.Client) (clients.Client, error) {
	cli, err := es.svc.UpdateTags(ctx, session, client)
	if err != nil {
		return cli, err
	}

	return es.update(ctx, session, clientUpdateTags, updateTagsStream, cli)
}

func (es *eventStore) UpdateSecret(ctx context.Context, session authn.Session, id, key string) (clients.Client, error) {
	cli, err := es.svc.UpdateSecret(ctx, session, id, key)
	if err != nil {
		return cli, err
	}

	return es.update(ctx, session, clientUpdateSecret, updateSecretStream, cli)
}

func (es *eventStore) update(ctx context.Context, session authn.Session, operation, stream string, client clients.Client) (clients.Client, error) {
	event := updateClientEvent{
		Client:    client,
		operation: operation,
		Session:   session,
		requestID: middleware.GetReqID(ctx),
	}

	if err := es.Publish(ctx, stream, event); err != nil {
		return client, err
	}

	return client, nil
}

func (es *eventStore) View(ctx context.Context, session authn.Session, id string, withRoles bool) (clients.Client, error) {
	cli, err := es.svc.View(ctx, session, id, withRoles)
	if err != nil {
		return cli, err
	}

	event := viewClientEvent{
		Client:    cli,
		Session:   session,
		requestID: middleware.GetReqID(ctx),
	}
	if err := es.Publish(ctx, viewStream, event); err != nil {
		return cli, err
	}

	return cli, nil
}

func (es *eventStore) ListClients(ctx context.Context, session authn.Session, pm clients.Page) (clients.ClientsPage, error) {
	cp, err := es.svc.ListClients(ctx, session, pm)
	if err != nil {
		return cp, err
	}
	event := listClientEvent{
		pm,
		session,
		middleware.GetReqID(ctx),
	}
	if err := es.Publish(ctx, listStream, event); err != nil {
		return cp, err
	}

	return cp, nil
}

func (es *eventStore) ListUserClients(ctx context.Context, session authn.Session, userID string, pm clients.Page) (clients.ClientsPage, error) {
	cp, err := es.svc.ListUserClients(ctx, session, userID, pm)
	if err != nil {
		return cp, err
	}
	event := listUserClientEvent{
		userID,
		pm,
		session,
		middleware.GetReqID(ctx),
	}
	if err := es.Publish(ctx, listByUserStream, event); err != nil {
		return cp, err
	}

	return cp, nil
}

func (es *eventStore) Enable(ctx context.Context, session authn.Session, id string) (clients.Client, error) {
	cli, err := es.svc.Enable(ctx, session, id)
	if err != nil {
		return cli, err
	}

	return es.changeStatus(ctx, session, clientEnable, enableStream, cli)
}

func (es *eventStore) Disable(ctx context.Context, session authn.Session, id string) (clients.Client, error) {
	cli, err := es.svc.Disable(ctx, session, id)
	if err != nil {
		return cli, err
	}

	return es.changeStatus(ctx, session, clientDisable, disableStream, cli)
}

func (es *eventStore) changeStatus(ctx context.Context, session authn.Session, operation, stream string, cli clients.Client) (clients.Client, error) {
	event := changeClientStatusEvent{
		id:        cli.ID,
		operation: operation,
		updatedAt: cli.UpdatedAt,
		updatedBy: cli.UpdatedBy,
		status:    cli.Status.String(),
		Session:   session,
		requestID: middleware.GetReqID(ctx),
	}
	if err := es.Publish(ctx, stream, event); err != nil {
		return cli, err
	}

	return cli, nil
}

func (es *eventStore) Delete(ctx context.Context, session authn.Session, id string) error {
	if err := es.svc.Delete(ctx, session, id); err != nil {
		return err
	}

	event := removeClientEvent{
		id:        id,
		Session:   session,
		requestID: middleware.GetReqID(ctx),
	}

	if err := es.Publish(ctx, removeStream, event); err != nil {
		return err
	}

	return nil
}

func (es *eventStore) SetParentGroup(ctx context.Context, session authn.Session, parentGroupID string, id string) (err error) {
	if err := es.svc.SetParentGroup(ctx, session, parentGroupID, id); err != nil {
		return err
	}

	event := setParentGroupEvent{
		parentGroupID: parentGroupID,
		id:            id,
		Session:       session,
		requestID:     middleware.GetReqID(ctx),
	}

	if err := es.Publish(ctx, setParentStream, event); err != nil {
		return err
	}

	return nil
}

func (es *eventStore) RemoveParentGroup(ctx context.Context, session authn.Session, id string) (err error) {
	if err := es.svc.RemoveParentGroup(ctx, session, id); err != nil {
		return err
	}

	event := removeParentGroupEvent{
		id:        id,
		Session:   session,
		requestID: middleware.GetReqID(ctx),
	}

	if err := es.Publish(ctx, removeParentStream, event); err != nil {
		return err
	}

	return nil
}
