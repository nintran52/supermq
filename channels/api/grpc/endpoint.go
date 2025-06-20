// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"context"

	ch "github.com/absmach/supermq/channels"
	channels "github.com/absmach/supermq/channels/private"
	"github.com/go-kit/kit/endpoint"
)

func authorizeEndpoint(svc channels.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(authorizeReq)
		if err := req.validate(); err != nil {
			return authorizeRes{}, err
		}

		if err := svc.Authorize(ctx, ch.AuthzReq{
			DomainID:   req.domainID,
			ClientID:   req.clientID,
			ClientType: req.clientType,
			ChannelID:  req.channelID,
			Type:       req.connType,
		}); err != nil {
			return authorizeRes{}, err
		}

		return authorizeRes{authorized: true}, nil
	}
}

func removeClientConnectionsEndpoint(svc channels.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(removeClientConnectionsReq)

		if err := svc.RemoveClientConnections(ctx, req.clientID); err != nil {
			return removeClientConnectionsRes{}, err
		}

		return removeClientConnectionsRes{}, nil
	}
}

func unsetParentGroupFromChannelsEndpoint(svc channels.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(unsetParentGroupFromChannelsReq)

		if err := svc.UnsetParentGroupFromChannels(ctx, req.parentGroupID); err != nil {
			return unsetParentGroupFromChannelsRes{}, err
		}

		return unsetParentGroupFromChannelsRes{}, nil
	}
}

func retrieveEntityEndpoint(svc channels.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(retrieveEntityReq)
		channel, err := svc.RetrieveByID(ctx, req.Id)
		if err != nil {
			return retrieveEntityRes{}, err
		}

		return retrieveEntityRes{id: channel.ID, domain: channel.Domain, parentGroup: channel.ParentGroup, status: uint8(channel.Status)}, nil
	}
}

func retrieveByRouteEndpoint(svc channels.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(retrieveByRouteReq)
		if err := req.validate(); err != nil {
			return retrieveEntityRes{}, err
		}

		channel, err := svc.RetrieveByRoute(ctx, req.route, req.domainID)
		if err != nil {
			return retrieveEntityRes{}, err
		}

		return retrieveEntityRes{id: channel.ID, status: uint8(channel.Status)}, nil
	}
}
