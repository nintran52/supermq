// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package private

import (
	"context"

	"github.com/absmach/supermq/auth"
	"github.com/absmach/supermq/channels"
	dom "github.com/absmach/supermq/domains"
	pkgDomains "github.com/absmach/supermq/pkg/domains"
	"github.com/absmach/supermq/pkg/errors"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	"github.com/absmach/supermq/pkg/policies"
)

var errDisabledDomain = errors.New("domain is disabled or frozen")

type Service interface {
	Authorize(ctx context.Context, req channels.AuthzReq) error
	UnsetParentGroupFromChannels(ctx context.Context, parentGroupID string) error
	RemoveClientConnections(ctx context.Context, clientID string) error
	RetrieveByID(ctx context.Context, id string) (channels.Channel, error)
	RetrieveByRoute(ctx context.Context, route, domainID string) (channels.Channel, error)
}

type service struct {
	repo      channels.Repository
	cache     channels.Cache
	evaluator policies.Evaluator
	policy    policies.Service
	domains   pkgDomains.Authorization
}

var _ Service = (*service)(nil)

func New(repo channels.Repository, cache channels.Cache, evaluator policies.Evaluator, policy policies.Service, domains pkgDomains.Authorization) Service {
	return service{repo, cache, evaluator, policy, domains}
}

func (svc service) Authorize(ctx context.Context, req channels.AuthzReq) error {
	d, err := svc.domains.RetrieveEntity(ctx, req.DomainID)
	if err != nil {
		return errors.Wrap(svcerr.ErrAuthorization, err)
	}
	if d.Status != dom.EnabledStatus {
		return errors.Wrap(svcerr.ErrAuthorization, errDisabledDomain)
	}
	switch req.ClientType {
	case policies.UserType:
		permission, err := req.Type.Permission()
		if err != nil {
			return err
		}
		pr := policies.Policy{
			Subject:     auth.EncodeDomainUserID(req.DomainID, req.ClientID),
			SubjectType: policies.UserType,
			Object:      req.ChannelID,
			Permission:  permission,
			ObjectType:  policies.ChannelType,
		}
		if err := svc.evaluator.CheckPolicy(ctx, pr); err != nil {
			return errors.Wrap(svcerr.ErrAuthorization, err)
		}
		return nil
	case policies.ClientType:
		// Optimization: Add cache
		if err := svc.repo.ClientAuthorize(ctx, channels.Connection{
			ChannelID: req.ChannelID,
			ClientID:  req.ClientID,
			Type:      req.Type,
		}); err != nil {
			return errors.Wrap(svcerr.ErrAuthorization, err)
		}
		return nil
	default:
		return svcerr.ErrAuthentication
	}
}

func (svc service) RemoveClientConnections(ctx context.Context, clientID string) error {
	return svc.repo.RemoveClientConnections(ctx, clientID)
}

func (svc service) UnsetParentGroupFromChannels(ctx context.Context, parentGroupID string) (retErr error) {
	chs, err := svc.repo.RetrieveParentGroupChannels(ctx, parentGroupID)
	if err != nil {
		return errors.Wrap(svcerr.ErrViewEntity, err)
	}

	if len(chs) > 0 {
		prs := []policies.Policy{}
		for _, ch := range chs {
			prs = append(prs, policies.Policy{
				SubjectType: policies.GroupType,
				Subject:     ch.ParentGroup,
				Relation:    policies.ParentGroupRelation,
				ObjectType:  policies.ChannelType,
				Object:      ch.ID,
			})
		}

		if err := svc.policy.DeletePolicies(ctx, prs); err != nil {
			return errors.Wrap(svcerr.ErrDeletePolicies, err)
		}
		defer func() {
			if retErr != nil {
				if errRollback := svc.policy.AddPolicies(ctx, prs); err != nil {
					retErr = errors.Wrap(retErr, errors.Wrap(errors.ErrRollbackTx, errRollback))
				}
			}
		}()

		if err := svc.repo.UnsetParentGroupFromChannels(ctx, parentGroupID); err != nil {
			return errors.Wrap(svcerr.ErrRemoveEntity, err)
		}
	}
	return nil
}

func (svc service) RetrieveByID(ctx context.Context, id string) (channels.Channel, error) {
	return svc.repo.RetrieveByID(ctx, id)
}

func (svc service) RetrieveByRoute(ctx context.Context, route, domainID string) (channels.Channel, error) {
	id, err := svc.cache.ID(ctx, route, domainID)
	if err == nil {
		return channels.Channel{ID: id}, nil
	}
	chn, err := svc.repo.RetrieveByRoute(ctx, route, domainID)
	if err != nil {
		return channels.Channel{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}
	if err := svc.cache.Save(ctx, route, domainID, chn.ID); err != nil {
		return channels.Channel{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}

	return channels.Channel{ID: chn.ID}, nil
}
