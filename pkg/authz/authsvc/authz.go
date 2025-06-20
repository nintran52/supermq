// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package authsvc

import (
	"context"

	grpcAuthV1 "github.com/absmach/supermq/api/grpc/auth/v1"
	"github.com/absmach/supermq/auth/api/grpc/auth"
	"github.com/absmach/supermq/domains"
	"github.com/absmach/supermq/pkg/authz"
	pkgDomians "github.com/absmach/supermq/pkg/domains"
	"github.com/absmach/supermq/pkg/errors"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	"github.com/absmach/supermq/pkg/grpcclient"
	"github.com/absmach/supermq/pkg/policies"
	grpchealth "google.golang.org/grpc/health/grpc_health_v1"
)

type authorization struct {
	authSvcClient grpcAuthV1.AuthServiceClient
	domains       pkgDomians.Authorization
}

var _ authz.Authorization = (*authorization)(nil)

func NewAuthorization(ctx context.Context, cfg grpcclient.Config, domainsAuthz pkgDomians.Authorization) (authz.Authorization, grpcclient.Handler, error) {
	client, err := grpcclient.NewHandler(cfg)
	if err != nil {
		return nil, nil, err
	}

	health := grpchealth.NewHealthClient(client.Connection())
	resp, err := health.Check(ctx, &grpchealth.HealthCheckRequest{
		Service: "auth",
	})
	if err != nil || resp.GetStatus() != grpchealth.HealthCheckResponse_SERVING {
		return nil, nil, grpcclient.ErrSvcNotServing
	}

	authSvcClient := auth.NewAuthClient(client.Connection(), cfg.Timeout)
	return authorization{
		authSvcClient: authSvcClient,
		domains:       domainsAuthz,
	}, client, nil
}

func (a authorization) Authorize(ctx context.Context, pr authz.PolicyReq) error {
	if pr.SubjectType == policies.UserType && (pr.ObjectType == policies.GroupType || pr.ObjectType == policies.ClientType || pr.ObjectType == policies.DomainType) {
		domainID := pr.Domain
		if domainID == "" {
			if pr.ObjectType != policies.DomainType {
				return svcerr.ErrDomainAuthorization
			}
			domainID = pr.Object
		}
		if err := a.checkDomain(ctx, pr.SubjectType, pr.Subject, domainID); err != nil {
			return errors.Wrap(svcerr.ErrDomainAuthorization, err)
		}
	}

	req := grpcAuthV1.AuthZReq{
		Domain:          pr.Domain,
		SubjectType:     pr.SubjectType,
		SubjectKind:     pr.SubjectKind,
		SubjectRelation: pr.SubjectRelation,
		Subject:         pr.Subject,
		Relation:        pr.Relation,
		Permission:      pr.Permission,
		Object:          pr.Object,
		ObjectType:      pr.ObjectType,
	}
	res, err := a.authSvcClient.Authorize(ctx, &req)
	if err != nil {
		return errors.Wrap(errors.ErrAuthorization, err)
	}
	if !res.GetAuthorized() {
		return errors.ErrAuthorization
	}
	return nil
}

func (a authorization) checkDomain(ctx context.Context, subjectType, subject, domainID string) error {
	dom, err := a.domains.RetrieveEntity(ctx, domainID)
	if err != nil {
		return errors.Wrap(svcerr.ErrViewEntity, err)
	}

	switch dom.Status {
	case domains.FreezeStatus:
		_, err := a.authSvcClient.Authorize(ctx, &grpcAuthV1.AuthZReq{
			Subject:     subject,
			SubjectType: subjectType,
			Permission:  policies.AdminPermission,
			Object:      policies.SuperMQObject,
			ObjectType:  policies.PlatformType,
		})

		return err
	case domains.DisabledStatus:
		_, err := a.authSvcClient.Authorize(ctx, &grpcAuthV1.AuthZReq{
			Subject:     subject,
			SubjectType: subjectType,
			Permission:  policies.AdminPermission,
			Object:      domainID,
			ObjectType:  policies.DomainType,
		})

		return err
	case domains.EnabledStatus:
		_, err := a.authSvcClient.Authorize(ctx, &grpcAuthV1.AuthZReq{
			Subject:     subject,
			SubjectType: subjectType,
			Permission:  policies.MembershipPermission,
			Object:      domainID,
			ObjectType:  policies.DomainType,
		})

		return err
	default:
		return svcerr.ErrInvalidStatus
	}
}

func (a authorization) AuthorizePAT(ctx context.Context, pr authz.PatReq) error {
	req := grpcAuthV1.AuthZPatReq{
		UserId:           pr.UserID,
		PatId:            pr.PatID,
		EntityType:       uint32(pr.EntityType),
		OptionalDomainId: pr.OptionalDomainID,
		Operation:        uint32(pr.Operation),
		EntityId:         pr.EntityID,
	}
	res, err := a.authSvcClient.AuthorizePAT(ctx, &req)
	if err != nil {
		return errors.Wrap(errors.ErrAuthorization, err)
	}
	if !res.Authorized {
		return errors.ErrAuthorization
	}
	return nil
}
