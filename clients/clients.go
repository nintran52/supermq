// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package clients

import (
	"context"
	"time"

	"github.com/absmach/supermq/pkg/authn"
	"github.com/absmach/supermq/pkg/connections"
	"github.com/absmach/supermq/pkg/postgres"
	"github.com/absmach/supermq/pkg/roles"
)

type Connection struct {
	ClientID  string
	ChannelID string
	DomainID  string
	Type      connections.ConnType
}

type ClientRepository struct {
	DB postgres.Database
}

// Repository is the interface that wraps the basic methods for
// a client repository.
type Repository interface {
	// RetrieveByID retrieves client by its unique ID.
	RetrieveByID(ctx context.Context, id string) (Client, error)

	// RetrieveByIDWithRoles retrieves client by its unique ID along with member roles.
	RetrieveByIDWithRoles(ctx context.Context, id, memberID string) (Client, error)

	// RetrieveAll retrieves all clients.
	RetrieveAll(ctx context.Context, pm Page) (ClientsPage, error)

	// RetrieveUserClients retrieve all clients of a given user id.
	RetrieveUserClients(ctx context.Context, domainID, userID string, pm Page) (ClientsPage, error)

	// SearchClients retrieves clients based on search criteria.
	SearchClients(ctx context.Context, pm Page) (ClientsPage, error)

	// RetrieveByIds
	RetrieveByIds(ctx context.Context, ids []string) (ClientsPage, error)

	// Update updates the client name and metadata.
	Update(ctx context.Context, client Client) (Client, error)

	// UpdateTags updates the client tags.
	UpdateTags(ctx context.Context, client Client) (Client, error)

	// UpdateIdentity updates identity for client with given id.
	UpdateIdentity(ctx context.Context, client Client) (Client, error)

	// UpdateSecret updates secret for client with given identity.
	UpdateSecret(ctx context.Context, client Client) (Client, error)

	// ChangeStatus changes client status to enabled or disabled
	ChangeStatus(ctx context.Context, client Client) (Client, error)

	// Delete deletes client with given id
	Delete(ctx context.Context, clientIDs ...string) error

	// Save persists the client account. A non-nil error is returned to indicate
	// operation failure.
	Save(ctx context.Context, client ...Client) ([]Client, error)

	// RetrieveBySecret retrieves a client based on the secret (key).
	RetrieveBySecret(ctx context.Context, key string) (Client, error)

	AddConnections(ctx context.Context, conns []Connection) error

	RemoveConnections(ctx context.Context, conns []Connection) error

	ClientConnectionsCount(ctx context.Context, id string) (uint64, error)

	DoesClientHaveConnections(ctx context.Context, id string) (bool, error)

	RemoveChannelConnections(ctx context.Context, channelID string) error

	RemoveClientConnections(ctx context.Context, clientID string) error

	// SetParentGroup set parent group id to a given channel id
	SetParentGroup(ctx context.Context, cli Client) error

	// RemoveParentGroup remove parent group id fr given chanel id
	RemoveParentGroup(ctx context.Context, cli Client) error

	RetrieveParentGroupClients(ctx context.Context, parentGroupID string) ([]Client, error)

	UnsetParentGroupFromClient(ctx context.Context, parentGroupID string) error

	roles.Repository
}

// Service specifies an API that must be fullfiled by the domain service
// implementation, and all of its decorators (e.g. logging & metrics).
type Service interface {
	// CreateClients creates new client. In case of the failed registration, a
	// non-nil error value is returned.
	CreateClients(ctx context.Context, session authn.Session, client ...Client) ([]Client, []roles.RoleProvision, error)

	// View retrieves client info for a given client ID and an authorized token.
	View(ctx context.Context, session authn.Session, id string, withRoles bool) (Client, error)

	// ListClients retrieves clients list for given page query.
	ListClients(ctx context.Context, session authn.Session, pm Page) (ClientsPage, error)

	// ListUserClients retrieves clients list for a given user id and page query.
	ListUserClients(ctx context.Context, session authn.Session, userID string, pm Page) (ClientsPage, error)

	// Update updates the client's name and metadata.
	Update(ctx context.Context, session authn.Session, client Client) (Client, error)

	// UpdateTags updates the client's tags.
	UpdateTags(ctx context.Context, session authn.Session, client Client) (Client, error)

	// UpdateSecret updates the client's secret
	UpdateSecret(ctx context.Context, session authn.Session, id, key string) (Client, error)

	// Enable logically enableds the client identified with the provided ID
	Enable(ctx context.Context, session authn.Session, id string) (Client, error)

	// Disable logically disables the client identified with the provided ID
	Disable(ctx context.Context, session authn.Session, id string) (Client, error)

	// Delete deletes client with given ID.
	Delete(ctx context.Context, session authn.Session, id string) error

	SetParentGroup(ctx context.Context, session authn.Session, parentGroupID string, id string) error

	RemoveParentGroup(ctx context.Context, session authn.Session, id string) error

	roles.RoleManager
}

// Cache contains client caching interface.
type Cache interface {
	// Save stores pair client secret, client id.
	Save(ctx context.Context, clientSecret, clientID string) error

	// ID returns client ID for given client secret.
	ID(ctx context.Context, clientSecret string) (string, error)

	// Removes client from cache.
	Remove(ctx context.Context, clientID string) error
}

// Client Struct represents a client.

type Client struct {
	ID          string      `json:"id"`
	Name        string      `json:"name,omitempty"`
	Tags        []string    `json:"tags,omitempty"`
	Domain      string      `json:"domain_id,omitempty"`
	ParentGroup string      `json:"parent_group_id,omitempty"`
	Credentials Credentials `json:"credentials,omitempty"`
	Metadata    Metadata    `json:"metadata,omitempty"`
	CreatedAt   time.Time   `json:"created_at,omitempty"`
	UpdatedAt   time.Time   `json:"updated_at,omitempty"`
	UpdatedBy   string      `json:"updated_by,omitempty"`
	Status      Status      `json:"status,omitempty"` // 1 for enabled, 0 for disabled
	Identity    string      `json:"identity,omitempty"`
	// Extended
	ParentGroupPath           string                    `json:"parent_group_path,omitempty"`
	RoleID                    string                    `json:"role_id,omitempty"`
	RoleName                  string                    `json:"role_name,omitempty"`
	Actions                   []string                  `json:"actions,omitempty"`
	AccessType                string                    `json:"access_type,omitempty"`
	AccessProviderId          string                    `json:"access_provider_id,omitempty"`
	AccessProviderRoleId      string                    `json:"access_provider_role_id,omitempty"`
	AccessProviderRoleName    string                    `json:"access_provider_role_name,omitempty"`
	AccessProviderRoleActions []string                  `json:"access_provider_role_actions,omitempty"`
	ConnectionTypes           []connections.ConnType    `json:"connection_types,omitempty"`
	MemberId                  string                    `json:"member_id,omitempty"`
	Roles                     []roles.MemberRoleActions `json:"roles,omitempty"`
}

// ClientsPage contains page related metadata as well as list.
type ClientsPage struct {
	Page
	Clients []Client
}

// MembersPage contains page related metadata as well as list of members that
// belong to this page.

type MembersPage struct {
	Page
	Members []Client
}

// Page contains the page metadata that helps navigation.

type Page struct {
	Total          uint64   `json:"total"`
	Offset         uint64   `json:"offset"`
	Limit          uint64   `json:"limit"`
	Order          string   `json:"order,omitempty"`
	Dir            string   `json:"dir,omitempty"`
	ID             string   `json:"id,omitempty"`
	Name           string   `json:"name,omitempty"`
	Metadata       Metadata `json:"metadata,omitempty"`
	Domain         string   `json:"domain,omitempty"`
	Tag            string   `json:"tag,omitempty"`
	Status         Status   `json:"status,omitempty"`
	Identity       string   `json:"identity,omitempty"`
	Group          *string  `json:"group,omitempty"`
	Channel        string   `json:"channel,omitempty"`
	ConnectionType string   `json:"connection_type,omitempty"`
	RoleName       string   `json:"role_name,omitempty"`
	RoleID         string   `json:"role_id,omitempty"`
	Actions        []string `json:"actions,omitempty"`
	AccessType     string   `json:"access_type,omitempty"`
	IDs            []string `json:"-"`
}

// Metadata represents arbitrary JSON.
type Metadata map[string]interface{}

// Credentials represent client credentials: its
// "identity" which can be a username, email, generated name;
// and "secret" which can be a password or access token.
type Credentials struct {
	Identity string `json:"identity,omitempty"` // username or generated login ID
	Secret   string `json:"secret,omitempty"`   // password or token
}
