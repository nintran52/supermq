// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"

	api "github.com/absmach/supermq/api/http"
	apiutil "github.com/absmach/supermq/api/http/util"
	"github.com/absmach/supermq/pkg/authn"
	"github.com/absmach/supermq/pkg/errors"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	"github.com/absmach/supermq/users"
	"github.com/go-kit/kit/endpoint"
)

func registrationEndpoint(svc users.Service, selfRegister bool) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createUserReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}
		session := authn.Session{}

		var ok bool
		if !selfRegister {
			session, ok = ctx.Value(api.SessionKey).(authn.Session)
			if !ok {
				return nil, svcerr.ErrAuthentication
			}
		}

		user, err := svc.Register(ctx, session, req.User, selfRegister)
		if err != nil {
			return nil, err
		}

		return createUserRes{
			User:    user,
			created: true,
		}, nil
	}
}

func viewEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(viewUserReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthentication
		}
		user, err := svc.View(ctx, session, req.id)
		if err != nil {
			return nil, err
		}

		return viewUserRes{User: user}, nil
	}
}

func viewProfileEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthentication
		}
		client, err := svc.ViewProfile(ctx, session)
		if err != nil {
			return nil, err
		}

		return viewUserRes{User: client}, nil
	}
}

func listUsersEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listUsersReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthentication
		}

		pm := users.Page{
			Status:    req.status,
			Offset:    req.offset,
			Limit:     req.limit,
			Username:  req.userName,
			Tag:       req.tag,
			Metadata:  req.metadata,
			FirstName: req.firstName,
			LastName:  req.lastName,
			Email:     req.email,
			Order:     req.order,
			Dir:       req.dir,
			Id:        req.id,
		}

		page, err := svc.ListUsers(ctx, session, pm)
		if err != nil {
			return nil, err
		}

		res := usersPageRes{
			pageRes: pageRes{
				Total:  page.Total,
				Offset: page.Offset,
				Limit:  page.Limit,
			},
			Users: []viewUserRes{},
		}
		for _, user := range page.Users {
			res.Users = append(res.Users, viewUserRes{User: user})
		}

		return res, nil
	}
}

func searchUsersEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(searchUsersReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		pm := users.Page{
			Offset:    req.Offset,
			Limit:     req.Limit,
			Username:  req.Username,
			FirstName: req.FirstName,
			LastName:  req.LastName,
			Id:        req.Id,
			Order:     req.Order,
			Dir:       req.Dir,
		}
		page, err := svc.SearchUsers(ctx, pm)
		if err != nil {
			return nil, err
		}

		res := usersPageRes{
			pageRes: pageRes{
				Total:  page.Total,
				Offset: page.Offset,
				Limit:  page.Limit,
			},
			Users: []viewUserRes{},
		}
		for _, user := range page.Users {
			res.Users = append(res.Users, viewUserRes{User: user})
		}

		return res, nil
	}
}

func updateEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(updateUserReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthentication
		}

		usr := users.UserReq{
			FirstName: req.FirstName,
			LastName:  req.LastName,
			Metadata:  req.Metadata,
		}

		user, err := svc.Update(ctx, session, req.id, usr)
		if err != nil {
			return nil, err
		}

		return updateUserRes{User: user}, nil
	}
}

func updateTagsEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(updateUserTagsReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthentication
		}

		usr := users.UserReq{
			Tags: req.Tags,
		}

		user, err := svc.UpdateTags(ctx, session, req.id, usr)
		if err != nil {
			return nil, err
		}

		return updateUserRes{User: user}, nil
	}
}

func updateEmailEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(updateEmailReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthentication
		}

		user, err := svc.UpdateEmail(ctx, session, req.id, req.Email)
		if err != nil {
			return nil, err
		}

		return updateUserRes{User: user}, nil
	}
}

// Password reset request endpoint.
// When successful password reset link is generated.
// Link is generated using SMQ_TOKEN_RESET_ENDPOINT env.
// and value from Referer header for host.
// {Referer}+{SMQ_TOKEN_RESET_ENDPOINT}+{token=TOKEN}
// http://supermq.com/reset-request?token=xxxxxxxxxxx.
// Email with a link is being sent to the user.
// When user clicks on a link it should get the ui with form to
// enter new password, when form is submitted token and new password
// must be sent as PUT request to 'password/reset' passwordResetEndpoint.
func passwordResetRequestEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(passwResetReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		if err := svc.GenerateResetToken(ctx, req.Email, req.Host); err != nil {
			return nil, err
		}

		return passwResetReqRes{Msg: MailSent}, nil
	}
}

// This is endpoint that actually sets new password in password reset flow.
// When user clicks on a link in email finally ends on this endpoint as explained in
// the comment above.
func passwordResetEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(resetTokenReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthentication
		}
		if err := svc.ResetSecret(ctx, session, req.Password); err != nil {
			return nil, err
		}

		return passwChangeRes{}, nil
	}
}

func updateSecretEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(updateUserSecretReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthentication
		}
		user, err := svc.UpdateSecret(ctx, session, req.OldSecret, req.NewSecret)
		if err != nil {
			return nil, err
		}

		return updateUserRes{User: user}, nil
	}
}

func updateUsernameEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(updateUsernameReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthorization
		}

		user, err := svc.UpdateUsername(ctx, session, req.id, req.Username)
		if err != nil {
			return nil, err
		}

		return updateUserRes{User: user}, nil
	}
}

func updateProfilePictureEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(updateProfilePictureReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		usr := users.UserReq{
			ProfilePicture: req.ProfilePicture,
		}

		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthorization
		}

		user, err := svc.UpdateProfilePicture(ctx, session, req.id, usr)
		if err != nil {
			return nil, err
		}

		return updateUserRes{User: user}, nil
	}
}

func updateRoleEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(updateUserRoleReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		user := users.User{
			ID:   req.id,
			Role: req.role,
		}

		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthentication
		}

		user, err := svc.UpdateRole(ctx, session, user)
		if err != nil {
			return nil, err
		}

		return updateUserRes{User: user}, nil
	}
}

func issueTokenEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(loginUserReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		token, err := svc.IssueToken(ctx, req.Username, req.Password)
		if err != nil {
			return nil, err
		}

		return tokenRes{
			AccessToken:  token.GetAccessToken(),
			RefreshToken: token.GetRefreshToken(),
			AccessType:   token.GetAccessType(),
		}, nil
	}
}

func refreshTokenEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(tokenReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthentication
		}

		token, err := svc.RefreshToken(ctx, session, req.RefreshToken)
		if err != nil {
			return nil, err
		}

		return tokenRes{
			AccessToken:  token.GetAccessToken(),
			RefreshToken: token.GetRefreshToken(),
			AccessType:   token.GetAccessType(),
		}, nil
	}
}

func enableEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(changeUserStatusReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthentication
		}

		user, err := svc.Enable(ctx, session, req.id)
		if err != nil {
			return nil, err
		}

		return changeUserStatusRes{User: user}, nil
	}
}

func disableEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(changeUserStatusReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthentication
		}

		user, err := svc.Disable(ctx, session, req.id)
		if err != nil {
			return nil, err
		}

		return changeUserStatusRes{User: user}, nil
	}
}

func deleteEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(changeUserStatusReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthentication
		}

		if err := svc.Delete(ctx, session, req.id); err != nil {
			return nil, err
		}

		return deleteUserRes{true}, nil
	}
}
