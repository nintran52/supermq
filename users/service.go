// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package users

import (
	"context"
	"net/mail"
	"time"

	"github.com/absmach/supermq"
	grpcTokenV1 "github.com/absmach/supermq/api/grpc/token/v1"
	apiutil "github.com/absmach/supermq/api/http/util"
	smqauth "github.com/absmach/supermq/auth"
	"github.com/absmach/supermq/pkg/authn"
	"github.com/absmach/supermq/pkg/errors"
	repoerr "github.com/absmach/supermq/pkg/errors/repository"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	"github.com/absmach/supermq/pkg/policies"
)

var (
	errIssueToken       = errors.New("failed to issue token")
	errRecoveryToken    = errors.New("failed to generate password recovery token")
	errLoginDisableUser = errors.New("failed to login in disabled user")
)

type service struct {
	token      grpcTokenV1.TokenServiceClient
	users      Repository
	idProvider supermq.IDProvider
	policies   policies.Service
	hasher     Hasher
	email      Emailer
}

// NewService returns a new Users service implementation.
func NewService(token grpcTokenV1.TokenServiceClient, urepo Repository, policyService policies.Service, emailer Emailer, hasher Hasher, idp supermq.IDProvider) Service {
	return service{
		token:      token,
		users:      urepo,
		policies:   policyService,
		hasher:     hasher,
		email:      emailer,
		idProvider: idp,
	}
}

func (svc service) Register(ctx context.Context, session authn.Session, u User, selfRegister bool) (uc User, err error) {
	if !selfRegister {
		if err := svc.checkSuperAdmin(ctx, session); err != nil {
			return User{}, err
		}
	}

	userID, err := svc.idProvider.ID()
	if err != nil {
		return User{}, err
	}

	if u.Credentials.Secret != "" {
		hash, err := svc.hasher.Hash(u.Credentials.Secret)
		if err != nil {
			return User{}, errors.Wrap(svcerr.ErrMalformedEntity, err)
		}
		u.Credentials.Secret = hash
	}

	if u.Status != DisabledStatus && u.Status != EnabledStatus {
		return User{}, errors.Wrap(svcerr.ErrMalformedEntity, svcerr.ErrInvalidStatus)
	}
	if u.Role != UserRole && u.Role != AdminRole {
		return User{}, errors.Wrap(svcerr.ErrMalformedEntity, svcerr.ErrInvalidRole)
	}
	u.ID = userID
	u.CreatedAt = time.Now().UTC()

	if err := svc.addUserPolicy(ctx, u.ID, u.Role); err != nil {
		return User{}, err
	}
	defer func() {
		if err != nil {
			if errRollback := svc.addUserPolicyRollback(ctx, u.ID, u.Role); errRollback != nil {
				err = errors.Wrap(errors.Wrap(apiutil.ErrRollbackTx, errRollback), err)
			}
		}
	}()
	user, err := svc.users.Save(ctx, u)
	if err != nil {
		return User{}, errors.Wrap(svcerr.ErrCreateEntity, err)
	}
	return user, nil
}

func (svc service) IssueToken(ctx context.Context, identity, secret string) (*grpcTokenV1.Token, error) {
	var dbUser User
	var err error

	if _, parseErr := mail.ParseAddress(identity); parseErr != nil {
		dbUser, err = svc.users.RetrieveByUsername(ctx, identity)
	} else {
		dbUser, err = svc.users.RetrieveByEmail(ctx, identity)
	}

	if err != nil {
		return &grpcTokenV1.Token{}, errors.Wrap(svcerr.ErrAuthentication, err)
	}

	if err := svc.hasher.Compare(secret, dbUser.Credentials.Secret); err != nil {
		return &grpcTokenV1.Token{}, errors.Wrap(svcerr.ErrLogin, err)
	}

	token, err := svc.token.Issue(ctx, &grpcTokenV1.IssueReq{UserId: dbUser.ID, UserRole: uint32(dbUser.Role + 1), Type: uint32(smqauth.AccessKey)})
	if err != nil {
		return &grpcTokenV1.Token{}, errors.Wrap(errIssueToken, err)
	}

	return token, nil
}

func (svc service) RefreshToken(ctx context.Context, session authn.Session, refreshToken string) (*grpcTokenV1.Token, error) {
	dbUser, err := svc.users.RetrieveByID(ctx, session.UserID)
	if err != nil {
		return &grpcTokenV1.Token{}, errors.Wrap(svcerr.ErrAuthentication, err)
	}
	if dbUser.Status == DisabledStatus {
		return &grpcTokenV1.Token{}, errors.Wrap(svcerr.ErrAuthentication, errLoginDisableUser)
	}

	return svc.token.Refresh(ctx, &grpcTokenV1.RefreshReq{RefreshToken: refreshToken})
}

func (svc service) View(ctx context.Context, session authn.Session, id string) (User, error) {
	user, err := svc.users.RetrieveByID(ctx, id)
	if err != nil {
		return User{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}

	if session.UserID != id {
		if err := svc.checkSuperAdmin(ctx, session); err != nil {
			return User{
				FirstName:   user.FirstName,
				LastName:    user.LastName,
				ID:          user.ID,
				Credentials: Credentials{Username: user.Credentials.Username},
			}, nil
		}
	}

	user.Credentials.Secret = ""

	return user, nil
}

func (svc service) ViewProfile(ctx context.Context, session authn.Session) (User, error) {
	user, err := svc.users.RetrieveByID(ctx, session.UserID)
	if err != nil {
		return User{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}
	user.Credentials.Secret = ""

	return user, nil
}

func (svc service) ListUsers(ctx context.Context, session authn.Session, pm Page) (UsersPage, error) {
	if err := svc.checkSuperAdmin(ctx, session); err != nil {
		return UsersPage{}, err
	}

	pm.Role = AllRole
	pg, err := svc.users.RetrieveAll(ctx, pm)
	if err != nil {
		return UsersPage{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}
	return pg, err
}

func (svc service) SearchUsers(ctx context.Context, pm Page) (UsersPage, error) {
	page := Page{
		Offset:    pm.Offset,
		Limit:     pm.Limit,
		FirstName: pm.FirstName,
		LastName:  pm.LastName,
		Username:  pm.Username,
		Id:        pm.Id,
		Role:      UserRole,
	}

	cp, err := svc.users.SearchUsers(ctx, page)
	if err != nil {
		return UsersPage{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}

	return cp, nil
}

func (svc service) Update(ctx context.Context, session authn.Session, id string, usr UserReq) (User, error) {
	if session.UserID != id {
		if err := svc.checkSuperAdmin(ctx, session); err != nil {
			return User{}, err
		}
	}
	updatedAt := time.Now().UTC()
	usr.UpdatedAt = &updatedAt
	usr.UpdatedBy = &session.UserID

	user, err := svc.users.Update(ctx, id, usr)
	if err != nil {
		return User{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}
	return user, nil
}

func (svc service) UpdateTags(ctx context.Context, session authn.Session, id string, usr UserReq) (User, error) {
	if session.UserID != id {
		if err := svc.checkSuperAdmin(ctx, session); err != nil {
			return User{}, err
		}
	}

	updatedAt := time.Now().UTC()
	usr.UpdatedAt = &updatedAt
	usr.UpdatedBy = &session.UserID

	user, err := svc.users.Update(ctx, id, usr)
	if err != nil {
		return User{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}

	return user, nil
}

func (svc service) UpdateProfilePicture(ctx context.Context, session authn.Session, id string, usr UserReq) (User, error) {
	if session.UserID != id {
		if err := svc.checkSuperAdmin(ctx, session); err != nil {
			return User{}, err
		}
	}

	updatedAt := time.Now().UTC()
	usr.UpdatedAt = &updatedAt
	usr.UpdatedBy = &session.UserID

	user, err := svc.users.Update(ctx, id, usr)
	if err != nil {
		return User{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}

	return user, nil
}

func (svc service) UpdateEmail(ctx context.Context, session authn.Session, userID, email string) (User, error) {
	if session.UserID != userID {
		if err := svc.checkSuperAdmin(ctx, session); err != nil {
			return User{}, err
		}
	}

	updatedAt := time.Now().UTC()
	usr := UserReq{
		Email:     &email,
		UpdatedAt: &updatedAt,
		UpdatedBy: &session.UserID,
	}
	user, err := svc.users.Update(ctx, userID, usr)
	if err != nil {
		return User{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}
	return user, nil
}

func (svc service) GenerateResetToken(ctx context.Context, email, host string) error {
	user, err := svc.users.RetrieveByEmail(ctx, email)
	if err != nil {
		return errors.Wrap(svcerr.ErrViewEntity, err)
	}
	issueReq := &grpcTokenV1.IssueReq{
		UserId: user.ID,
		Type:   uint32(smqauth.RecoveryKey),
	}
	token, err := svc.token.Issue(ctx, issueReq)
	if err != nil {
		return errors.Wrap(errRecoveryToken, err)
	}

	return svc.SendPasswordReset(ctx, host, email, user.Credentials.Username, token.AccessToken)
}

func (svc service) ResetSecret(ctx context.Context, session authn.Session, secret string) error {
	u, err := svc.users.RetrieveByID(ctx, session.UserID)
	if err != nil {
		return errors.Wrap(svcerr.ErrViewEntity, err)
	}

	secret, err = svc.hasher.Hash(secret)
	if err != nil {
		return errors.Wrap(svcerr.ErrMalformedEntity, err)
	}
	u = User{
		ID:    u.ID,
		Email: u.Email,
		Credentials: Credentials{
			Secret: secret,
		},
		UpdatedAt: time.Now().UTC(),
		UpdatedBy: session.UserID,
	}
	if _, err := svc.users.UpdateSecret(ctx, u); err != nil {
		return errors.Wrap(svcerr.ErrAuthorization, err)
	}
	return nil
}

func (svc service) UpdateSecret(ctx context.Context, session authn.Session, oldSecret, newSecret string) (User, error) {
	dbUser, err := svc.users.RetrieveByID(ctx, session.UserID)
	if err != nil {
		return User{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}
	if _, err := svc.IssueToken(ctx, dbUser.Credentials.Username, oldSecret); err != nil {
		return User{}, err
	}
	newSecret, err = svc.hasher.Hash(newSecret)
	if err != nil {
		return User{}, errors.Wrap(svcerr.ErrMalformedEntity, err)
	}
	dbUser.Credentials.Secret = newSecret
	dbUser.UpdatedAt = time.Now().UTC()
	dbUser.UpdatedBy = session.UserID

	dbUser, err = svc.users.UpdateSecret(ctx, dbUser)
	if err != nil {
		return User{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}

	return dbUser, nil
}

func (svc service) UpdateUsername(ctx context.Context, session authn.Session, id, username string) (User, error) {
	if session.UserID != id {
		if err := svc.checkSuperAdmin(ctx, session); err != nil {
			return User{}, err
		}
	}

	usr := User{
		ID: id,
		Credentials: Credentials{
			Username: username,
		},
		UpdatedAt: time.Now().UTC(),
		UpdatedBy: session.UserID,
	}
	updatedUser, err := svc.users.UpdateUsername(ctx, usr)
	if err != nil {
		return User{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}
	return updatedUser, nil
}

func (svc service) SendPasswordReset(_ context.Context, host, email, user, token string) error {
	to := []string{email}
	return svc.email.SendPasswordReset(to, host, user, token)
}

func (svc service) UpdateRole(ctx context.Context, session authn.Session, usr User) (User, error) {
	if err := svc.checkSuperAdmin(ctx, session); err != nil {
		return User{}, err
	}
	updateAt := time.Now().UTC()
	uReq := UserReq{
		Role:      &usr.Role,
		UpdatedAt: &updateAt,
		UpdatedBy: &session.UserID,
	}

	if err := svc.updateUserPolicy(ctx, usr.ID, usr.Role); err != nil {
		return User{}, err
	}

	u, err := svc.users.Update(ctx, usr.ID, uReq)
	if err != nil {
		// If failed to update role in DB, then revert back to platform admin policies in spicedb
		if errRollback := svc.updateUserPolicy(ctx, usr.ID, UserRole); errRollback != nil {
			return User{}, errors.Wrap(errRollback, err)
		}
		return User{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}
	return u, nil
}

func (svc service) Enable(ctx context.Context, session authn.Session, id string) (User, error) {
	u := User{
		ID:        id,
		UpdatedAt: time.Now().UTC(),
		Status:    EnabledStatus,
	}
	user, err := svc.changeUserStatus(ctx, session, u)
	if err != nil {
		return User{}, errors.Wrap(svcerr.ErrEnableUser, err)
	}

	return user, nil
}

func (svc service) Disable(ctx context.Context, session authn.Session, id string) (User, error) {
	user := User{
		ID:        id,
		UpdatedAt: time.Now().UTC(),
		Status:    DisabledStatus,
	}
	user, err := svc.changeUserStatus(ctx, session, user)
	if err != nil {
		return User{}, errors.Wrap(svcerr.ErrDisableUser, err)
	}

	return user, nil
}

func (svc service) changeUserStatus(ctx context.Context, session authn.Session, user User) (User, error) {
	if session.UserID != user.ID {
		if err := svc.checkSuperAdmin(ctx, session); err != nil {
			return User{}, err
		}
	}
	dbu, err := svc.users.RetrieveByID(ctx, user.ID)
	if err != nil {
		return User{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}
	if dbu.Status == user.Status {
		return User{}, errors.ErrStatusAlreadyAssigned
	}
	user.UpdatedBy = session.UserID

	user, err = svc.users.ChangeStatus(ctx, user)
	if err != nil {
		return User{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}
	return user, nil
}

func (svc service) Delete(ctx context.Context, session authn.Session, id string) error {
	user := User{
		ID:        id,
		UpdatedAt: time.Now().UTC(),
		Status:    DeletedStatus,
	}

	if _, err := svc.changeUserStatus(ctx, session, user); err != nil {
		return err
	}

	return nil
}

func (svc *service) checkSuperAdmin(ctx context.Context, session authn.Session) error {
	if !session.SuperAdmin {
		if err := svc.users.CheckSuperAdmin(ctx, session.UserID); err != nil {
			return errors.Wrap(svcerr.ErrAuthorization, err)
		}
	}

	return nil
}

func (svc service) OAuthCallback(ctx context.Context, user User) (User, error) {
	ruser, err := svc.users.RetrieveByEmail(ctx, user.Email)
	if err != nil {
		switch errors.Contains(err, repoerr.ErrNotFound) {
		case true:
			ruser, err = svc.Register(ctx, authn.Session{}, user, true)
			if err != nil {
				return User{}, err
			}
		default:
			return User{}, err
		}
	}

	return User{
		ID:   ruser.ID,
		Role: ruser.Role,
	}, nil
}

func (svc service) OAuthAddUserPolicy(ctx context.Context, user User) error {
	return svc.addUserPolicy(ctx, user.ID, user.Role)
}

func (svc service) Identify(ctx context.Context, session authn.Session) (string, error) {
	return session.UserID, nil
}

func (svc service) addUserPolicy(ctx context.Context, userID string, role Role) error {
	policyList := []policies.Policy{}

	policyList = append(policyList, policies.Policy{
		SubjectType: policies.UserType,
		Subject:     userID,
		Relation:    policies.MemberRelation,
		ObjectType:  policies.PlatformType,
		Object:      policies.SuperMQObject,
	})

	if role == AdminRole {
		policyList = append(policyList, policies.Policy{
			SubjectType: policies.UserType,
			Subject:     userID,
			Relation:    policies.AdministratorRelation,
			ObjectType:  policies.PlatformType,
			Object:      policies.SuperMQObject,
		})
	}
	err := svc.policies.AddPolicies(ctx, policyList)
	if err != nil {
		return errors.Wrap(svcerr.ErrAddPolicies, err)
	}

	return nil
}

func (svc service) addUserPolicyRollback(ctx context.Context, userID string, role Role) error {
	policyList := []policies.Policy{}

	policyList = append(policyList, policies.Policy{
		SubjectType: policies.UserType,
		Subject:     userID,
		Relation:    policies.MemberRelation,
		ObjectType:  policies.PlatformType,
		Object:      policies.SuperMQObject,
	})

	if role == AdminRole {
		policyList = append(policyList, policies.Policy{
			SubjectType: policies.UserType,
			Subject:     userID,
			Relation:    policies.AdministratorRelation,
			ObjectType:  policies.PlatformType,
			Object:      policies.SuperMQObject,
		})
	}
	err := svc.policies.DeletePolicies(ctx, policyList)
	if err != nil {
		return errors.Wrap(svcerr.ErrDeletePolicies, err)
	}

	return nil
}

func (svc service) updateUserPolicy(ctx context.Context, userID string, role Role) error {
	switch role {
	case AdminRole:
		err := svc.policies.AddPolicy(ctx, policies.Policy{
			SubjectType: policies.UserType,
			Subject:     userID,
			Relation:    policies.AdministratorRelation,
			ObjectType:  policies.PlatformType,
			Object:      policies.SuperMQObject,
		})
		if err != nil {
			return errors.Wrap(svcerr.ErrAddPolicies, err)
		}

		return nil
	case UserRole:
		fallthrough
	default:
		err := svc.policies.DeletePolicyFilter(ctx, policies.Policy{
			SubjectType: policies.UserType,
			Subject:     userID,
			Relation:    policies.AdministratorRelation,
			ObjectType:  policies.PlatformType,
			Object:      policies.SuperMQObject,
		})
		if err != nil {
			return errors.Wrap(svcerr.ErrDeletePolicies, err)
		}

		return nil
	}
}
