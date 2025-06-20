// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package auth_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/absmach/supermq/auth"
	"github.com/absmach/supermq/auth/jwt"
	"github.com/absmach/supermq/auth/mocks"
	"github.com/absmach/supermq/internal/testsutil"
	"github.com/absmach/supermq/pkg/errors"
	repoerr "github.com/absmach/supermq/pkg/errors/repository"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	"github.com/absmach/supermq/pkg/policies"
	policymocks "github.com/absmach/supermq/pkg/policies/mocks"
	"github.com/absmach/supermq/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const (
	secret          = "secret"
	email           = "test@example.com"
	id              = "testID"
	groupName       = "smqx"
	description     = "Description"
	memberRelation  = "member"
	authoritiesObj  = "authorities"
	loginDuration   = 30 * time.Minute
	refreshDuration = 24 * time.Hour
	invalidDuration = 7 * 24 * time.Hour
	validID         = "d4ebb847-5d0e-4e46-bdd9-b6aceaaa3a22"
)

var (
	errIssueUser = errors.New("failed to issue new login key")
	errRoleAuth  = errors.New("failed to authorize user role")
	ErrExpiry    = errors.New("token is expired")
	inValidToken = "invalid"
	userID       = testsutil.GenerateUUID(&testing.T{})
	domainID     = testsutil.GenerateUUID(&testing.T{})
)

var (
	krepo      *mocks.KeyRepository
	pService   *policymocks.Service
	pEvaluator *policymocks.Evaluator
	patsrepo   *mocks.PATSRepository
	cache      *mocks.Cache
	hasher     *mocks.Hasher
)

func newService() (auth.Service, string) {
	krepo = new(mocks.KeyRepository)
	cache = new(mocks.Cache)
	pService = new(policymocks.Service)
	pEvaluator = new(policymocks.Evaluator)
	patsrepo = new(mocks.PATSRepository)
	hasher = new(mocks.Hasher)
	idProvider := uuid.NewMock()

	t := jwt.New([]byte(secret))
	key := auth.Key{
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(refreshDuration),
		Subject:   userID,
		Type:      auth.AccessKey,
		Role:      auth.UserRole,
	}
	token, _ := t.Issue(key)

	return auth.New(krepo, patsrepo, cache, hasher, idProvider, t, pEvaluator, pService, loginDuration, refreshDuration, invalidDuration), token
}

func TestIssue(t *testing.T) {
	svc, accessToken := newService()

	n := jwt.New([]byte(secret))

	apikey := auth.Key{
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(refreshDuration),
		Subject:   userID,
		Type:      auth.APIKey,
		Role:      auth.UserRole,
	}
	apiToken, err := n.Issue(apikey)
	assert.Nil(t, err, fmt.Sprintf("Issuing API key expected to succeed: %s", err))

	refreshkey := auth.Key{
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(refreshDuration),
		Subject:   userID,
		Type:      auth.RefreshKey,
		Role:      auth.UserRole,
	}
	refreshToken, err := n.Issue(refreshkey)
	assert.Nil(t, err, fmt.Sprintf("Issuing refresh key expected to succeed: %s", err))

	cases := []struct {
		desc         string
		key          auth.Key
		token        string
		roleCheckErr error
		err          error
	}{
		{
			desc: "issue recovery key",
			key: auth.Key{
				Type:     auth.RecoveryKey,
				Subject:  userID,
				Role:     auth.UserRole,
				IssuedAt: time.Now(),
			},
			token: "",
			err:   nil,
		},
	}

	for _, tc := range cases {
		policyCall := pEvaluator.On("CheckPolicy", mock.Anything, policies.Policy{
			Subject:     tc.key.Subject,
			SubjectType: policies.UserType,
			Permission:  policies.MembershipPermission,
			Object:      policies.SuperMQObject,
			ObjectType:  policies.PlatformType,
		}).Return(tc.roleCheckErr)
		_, err := svc.Issue(context.Background(), tc.token, tc.key)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
		policyCall.Unset()
	}

	cases2 := []struct {
		desc         string
		key          auth.Key
		saveResponse auth.Key
		token        string
		saveErr      error
		roleCheckErr error
		err          error
	}{
		{
			desc: "issue access key",
			key: auth.Key{
				Type:     auth.AccessKey,
				Subject:  userID,
				Role:     auth.UserRole,
				IssuedAt: time.Now(),
			},
			token: accessToken,
			err:   nil,
		},
	}
	for _, tc := range cases2 {
		repoCall := krepo.On("Save", mock.Anything, mock.Anything).Return(mock.Anything, tc.saveErr)
		policyCall := pEvaluator.On("CheckPolicy", mock.Anything, policies.Policy{
			Subject:     tc.key.Subject,
			SubjectType: policies.UserType,
			Permission:  policies.MembershipPermission,
			Object:      policies.SuperMQObject,
			ObjectType:  policies.PlatformType,
		}).Return(tc.roleCheckErr)
		_, err := svc.Issue(context.Background(), tc.token, tc.key)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Unset()
		policyCall.Unset()
	}

	cases3 := []struct {
		desc         string
		key          auth.Key
		token        string
		saveErr      error
		roleCheckErr error
		err          error
	}{
		{
			desc: "issue API key",
			key: auth.Key{
				Type:     auth.APIKey,
				Subject:  userID,
				Role:     auth.UserRole,
				IssuedAt: time.Now(),
			},
			token: accessToken,
			err:   nil,
		},
		{
			desc: "issue API key with an invalid token",
			key: auth.Key{
				Type:     auth.APIKey,
				Subject:  userID,
				Role:     auth.UserRole,
				IssuedAt: time.Now(),
			},
			token: "invalid",
			err:   svcerr.ErrAuthentication,
		},
		{
			desc: " issue API key with invalid key request",
			key: auth.Key{
				Type:     auth.APIKey,
				Subject:  "",
				Role:     auth.UserRole,
				IssuedAt: time.Now(),
			},
			token: apiToken,
			err:   svcerr.ErrAuthentication,
		},
		{
			desc: "issue API key with failed to save",
			key: auth.Key{
				Type:     auth.APIKey,
				Subject:  userID,
				Role:     auth.UserRole,
				IssuedAt: time.Now(),
			},
			token:   accessToken,
			saveErr: repoerr.ErrNotFound,
			err:     repoerr.ErrNotFound,
		},
		{
			desc: "issue API key with failed to check role",
			key: auth.Key{
				Type:     auth.APIKey,
				Subject:  userID,
				Role:     auth.UserRole,
				IssuedAt: time.Now(),
			},
			token:        accessToken,
			roleCheckErr: errRoleAuth,
			err:          errRoleAuth,
		},
	}
	for _, tc := range cases3 {
		repoCall := krepo.On("Save", mock.Anything, mock.Anything).Return(mock.Anything, tc.saveErr)
		policyCall := pEvaluator.On("CheckPolicy", mock.Anything, policies.Policy{
			Subject:     tc.key.Subject,
			SubjectType: policies.UserType,
			Permission:  policies.MembershipPermission,
			Object:      policies.SuperMQObject,
			ObjectType:  policies.PlatformType,
		}).Return(tc.roleCheckErr)
		_, err := svc.Issue(context.Background(), tc.token, tc.key)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Unset()
		policyCall.Unset()
	}

	cases4 := []struct {
		desc         string
		key          auth.Key
		token        string
		roleCheckErr error
		err          error
	}{
		{
			desc: "issue refresh key",
			key: auth.Key{
				Type:     auth.RefreshKey,
				IssuedAt: time.Now(),
				Subject:  userID,
				Role:     auth.UserRole,
			},
			token: refreshToken,
			err:   nil,
		},
		{
			desc: "issue refresh key with invalid token",
			key: auth.Key{
				Type:     auth.RefreshKey,
				IssuedAt: time.Now(),
				Subject:  userID,
				Role:     auth.UserRole,
			},
			token: inValidToken,
			err:   svcerr.ErrAuthentication,
		},
		{
			desc: "issue refresh key with empty token",
			key: auth.Key{
				Type:     auth.RefreshKey,
				IssuedAt: time.Now(),
				Subject:  userID,
				Role:     auth.UserRole,
			},
			token: "",
			err:   svcerr.ErrAuthentication,
		},
		{
			desc: "issue refresh key with failed to check role",
			key: auth.Key{
				Type:     auth.RefreshKey,
				IssuedAt: time.Now(),
				Subject:  userID,
				Role:     auth.UserRole,
			},
			token:        refreshToken,
			roleCheckErr: errRoleAuth,
			err:          errRoleAuth,
		},
	}
	for _, tc := range cases4 {
		policyCall := pEvaluator.On("CheckPolicy", mock.Anything, policies.Policy{
			Subject:     tc.key.Subject,
			SubjectType: policies.UserType,
			Permission:  policies.MembershipPermission,
			Object:      policies.SuperMQObject,
			ObjectType:  policies.PlatformType,
		}).Return(tc.roleCheckErr)
		_, err := svc.Issue(context.Background(), tc.token, tc.key)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
		policyCall.Unset()
	}
}

func TestRevoke(t *testing.T) {
	svc, _ := newService()
	repocall := krepo.On("Save", mock.Anything, mock.Anything).Return(mock.Anything, errIssueUser)
	policyCall := pEvaluator.On("CheckPolicy", mock.Anything, mock.Anything).Return(nil)
	secret, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.AccessKey, Role: auth.UserRole, IssuedAt: time.Now(), Subject: userID})
	repocall.Unset()
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))
	repocall1 := krepo.On("Save", mock.Anything, mock.Anything).Return(mock.Anything, nil)
	key := auth.Key{
		Type:     auth.APIKey,
		Role:     auth.UserRole,
		IssuedAt: time.Now(),
		Subject:  userID,
	}
	_, err = svc.Issue(context.Background(), secret.AccessToken, key)
	assert.Nil(t, err, fmt.Sprintf("Issuing user's key expected to succeed: %s", err))
	repocall1.Unset()
	policyCall.Unset()

	cases := []struct {
		desc  string
		id    string
		token string
		err   error
	}{
		{
			desc:  "revoke login key",
			token: secret.AccessToken,
			err:   nil,
		},
		{
			desc:  "revoke non-existing login key",
			token: secret.AccessToken,
			err:   nil,
		},
		{
			desc:  "revoke with empty login key",
			token: "",
			err:   svcerr.ErrAuthentication,
		},
		{
			desc:  "revoke login key with failed to remove",
			id:    "invalidID",
			token: secret.AccessToken,
			err:   svcerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		repocall := krepo.On("Remove", mock.Anything, mock.Anything, mock.Anything).Return(tc.err)
		err := svc.Revoke(context.Background(), tc.token, tc.id)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
		repocall.Unset()
	}
}

func TestRetrieve(t *testing.T) {
	svc, _ := newService()
	repocall := krepo.On("Save", mock.Anything, mock.Anything).Return(mock.Anything, nil)
	repocall1 := pEvaluator.On("CheckPolicy", mock.Anything, mock.Anything).Return(nil)
	secret, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.AccessKey, Subject: userID, Role: auth.UserRole, IssuedAt: time.Now()})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))
	repocall.Unset()
	key := auth.Key{
		ID:       "id",
		Type:     auth.APIKey,
		Subject:  userID,
		Role:     auth.UserRole,
		IssuedAt: time.Now(),
	}

	repocall3 := krepo.On("Save", mock.Anything, mock.Anything).Return(mock.Anything, nil)
	userToken, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.AccessKey, Subject: userID, Role: auth.UserRole, IssuedAt: time.Now()})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))
	repocall3.Unset()

	repocall4 := krepo.On("Save", mock.Anything, mock.Anything).Return(mock.Anything, nil)
	apiToken, err := svc.Issue(context.Background(), secret.AccessToken, key)
	assert.Nil(t, err, fmt.Sprintf("Issuing login's key expected to succeed: %s", err))
	repocall4.Unset()

	repocall5 := krepo.On("Save", mock.Anything, mock.Anything).Return(mock.Anything, nil)
	resetToken, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.RecoveryKey, Subject: userID, Role: auth.UserRole, IssuedAt: time.Now()})
	assert.Nil(t, err, fmt.Sprintf("Issuing reset key expected to succeed: %s", err))
	repocall5.Unset()
	repocall1.Unset()

	cases := []struct {
		desc  string
		id    string
		token string
		err   error
	}{
		{
			desc:  "retrieve login key",
			token: userToken.AccessToken,
			err:   nil,
		},
		{
			desc:  "retrieve non-existing login key",
			id:    "invalid",
			token: userToken.AccessToken,
			err:   svcerr.ErrNotFound,
		},
		{
			desc:  "retrieve with wrong login key",
			token: "wrong",
			err:   svcerr.ErrAuthentication,
		},
		{
			desc:  "retrieve with API token",
			token: apiToken.AccessToken,
			err:   svcerr.ErrAuthentication,
		},
		{
			desc:  "retrieve with reset token",
			token: resetToken.AccessToken,
			err:   svcerr.ErrAuthentication,
		},
	}

	for _, tc := range cases {
		repocall := krepo.On("Retrieve", mock.Anything, mock.Anything, mock.Anything).Return(auth.Key{}, tc.err)
		_, err := svc.RetrieveKey(context.Background(), tc.token, tc.id)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
		repocall.Unset()
	}
}

func TestIdentify(t *testing.T) {
	svc, _ := newService()

	repocall := krepo.On("Save", mock.Anything, mock.Anything).Return(mock.Anything, nil)
	repocall1 := pEvaluator.On("CheckPolicy", mock.Anything, mock.Anything).Return(nil)
	loginSecret, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.AccessKey, Subject: userID, Role: auth.UserRole, IssuedAt: time.Now()})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	recoverySecret, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.RecoveryKey, Role: auth.UserRole, IssuedAt: time.Now(), Subject: userID})
	assert.Nil(t, err, fmt.Sprintf("Issuing reset key expected to succeed: %s", err))

	apiSecret, err := svc.Issue(context.Background(), loginSecret.AccessToken, auth.Key{Type: auth.APIKey, Role: auth.UserRole, Subject: userID, IssuedAt: time.Now(), ExpiresAt: time.Now().Add(time.Minute)})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	exp0 := time.Now().UTC().Add(-10 * time.Second).Round(time.Second)
	exp1 := time.Now().UTC().Add(-1 * time.Minute).Round(time.Second)
	expSecret, err := svc.Issue(context.Background(), loginSecret.AccessToken, auth.Key{Type: auth.APIKey, Role: auth.UserRole, IssuedAt: exp0, ExpiresAt: exp1})
	assert.Nil(t, err, fmt.Sprintf("Issuing expired login key expected to succeed: %s", err))
	repocall.Unset()
	repocall1.Unset()

	te := jwt.New([]byte(secret))
	key := auth.Key{
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(refreshDuration),
		Type:      7,
		Subject:   userID,
		Role:      auth.UserRole,
	}
	invalidTokenType, _ := te.Issue(key)

	cases := []struct {
		desc    string
		key     string
		subject string
		err     error
	}{
		{
			desc:    "identify login key",
			key:     loginSecret.AccessToken,
			subject: userID,
			err:     nil,
		},
		{
			desc:    "identify refresh key",
			key:     loginSecret.RefreshToken,
			subject: userID,
			err:     nil,
		},
		{
			desc:    "identify recovery key",
			key:     recoverySecret.AccessToken,
			subject: userID,
			err:     nil,
		},
		{
			desc:    "identify API key",
			key:     apiSecret.AccessToken,
			subject: userID,
			err:     nil,
		},
		{
			desc:    "identify expired API key",
			key:     expSecret.AccessToken,
			subject: "",
			err:     auth.ErrKeyExpired,
		},
		{
			desc:    "identify API key with failed to retrieve",
			key:     apiSecret.AccessToken,
			subject: "",
			err:     svcerr.ErrAuthentication,
		},
		{
			desc:    "identify invalid key",
			key:     "invalid",
			subject: "",
			err:     svcerr.ErrAuthentication,
		},
		{
			desc:    "identify invalid key type",
			key:     invalidTokenType,
			subject: "",
			err:     svcerr.ErrAuthentication,
		},
	}

	for _, tc := range cases {
		repocall := krepo.On("Retrieve", mock.Anything, mock.Anything, mock.Anything).Return(auth.Key{}, tc.err)
		repocall1 := krepo.On("Remove", mock.Anything, mock.Anything, mock.Anything).Return(tc.err)
		idt, err := svc.Identify(context.Background(), tc.key)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.subject, idt.Subject, fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.subject, idt))
		repocall.Unset()
		repocall1.Unset()
	}
}

func TestAuthorize(t *testing.T) {
	svc, accessToken := newService()

	repoCall := krepo.On("Save", mock.Anything, mock.Anything).Return(mock.Anything, nil)
	repoCall1 := pEvaluator.On("CheckPolicy", mock.Anything, mock.Anything).Return(nil)
	loginSecret, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.AccessKey, Subject: userID, Role: auth.UserRole, IssuedAt: time.Now()})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))
	repoCall.Unset()

	repoCall = krepo.On("Save", mock.Anything, mock.Anything).Return(mock.Anything, nil)
	exp1 := time.Now().Add(-2 * time.Second)
	expSecret, err := svc.Issue(context.Background(), loginSecret.AccessToken, auth.Key{Type: auth.APIKey, Role: auth.UserRole, IssuedAt: time.Now(), ExpiresAt: exp1})
	assert.Nil(t, err, fmt.Sprintf("Issuing expired login key expected to succeed: %s", err))
	repoCall.Unset()

	repoCall = krepo.On("Save", mock.Anything, mock.Anything).Return(mock.Anything, nil)
	repoCall1 = pEvaluator.On("CheckPolicy", mock.Anything, mock.Anything).Return(nil)
	emptySubject, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.AccessKey, Subject: "", Role: auth.UserRole, IssuedAt: time.Now()})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))
	repoCall.Unset()
	repoCall1.Unset()

	cases := []struct {
		desc                 string
		policyReq            policies.Policy
		checkDomainPolicyReq policies.Policy
		checkPolicyReq       policies.Policy
		callBackErr          error
		checkPolicyErr       error
		checkDomainPolicyErr error
		err                  error
	}{
		{
			desc: "authorize token successfully",
			policyReq: policies.Policy{
				Subject:     accessToken,
				SubjectType: policies.UserType,
				SubjectKind: policies.TokenKind,
				Object:      policies.SuperMQObject,
				ObjectType:  policies.PlatformType,
				Permission:  policies.AdminPermission,
			},
			checkPolicyReq: policies.Policy{
				Subject:     userID,
				SubjectType: policies.UserType,
				SubjectKind: policies.TokenKind,
				Object:      policies.SuperMQObject,
				ObjectType:  policies.PlatformType,
				Permission:  policies.AdminPermission,
			},
			checkDomainPolicyReq: policies.Policy{
				Subject:     userID,
				SubjectType: policies.UserType,
				ObjectType:  policies.DomainType,
				Permission:  policies.MembershipPermission,
			},
			err: nil,
		},
		{
			desc: "authorize with malformed policy request",
			policyReq: policies.Policy{
				Subject:     accessToken,
				SubjectType: policies.UserType,
				SubjectKind: policies.TokenKind,
				Object:      domainID,
				ObjectType:  policies.PlatformType,
				Permission:  policies.AdminPermission,
			},
			checkPolicyReq: policies.Policy{},
			checkDomainPolicyReq: policies.Policy{
				Subject:     userID,
				SubjectType: policies.UserType,
				ObjectType:  policies.DomainType,
				Permission:  policies.MembershipPermission,
			},
			err: svcerr.ErrMalformedEntity,
		},
		{
			desc: "authorize token with disabled domain",
			policyReq: policies.Policy{
				Subject:     accessToken,
				SubjectType: policies.UserType,
				SubjectKind: policies.TokenKind,
				Object:      validID,
				ObjectType:  policies.DomainType,
				Permission:  policies.AdminPermission,
			},
			checkDomainPolicyReq: policies.Policy{
				Subject:     userID,
				SubjectType: policies.UserType,
				Object:      validID,
				ObjectType:  policies.DomainType,
				Permission:  policies.MembershipPermission,
			},
			checkDomainPolicyErr: svcerr.ErrAuthorization,
			err:                  svcerr.ErrDomainAuthorization,
		},
		{
			desc: "authorize an expired token",
			policyReq: policies.Policy{
				Subject:     expSecret.AccessToken,
				SubjectType: policies.UserType,
				SubjectKind: policies.TokenKind,
				Object:      policies.SuperMQObject,
				ObjectType:  policies.PlatformType,
				Permission:  policies.AdminPermission,
			},
			checkPolicyReq: policies.Policy{},
			checkDomainPolicyReq: policies.Policy{
				Subject:     userID,
				SubjectType: policies.UserType,
				Object:      validID,
				ObjectType:  policies.DomainType,
				Permission:  policies.MembershipPermission,
			},
			err: svcerr.ErrAuthentication,
		},
		{
			desc: "authorize a token with an empty subject",
			policyReq: policies.Policy{
				Subject:     emptySubject.AccessToken,
				SubjectType: policies.UserType,
				SubjectKind: policies.TokenKind,
				Object:      validID,
				ObjectType:  policies.DomainType,
				Permission:  policies.AdminPermission,
			},
			checkPolicyReq: policies.Policy{},
			checkDomainPolicyReq: policies.Policy{
				Subject:     userID,
				SubjectType: policies.UserType,
				Object:      validID,
				ObjectType:  policies.DomainType,
				Permission:  policies.MembershipPermission,
			},
			err: svcerr.ErrDomainAuthorization,
		},
		{
			desc: "authorize a token with an empty subject and invalid type",
			policyReq: policies.Policy{
				Subject:     emptySubject.AccessToken,
				SubjectType: policies.UserType,
				SubjectKind: policies.TokenKind,
				Object:      policies.SuperMQObject,
				ObjectType:  policies.DomainType,
				Permission:  policies.AdminPermission,
			},
			checkPolicyReq: policies.Policy{
				SubjectType: policies.UserType,
				Object:      policies.SuperMQObject,
				ObjectType:  policies.PlatformKind,
				Permission:  policies.AdminPermission,
			},
			checkDomainPolicyReq: policies.Policy{
				Subject:     userID,
				SubjectType: policies.UserType,
				Object:      validID,
				ObjectType:  policies.DomainType,
				Permission:  policies.MembershipPermission,
			},
			err: svcerr.ErrDomainAuthorization,
		},
		{
			desc: "authorize a token with an empty subject and invalid object type",
			policyReq: policies.Policy{
				Subject:     emptySubject.AccessToken,
				SubjectType: policies.UserType,
				SubjectKind: policies.TokenKind,
				Object:      validID,
				ObjectType:  policies.UserType,
				Permission:  policies.AdminPermission,
			},
			checkPolicyReq: policies.Policy{
				SubjectType: policies.UserType,
				Object:      policies.SuperMQObject,
				ObjectType:  policies.PlatformType,
				Permission:  policies.AdminPermission,
			},
			checkDomainPolicyReq: policies.Policy{
				Subject:     userID,
				SubjectType: policies.UserType,
				Object:      validID,
				ObjectType:  policies.DomainType,
				Permission:  policies.MembershipPermission,
			},
			err: svcerr.ErrAuthentication,
		},
		{
			desc: "authorize a user key successfully",
			policyReq: policies.Policy{
				SubjectType: policies.UserType,
				SubjectKind: policies.UsersKind,
				Object:      policies.SuperMQObject,
				ObjectType:  policies.PlatformType,
				Permission:  policies.AdminPermission,
			},
			checkPolicyReq: policies.Policy{
				SubjectType: policies.UserType,
				SubjectKind: policies.UsersKind,
				Object:      policies.SuperMQObject,
				ObjectType:  policies.PlatformType,
				Permission:  policies.AdminPermission,
			},
			checkDomainPolicyReq: policies.Policy{
				Subject:     userID,
				SubjectType: policies.UserType,
				Object:      validID,
				ObjectType:  policies.DomainType,
				Permission:  policies.MembershipPermission,
			},
			err: nil,
		},
		{
			desc: "authorize token with empty subject and domain object type",
			policyReq: policies.Policy{
				Subject:     emptySubject.AccessToken,
				SubjectType: policies.UserType,
				SubjectKind: policies.TokenKind,
				Object:      policies.SuperMQObject,
				ObjectType:  policies.DomainType,
				Permission:  policies.AdminPermission,
			},
			checkPolicyReq: policies.Policy{
				SubjectType: policies.UserType,
				Object:      policies.SuperMQObject,
				ObjectType:  policies.PlatformType,
				Permission:  policies.AdminPermission,
			},
			checkDomainPolicyReq: policies.Policy{
				Subject:     userID,
				SubjectType: policies.UserType,
				Object:      validID,
				ObjectType:  policies.DomainType,
				Permission:  policies.MembershipPermission,
			},
			err: svcerr.ErrDomainAuthorization,
		},
	}
	for _, tc := range cases {
		policyCall := pEvaluator.On("CheckPolicy", mock.Anything, tc.checkPolicyReq).Return(tc.checkPolicyErr)
		policyCall1 := pEvaluator.On("CheckPolicy", mock.Anything, tc.checkDomainPolicyReq).Return(tc.checkDomainPolicyErr)
		repoCall := krepo.On("Remove", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		err := svc.Authorize(context.Background(), tc.policyReq)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
		policyCall.Unset()
		policyCall1.Unset()
		repoCall.Unset()
	}
}

func TestSwitchToPermission(t *testing.T) {
	cases := []struct {
		desc     string
		relation string
		result   string
	}{
		{
			desc:     "switch to admin permission",
			relation: policies.AdministratorRelation,
			result:   policies.AdminPermission,
		},
		{
			desc:     "switch to editor permission",
			relation: policies.EditorRelation,
			result:   policies.EditPermission,
		},
		{
			desc:     "switch to contributor permission",
			relation: policies.ContributorRelation,
			result:   policies.ViewPermission,
		},
		{
			desc:     "switch to member permission",
			relation: policies.MemberRelation,
			result:   policies.MembershipPermission,
		},
		{
			desc:     "switch to group permission",
			relation: policies.GroupRelation,
			result:   policies.GroupRelation,
		},
		{
			desc:     "switch to guest permission",
			relation: policies.GuestRelation,
			result:   policies.ViewPermission,
		},
	}
	for _, tc := range cases {
		result := auth.SwitchToPermission(tc.relation)
		assert.Equal(t, tc.result, result, fmt.Sprintf("switching to permission expected to succeed: %s", result))
	}
}

func TestEncodeDomainUserID(t *testing.T) {
	cases := []struct {
		desc     string
		domainID string
		userID   string
		response string
	}{
		{
			desc:     "encode domain user id successfully",
			domainID: validID,
			userID:   validID,
			response: validID + "_" + validID,
		},
		{
			desc:     "encode domain user id with empty userID",
			domainID: validID,
			userID:   "",
			response: "",
		},
		{
			desc:     "encode domain user id with empty domain ID",
			domainID: "",
			userID:   validID,
			response: "",
		},
		{
			desc:     "encode domain user id with empty domain ID and userID",
			domainID: "",
			userID:   "",
			response: "",
		},
	}

	for _, tc := range cases {
		ar := auth.EncodeDomainUserID(tc.domainID, tc.userID)
		assert.Equal(t, tc.response, ar, fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.response, ar))
	}
}

func TestDecodeDomainUserID(t *testing.T) {
	cases := []struct {
		desc         string
		domainUserID string
		respDomainID string
		respUserID   string
	}{
		{
			desc:         "decode domain user id successfully",
			domainUserID: validID + "_" + validID,
			respDomainID: validID,
			respUserID:   validID,
		},
		{
			desc:         "decode domain user id with empty domainUserID",
			domainUserID: "",
			respDomainID: "",
			respUserID:   "",
		},
		{
			desc:         "decode domain user id with empty UserID",
			domainUserID: validID,
			respDomainID: validID,
			respUserID:   "",
		},
		{
			desc:         "decode domain user id with invalid domainuserId",
			domainUserID: validID + "_" + validID + "_" + validID + "_" + validID,
			respDomainID: "",
			respUserID:   "",
		},
	}

	for _, tc := range cases {
		ar, er := auth.DecodeDomainUserID(tc.domainUserID)
		assert.Equal(t, tc.respUserID, er, fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.respUserID, er))
		assert.Equal(t, tc.respDomainID, ar, fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.respDomainID, ar))
	}
}
