// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"context"
	"maps"
	"time"

	"github.com/absmach/supermq/auth"
	"github.com/absmach/supermq/domains"
	"github.com/absmach/supermq/pkg/authn"
	"github.com/absmach/supermq/pkg/authz"
	smqauthz "github.com/absmach/supermq/pkg/authz"
	"github.com/absmach/supermq/pkg/callout"
	"github.com/absmach/supermq/pkg/errors"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	"github.com/absmach/supermq/pkg/policies"
	"github.com/absmach/supermq/pkg/roles"
	rmMW "github.com/absmach/supermq/pkg/roles/rolemanager/middleware"
	"github.com/absmach/supermq/pkg/svcutil"
)

var _ domains.Service = (*authorizationMiddleware)(nil)

// ErrMemberExist indicates that the user is already a member of the domain.
var ErrMemberExist = errors.New("user is already a member of the domain")

type authorizationMiddleware struct {
	svc     domains.Service
	authz   smqauthz.Authorization
	opp     svcutil.OperationPerm
	callout callout.Callout
	rmMW.RoleManagerAuthorizationMiddleware
}

// AuthorizationMiddleware adds authorization to the clients service.
func AuthorizationMiddleware(entityType string, svc domains.Service, authz smqauthz.Authorization, domainsOpPerm, rolesOpPerm map[svcutil.Operation]svcutil.Permission, callout callout.Callout) (domains.Service, error) {
	opp := domains.NewOperationPerm()
	if err := opp.AddOperationPermissionMap(domainsOpPerm); err != nil {
		return nil, err
	}
	if err := opp.Validate(); err != nil {
		return nil, err
	}

	ram, err := rmMW.NewRoleManagerAuthorizationMiddleware(entityType, svc, authz, rolesOpPerm, callout)
	if err != nil {
		return nil, err
	}
	return &authorizationMiddleware{
		svc:                                svc,
		authz:                              authz,
		opp:                                opp,
		callout:                            callout,
		RoleManagerAuthorizationMiddleware: ram,
	}, nil
}

func (am *authorizationMiddleware) CreateDomain(ctx context.Context, session authn.Session, d domains.Domain) (domains.Domain, []roles.RoleProvision, error) {
	params := map[string]any{
		"domain": d,
	}
	if err := am.callOut(ctx, session, domains.OpCreateDomain.String(domains.OperationNames), params); err != nil {
		return domains.Domain{}, nil, err
	}
	return am.svc.CreateDomain(ctx, session, d)
}

func (am *authorizationMiddleware) RetrieveDomain(ctx context.Context, session authn.Session, id string, withRoles bool) (domains.Domain, error) {
	if err := am.checkSuperAdmin(ctx, session.UserID); err == nil {
		session.SuperAdmin = true
		return am.svc.RetrieveDomain(ctx, session, id, withRoles)
	}

	if err := am.authorize(ctx, domains.OpRetrieveDomain, authz.PolicyReq{
		Subject:     session.DomainUserID,
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Object:      id,
		ObjectType:  policies.DomainType,
	}); err != nil {
		return domains.Domain{}, err
	}
	params := map[string]any{
		"domain":     id,
		"with_roles": withRoles,
	}
	if err := am.callOut(ctx, session, domains.OpRetrieveDomain.String(domains.OperationNames), params); err != nil {
		return domains.Domain{}, err
	}
	return am.svc.RetrieveDomain(ctx, session, id, withRoles)
}

func (am *authorizationMiddleware) UpdateDomain(ctx context.Context, session authn.Session, id string, d domains.DomainReq) (domains.Domain, error) {
	if err := am.authorize(ctx, domains.OpUpdateDomain, authz.PolicyReq{
		Subject:     session.DomainUserID,
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Object:      id,
		ObjectType:  policies.DomainType,
	}); err != nil {
		return domains.Domain{}, err
	}
	params := map[string]any{
		"domain":     id,
		"domain_req": d,
	}
	if err := am.callOut(ctx, session, domains.OpUpdateDomain.String(domains.OperationNames), params); err != nil {
		return domains.Domain{}, err
	}
	return am.svc.UpdateDomain(ctx, session, id, d)
}

func (am *authorizationMiddleware) EnableDomain(ctx context.Context, session authn.Session, id string) (domains.Domain, error) {
	if err := am.authorize(ctx, domains.OpEnableDomain, authz.PolicyReq{
		Subject:     session.DomainUserID,
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Object:      id,
		ObjectType:  policies.DomainType,
	}); err != nil {
		return domains.Domain{}, err
	}
	params := map[string]any{
		"domain": id,
	}
	if err := am.callOut(ctx, session, domains.OpEnableDomain.String(domains.OperationNames), params); err != nil {
		return domains.Domain{}, err
	}
	return am.svc.EnableDomain(ctx, session, id)
}

func (am *authorizationMiddleware) DisableDomain(ctx context.Context, session authn.Session, id string) (domains.Domain, error) {
	if err := am.authorize(ctx, domains.OpDisableDomain, authz.PolicyReq{
		Subject:     session.DomainUserID,
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Object:      id,
		ObjectType:  policies.DomainType,
	}); err != nil {
		return domains.Domain{}, err
	}
	params := map[string]any{
		"domain": id,
	}
	if err := am.callOut(ctx, session, domains.OpDisableDomain.String(domains.OperationNames), params); err != nil {
		return domains.Domain{}, err
	}
	return am.svc.DisableDomain(ctx, session, id)
}

func (am *authorizationMiddleware) FreezeDomain(ctx context.Context, session authn.Session, id string) (domains.Domain, error) {
	// Only SuperAdmin can freeze the domain
	if err := am.authz.Authorize(ctx, authz.PolicyReq{
		Subject:     session.UserID,
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Permission:  policies.AdminPermission,
		Object:      policies.SuperMQObject,
		ObjectType:  policies.PlatformType,
	}); err != nil {
		return domains.Domain{}, err
	}
	params := map[string]any{
		"domain": id,
	}
	if err := am.callOut(ctx, session, domains.OpFreezeDomain.String(domains.OperationNames), params); err != nil {
		return domains.Domain{}, err
	}
	return am.svc.FreezeDomain(ctx, session, id)
}

func (am *authorizationMiddleware) ListDomains(ctx context.Context, session authn.Session, page domains.Page) (domains.DomainsPage, error) {
	if err := am.checkSuperAdmin(ctx, session.UserID); err == nil {
		session.SuperAdmin = true
	}
	params := map[string]any{
		"page": page,
	}
	if err := am.callOut(ctx, session, domains.OpListDomains.String(domains.OperationNames), params); err != nil {
		return domains.DomainsPage{}, err
	}
	return am.svc.ListDomains(ctx, session, page)
}

func (am *authorizationMiddleware) SendInvitation(ctx context.Context, session authn.Session, invitation domains.Invitation) (err error) {
	domainUserId := auth.EncodeDomainUserID(invitation.DomainID, invitation.InviteeUserID)
	if err := am.extAuthorize(ctx, domainUserId, policies.MembershipPermission, policies.DomainType, invitation.DomainID); err == nil {
		// return error if the user is already a member of the domain
		return errors.Wrap(svcerr.ErrConflict, ErrMemberExist)
	}

	if err := am.checkAdmin(ctx, session); err != nil {
		return err
	}
	params := map[string]any{
		"invitation": invitation,
		"domain":     invitation.DomainID,
	}
	if err := am.callOut(ctx, session, domains.OpSendInvitation.String(domains.OperationNames), params); err != nil {
		return err
	}

	return am.svc.SendInvitation(ctx, session, invitation)
}

func (am *authorizationMiddleware) ViewInvitation(ctx context.Context, session authn.Session, inviteeUserID, domain string) (invitation domains.Invitation, err error) {
	session.DomainUserID = auth.EncodeDomainUserID(session.DomainID, session.UserID)
	if session.UserID != inviteeUserID {
		if err := am.checkAdmin(ctx, session); err != nil {
			return domains.Invitation{}, err
		}
	}

	params := map[string]any{
		"invitee_user_id": inviteeUserID,
		"domain":          domain,
	}
	if err := am.callOut(ctx, session, domains.OpViewInvitation.String(domains.OperationNames), params); err != nil {
		return domains.Invitation{}, err
	}

	return am.svc.ViewInvitation(ctx, session, inviteeUserID, domain)
}

func (am *authorizationMiddleware) ListInvitations(ctx context.Context, session authn.Session, page domains.InvitationPageMeta) (invs domains.InvitationPage, err error) {
	session.DomainUserID = auth.EncodeDomainUserID(session.DomainID, session.UserID)
	if err := am.extAuthorize(ctx, session.UserID, policies.AdminPermission, policies.PlatformType, policies.SuperMQObject); err == nil {
		session.SuperAdmin = true
		page.DomainID = ""
	}

	if !session.SuperAdmin {
		switch {
		case page.DomainID != "":
			if err := am.extAuthorize(ctx, session.DomainUserID, policies.AdminPermission, policies.DomainType, page.DomainID); err != nil {
				return domains.InvitationPage{}, err
			}
		default:
			page.InvitedByOrUserID = session.UserID
		}
	}

	params := map[string]any{
		"page": page,
	}
	if err := am.callOut(ctx, session, domains.OpListInvitations.String(domains.OperationNames), params); err != nil {
		return domains.InvitationPage{}, err
	}

	return am.svc.ListInvitations(ctx, session, page)
}

func (am *authorizationMiddleware) AcceptInvitation(ctx context.Context, session authn.Session, domainID string) (inv domains.Invitation, err error) {
	params := map[string]any{
		"domain": domainID,
	}
	if err := am.callOut(ctx, session, domains.OpAcceptInvitation.String(domains.OperationNames), params); err != nil {
		return domains.Invitation{}, err
	}
	return am.svc.AcceptInvitation(ctx, session, domainID)
}

func (am *authorizationMiddleware) RejectInvitation(ctx context.Context, session authn.Session, domainID string) (err error) {
	params := map[string]any{
		"domain": domainID,
	}
	if err := am.callOut(ctx, session, domains.OpRejectInvitation.String(domains.OperationNames), params); err != nil {
		return err
	}
	return am.svc.RejectInvitation(ctx, session, domainID)
}

func (am *authorizationMiddleware) DeleteInvitation(ctx context.Context, session authn.Session, inviteeUserID, domainID string) (err error) {
	session.DomainUserID = auth.EncodeDomainUserID(session.DomainID, session.UserID)
	if err := am.checkAdmin(ctx, session); err != nil {
		return err
	}

	params := map[string]any{
		"invitee_user_id": inviteeUserID,
		"domain":          domainID,
	}
	if err := am.callOut(ctx, session, domains.OpDeleteInvitation.String(domains.OperationNames), params); err != nil {
		return err
	}

	return am.svc.DeleteInvitation(ctx, session, inviteeUserID, domainID)
}

func (am *authorizationMiddleware) authorize(ctx context.Context, op svcutil.Operation, authReq authz.PolicyReq) error {
	perm, err := am.opp.GetPermission(op)
	if err != nil {
		return err
	}
	authReq.Permission = perm.String()

	if err := am.authz.Authorize(ctx, authReq); err != nil {
		return err
	}

	return nil
}

// checkAdmin checks if the given user is a domain or platform administrator.
func (am *authorizationMiddleware) checkAdmin(ctx context.Context, session authn.Session) error {
	req := smqauthz.PolicyReq{
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Subject:     session.DomainUserID,
		Permission:  policies.AdminPermission,
		ObjectType:  policies.DomainType,
		Object:      session.DomainID,
	}
	if err := am.authz.Authorize(ctx, req); err == nil {
		return nil
	}

	req = smqauthz.PolicyReq{
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Subject:     session.UserID,
		Permission:  policies.AdminPermission,
		ObjectType:  policies.PlatformType,
		Object:      policies.SuperMQObject,
	}

	if err := am.authz.Authorize(ctx, req); err == nil {
		return nil
	}

	return svcerr.ErrAuthorization
}

func (am *authorizationMiddleware) checkSuperAdmin(ctx context.Context, userID string) error {
	if err := am.authz.Authorize(ctx, smqauthz.PolicyReq{
		SubjectType: policies.UserType,
		Subject:     userID,
		Permission:  policies.AdminPermission,
		ObjectType:  policies.PlatformType,
		Object:      policies.SuperMQObject,
	}); err != nil {
		return err
	}
	return nil
}

func (am *authorizationMiddleware) extAuthorize(ctx context.Context, subj, perm, objType, obj string) error {
	req := authz.PolicyReq{
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Subject:     subj,
		Permission:  perm,
		ObjectType:  objType,
		Object:      obj,
	}
	if err := am.authz.Authorize(ctx, req); err != nil {
		return err
	}

	return nil
}

func (am *authorizationMiddleware) callOut(ctx context.Context, session authn.Session, op string, params map[string]interface{}) error {
	pl := map[string]any{
		"entity_type":  policies.DomainType,
		"subject_type": policies.UserType,
		"subject_id":   session.UserID,
		"time":         time.Now().UTC(),
	}

	maps.Copy(params, pl)

	if err := am.callout.Callout(ctx, op, params); err != nil {
		return err
	}

	return nil
}
