// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

// Package coap contains the domain concept definitions needed to support
// SuperMQ CoAP adapter service functionality. All constant values are taken
// from RFC, and could be adjusted based on specific use case.
package coap

import (
	"context"

	grpcChannelsV1 "github.com/absmach/supermq/api/grpc/channels/v1"
	grpcClientsV1 "github.com/absmach/supermq/api/grpc/clients/v1"
	"github.com/absmach/supermq/pkg/connections"
	"github.com/absmach/supermq/pkg/errors"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	"github.com/absmach/supermq/pkg/messaging"
	"github.com/absmach/supermq/pkg/policies"
)

var errFailedToDisconnectClient = errors.New("failed to disconnect client")

// Service specifies CoAP service API.
type Service interface {
	// Publish publishes message to specified channel.
	// Key is used to authorize publisher.
	Publish(ctx context.Context, key string, msg *messaging.Message) error

	// Subscribes to channel with specified id, domainID, subtopic and adds subscription to
	// service map of subscriptions under given ID.
	Subscribe(ctx context.Context, key, domainID, chanID, subtopic string, c Client) error

	// Unsubscribe method is used to stop observing resource.
	Unsubscribe(ctx context.Context, key, domainID, chanID, subptopic, token string) error

	// DisconnectHandler method is used to disconnected the client
	DisconnectHandler(ctx context.Context, domainID, chanID, subptopic, token string) error
}

var _ Service = (*adapterService)(nil)

// Observers is a map of maps,.
type adapterService struct {
	clients  grpcClientsV1.ClientsServiceClient
	channels grpcChannelsV1.ChannelsServiceClient
	pubsub   messaging.PubSub
}

// New instantiates the CoAP adapter implementation.
func New(clients grpcClientsV1.ClientsServiceClient, channels grpcChannelsV1.ChannelsServiceClient, pubsub messaging.PubSub) Service {
	as := &adapterService{
		clients:  clients,
		channels: channels,
		pubsub:   pubsub,
	}

	return as
}

func (svc *adapterService) Publish(ctx context.Context, key string, msg *messaging.Message) error {
	authnRes, err := svc.clients.Authenticate(ctx, &grpcClientsV1.AuthnReq{
		ClientSecret: key,
	})
	if err != nil {
		return errors.Wrap(svcerr.ErrAuthentication, err)
	}
	if !authnRes.Authenticated {
		return svcerr.ErrAuthentication
	}

	authzRes, err := svc.channels.Authorize(ctx, &grpcChannelsV1.AuthzReq{
		DomainId:   msg.GetDomain(),
		ClientId:   authnRes.GetId(),
		ClientType: policies.ClientType,
		Type:       uint32(connections.Publish),
		ChannelId:  msg.GetChannel(),
	})
	if err != nil {
		return errors.Wrap(svcerr.ErrAuthorization, err)
	}
	if !authzRes.Authorized {
		return svcerr.ErrAuthorization
	}

	msg.Publisher = authnRes.GetId()

	return svc.pubsub.Publish(ctx, messaging.EncodeMessageTopic(msg), msg)
}

func (svc *adapterService) Subscribe(ctx context.Context, key, domainID, chanID, subtopic string, c Client) error {
	authnRes, err := svc.clients.Authenticate(ctx, &grpcClientsV1.AuthnReq{
		ClientSecret: key,
	})
	if err != nil {
		return errors.Wrap(svcerr.ErrAuthentication, err)
	}
	if !authnRes.Authenticated {
		return svcerr.ErrAuthentication
	}

	clientID := authnRes.GetId()
	authzRes, err := svc.channels.Authorize(ctx, &grpcChannelsV1.AuthzReq{
		DomainId:   domainID,
		ClientId:   clientID,
		ClientType: policies.ClientType,
		Type:       uint32(connections.Subscribe),
		ChannelId:  chanID,
	})
	if err != nil {
		return errors.Wrap(svcerr.ErrAuthorization, err)
	}
	if !authzRes.Authorized {
		return svcerr.ErrAuthorization
	}

	subject := messaging.EncodeTopic(domainID, chanID, subtopic)
	authzc := newAuthzClient(clientID, domainID, chanID, subtopic, svc.channels, c)
	subCfg := messaging.SubscriberConfig{
		ID:       c.Token(),
		ClientID: clientID,
		Topic:    subject,
		Handler:  authzc,
	}
	return svc.pubsub.Subscribe(ctx, subCfg)
}

func (svc *adapterService) Unsubscribe(ctx context.Context, key, domainID, chanID, subtopic, token string) error {
	authnRes, err := svc.clients.Authenticate(ctx, &grpcClientsV1.AuthnReq{
		ClientSecret: key,
	})
	if err != nil {
		return errors.Wrap(svcerr.ErrAuthentication, err)
	}
	if !authnRes.Authenticated {
		return svcerr.ErrAuthentication
	}

	authzRes, err := svc.channels.Authorize(ctx, &grpcChannelsV1.AuthzReq{
		DomainId:   domainID,
		ClientId:   authnRes.GetId(),
		ClientType: policies.ClientType,
		Type:       uint32(connections.Subscribe),
		ChannelId:  chanID,
	})
	if err != nil {
		return errors.Wrap(svcerr.ErrAuthorization, err)
	}
	if !authzRes.Authorized {
		return svcerr.ErrAuthorization
	}

	subject := messaging.EncodeTopic(domainID, chanID, subtopic)

	return svc.pubsub.Unsubscribe(ctx, token, subject)
}

func (svc *adapterService) DisconnectHandler(ctx context.Context, domainID, chanID, subtopic, token string) error {
	subject := messaging.EncodeTopic(domainID, chanID, subtopic)

	return svc.pubsub.Unsubscribe(ctx, token, subject)
}

type authzClient interface {
	// Handle handles incoming messages.
	Handle(m *messaging.Message) error

	// Cancel cancels the client.
	Cancel() error
}

type ac struct {
	clientID  string
	channelID string
	domainID  string
	subTopic  string
	channels  grpcChannelsV1.ChannelsServiceClient
	client    Client
}

func newAuthzClient(clientID, domainID, channelID, subTopic string, channels grpcChannelsV1.ChannelsServiceClient, client Client) authzClient {
	return ac{clientID, channelID, domainID, subTopic, channels, client}
}

func (a ac) Handle(m *messaging.Message) error {
	res, err := a.channels.Authorize(context.Background(), &grpcChannelsV1.AuthzReq{
		ClientId:   a.clientID,
		ClientType: policies.ClientType,
		ChannelId:  a.channelID,
		DomainId:   a.domainID,
		Type:       uint32(connections.Subscribe),
	})
	if err != nil {
		if disErr := a.Cancel(); disErr != nil {
			return errors.Wrap(err, errors.Wrap(errFailedToDisconnectClient, disErr))
		}
		return err
	}
	if !res.GetAuthorized() {
		err := svcerr.ErrAuthorization
		if disErr := a.Cancel(); disErr != nil {
			return errors.Wrap(err, errors.Wrap(errFailedToDisconnectClient, disErr))
		}
		return err
	}
	return a.client.Handle(m)
}

func (a ac) Cancel() error {
	return a.client.Cancel()
}
