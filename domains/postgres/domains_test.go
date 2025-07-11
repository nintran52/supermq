// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package postgres_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/absmach/supermq/domains"
	"github.com/absmach/supermq/domains/postgres"
	"github.com/absmach/supermq/internal/testsutil"
	"github.com/absmach/supermq/pkg/errors"
	repoerr "github.com/absmach/supermq/pkg/errors/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const invalid = "invalid"

var (
	domainID = testsutil.GenerateUUID(&testing.T{})
	userID   = testsutil.GenerateUUID(&testing.T{})
)

func TestSaveDomain(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM domains")
		require.Nil(t, err, fmt.Sprintf("clean domains unexpected error: %s", err))
	})

	repo := postgres.NewRepository(database)

	cases := []struct {
		desc   string
		domain domains.Domain
		err    error
	}{
		{
			desc: "add new domain with all fields successfully",
			domain: domains.Domain{
				ID:    domainID,
				Name:  "test",
				Route: "test",
				Tags:  []string{"test"},
				Metadata: map[string]interface{}{
					"test": "test",
				},
				CreatedAt: time.Now().UTC().Truncate(time.Millisecond),
				UpdatedAt: time.Now().UTC().Truncate(time.Millisecond),
				CreatedBy: userID,
				UpdatedBy: userID,
				Status:    domains.EnabledStatus,
			},
			err: nil,
		},
		{
			desc: "add the same domain again",
			domain: domains.Domain{
				ID:    domainID,
				Name:  "test",
				Route: "test",
				Tags:  []string{"test"},
				Metadata: map[string]interface{}{
					"test": "test",
				},
				CreatedAt: time.Now().UTC().Truncate(time.Millisecond),
				UpdatedAt: time.Now().UTC().Truncate(time.Millisecond),
				CreatedBy: userID,
				UpdatedBy: userID,
				Status:    domains.EnabledStatus,
			},
			err: repoerr.ErrConflict,
		},
		{
			desc: "add domain with empty ID",
			domain: domains.Domain{
				ID:    "",
				Name:  "test1",
				Route: "test1",
				Tags:  []string{"test"},
				Metadata: map[string]interface{}{
					"test": "test",
				},
				CreatedAt: time.Now().UTC().Truncate(time.Millisecond),
				UpdatedAt: time.Now().UTC().Truncate(time.Millisecond),
				CreatedBy: userID,
				UpdatedBy: userID,
				Status:    domains.EnabledStatus,
			},
			err: nil,
		},
		{
			desc: "add domain with empty route",
			domain: domains.Domain{
				ID:    testsutil.GenerateUUID(&testing.T{}),
				Name:  "test1",
				Route: "",
				Tags:  []string{"test"},
				Metadata: map[string]interface{}{
					"test": "test",
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
				CreatedBy: userID,
				UpdatedBy: userID,
				Status:    domains.EnabledStatus,
			},
			err: repoerr.ErrCreateEntity,
		},
		{
			desc: "add domain with malformed metadata",
			domain: domains.Domain{
				ID:    domainID,
				Name:  "test1",
				Route: "test1",
				Tags:  []string{"test"},
				Metadata: map[string]interface{}{
					"key": make(chan int),
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
				CreatedBy: userID,
				UpdatedBy: userID,
				Status:    domains.EnabledStatus,
			},
			err: repoerr.ErrCreateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			domain, err := repo.SaveDomain(context.Background(), tc.domain)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			if err == nil {
				assert.Equal(t, tc.domain, domain, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.domain, domain))
			}
		})
	}
}

func TestRetrieveByID(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM domains")
		require.Nil(t, err, fmt.Sprintf("clean domains unexpected error: %s", err))
	})

	repo := postgres.NewRepository(database)

	domain := domains.Domain{
		ID:    domainID,
		Name:  "test",
		Route: "test",
		Tags:  []string{"test"},
		Metadata: map[string]interface{}{
			"test": "test",
		},
		CreatedBy: userID,
		UpdatedBy: userID,
		CreatedAt: time.Now().UTC().Truncate(time.Millisecond),
		UpdatedAt: time.Now().UTC().Truncate(time.Millisecond),
		Status:    domains.EnabledStatus,
	}

	_, err := repo.SaveDomain(context.Background(), domain)
	require.Nil(t, err, fmt.Sprintf("failed to save domain %s", domain.ID))

	cases := []struct {
		desc     string
		domainID string
		response domains.Domain
		err      error
	}{
		{
			desc:     "retrieve existing domain",
			domainID: domain.ID,
			response: domain,
			err:      nil,
		},
		{
			desc:     "retrieve non-existing domain",
			domainID: invalid,
			response: domains.Domain{},
			err:      repoerr.ErrNotFound,
		},
		{
			desc:     "retrieve with empty domain id",
			domainID: "",
			response: domains.Domain{},
			err:      repoerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			d, err := repo.RetrieveDomainByID(context.Background(), tc.domainID)
			assert.Equal(t, tc.response, d, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, d))
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.err, err))
		})
	}
}

func TestRetrieveByRoute(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM domains")
		require.Nil(t, err, fmt.Sprintf("clean domains unexpected error: %s", err))
	})

	repo := postgres.NewRepository(database)

	validRoute := "testRoute"
	domain := domains.Domain{
		ID:    domainID,
		Name:  "test",
		Route: validRoute,
		Tags:  []string{"test"},
		Metadata: map[string]interface{}{
			"test": "test",
		},
		CreatedBy: userID,
		UpdatedBy: userID,
		CreatedAt: time.Now().UTC().Truncate(time.Millisecond),
		UpdatedAt: time.Now().UTC().Truncate(time.Millisecond),
		Status:    domains.EnabledStatus,
	}

	_, err := repo.SaveDomain(context.Background(), domain)
	require.Nil(t, err, fmt.Sprintf("failed to save domain %s", domain.ID))

	cases := []struct {
		desc     string
		route    string
		response domains.Domain
		err      error
	}{
		{
			desc:     "retrieve existing domain",
			route:    validRoute,
			response: domain,
			err:      nil,
		},
		{
			desc:     "retrieve doamin with invalid route",
			route:    invalid,
			response: domains.Domain{},
			err:      repoerr.ErrNotFound,
		},
		{
			desc:     "retrieve with empty domain route",
			route:    "",
			response: domains.Domain{},
			err:      repoerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			d, err := repo.RetrieveDomainByRoute(context.Background(), tc.route)
			assert.Equal(t, tc.response, d, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, d))
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.err, err))
		})
	}
}

func TestRetrieveAllByIDs(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM domains")
		require.Nil(t, err, fmt.Sprintf("clean domains unexpected error: %s", err))
	})

	repo := postgres.NewRepository(database)

	items := []domains.Domain{}
	for i := 0; i < 10; i++ {
		domain := domains.Domain{
			ID:    testsutil.GenerateUUID(t),
			Name:  fmt.Sprintf(`"test%d"`, i),
			Route: fmt.Sprintf(`"test%d"`, i),
			Tags:  []string{"test"},
			Metadata: map[string]interface{}{
				"test": "test",
			},
			CreatedBy: userID,
			UpdatedBy: userID,
			Status:    domains.EnabledStatus,
		}
		if i%5 == 0 {
			domain.Status = domains.DisabledStatus
			domain.Tags = []string{"test", "admin"}
			domain.Metadata = map[string]interface{}{
				"test1": "test1",
			}
		}
		_, err := repo.SaveDomain(context.Background(), domain)
		require.Nil(t, err, fmt.Sprintf("save domain unexpected error: %s", err))
		items = append(items, domain)
	}

	cases := []struct {
		desc     string
		pm       domains.Page
		response domains.DomainsPage
		err      error
	}{
		{
			desc: "retrieve by ids successfully",
			pm: domains.Page{
				Offset: 0,
				Limit:  10,
				IDs:    []string{items[1].ID, items[2].ID},
			},
			response: domains.DomainsPage{
				Total:   2,
				Offset:  0,
				Limit:   10,
				Domains: []domains.Domain{items[1], items[2]},
			},
			err: nil,
		},
		{
			desc: "retrieve by ids with empty ids",
			pm: domains.Page{
				Offset: 0,
				Limit:  10,
				IDs:    []string{},
			},
			response: domains.DomainsPage{
				Total:  0,
				Offset: 0,
				Limit:  0,
			},
			err: nil,
		},
		{
			desc: "retrieve by ids with invalid ids",
			pm: domains.Page{
				Offset: 0,
				Limit:  10,
				IDs:    []string{invalid},
			},
			response: domains.DomainsPage{
				Total:  0,
				Offset: 0,
				Limit:  10,
			},
			err: nil,
		},
		{
			desc: "retrieve by ids and status",
			pm: domains.Page{
				Offset: 0,
				Limit:  10,
				IDs:    []string{items[0].ID, items[1].ID},
				Status: domains.DisabledStatus,
			},
			response: domains.DomainsPage{
				Total:   1,
				Offset:  0,
				Limit:   10,
				Domains: []domains.Domain{items[0]},
			},
		},
		{
			desc: "retrieve by ids and status with invalid status",
			pm: domains.Page{
				Offset: 0,
				Limit:  10,
				IDs:    []string{items[0].ID, items[1].ID},
				Status: 5,
			},
			response: domains.DomainsPage{
				Total:   2,
				Offset:  0,
				Limit:   10,
				Domains: []domains.Domain{items[0], items[1]},
			},
		},
		{
			desc: "retrieve by ids and tags",
			pm: domains.Page{
				Offset: 0,
				Limit:  10,
				IDs:    []string{items[0].ID, items[1].ID},
				Tag:    "test",
			},
			response: domains.DomainsPage{
				Total:   1,
				Offset:  0,
				Limit:   10,
				Domains: []domains.Domain{items[1]},
			},
		},
		{
			desc: "retrieve by ids and metadata",
			pm: domains.Page{
				Offset: 0,
				Limit:  10,
				IDs:    []string{items[1].ID, items[2].ID},
				Metadata: map[string]interface{}{
					"test": "test",
				},
				Status: domains.EnabledStatus,
			},
			response: domains.DomainsPage{
				Total:   2,
				Offset:  0,
				Limit:   10,
				Domains: items[1:3],
			},
		},
		{
			desc: "retrieve by ids and metadata with invalid metadata",
			pm: domains.Page{
				Offset: 0,
				Limit:  10,
				IDs:    []string{items[1].ID, items[2].ID},
				Metadata: map[string]interface{}{
					"test1": "test1",
				},
				Status: domains.EnabledStatus,
			},
			response: domains.DomainsPage{
				Total:  0,
				Offset: 0,
				Limit:  10,
			},
		},
		{
			desc: "retrieve by ids and malfomed metadata",
			pm: domains.Page{
				Offset: 0,
				Limit:  10,
				IDs:    []string{items[1].ID, items[2].ID},
				Metadata: map[string]interface{}{
					"key": make(chan int),
				},
				Status: domains.EnabledStatus,
			},
			response: domains.DomainsPage{},
			err:      repoerr.ErrViewEntity,
		},
		{
			desc: "retrieve all by ids and id",
			pm: domains.Page{
				Offset: 0,
				Limit:  10,
				ID:     items[1].ID,
				IDs:    []string{items[1].ID, items[2].ID},
			},
			response: domains.DomainsPage{
				Total:   1,
				Offset:  0,
				Limit:   10,
				Domains: []domains.Domain{items[1]},
			},
		},
		{
			desc: "retrieve all by ids and id with invalid id",
			pm: domains.Page{
				Offset: 0,
				Limit:  10,
				ID:     invalid,
				IDs:    []string{items[1].ID, items[2].ID},
			},
			response: domains.DomainsPage{
				Total:  0,
				Offset: 0,
				Limit:  10,
			},
		},
		{
			desc: "retrieve all by ids and name",
			pm: domains.Page{
				Offset: 0,
				Limit:  10,
				Name:   items[1].Name,
				IDs:    []string{items[1].ID, items[2].ID},
			},
			response: domains.DomainsPage{
				Total:   1,
				Offset:  0,
				Limit:   10,
				Domains: []domains.Domain{items[1]},
			},
		},
		{
			desc:     "retrieve all by ids with empty page",
			pm:       domains.Page{},
			response: domains.DomainsPage{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			dp, err := repo.RetrieveAllDomainsByIDs(context.Background(), tc.pm)
			assert.Equal(t, tc.response, dp, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, dp))
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.err, err))
		})
	}
}

func TestUpdate(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM domains")
		require.Nil(t, err, fmt.Sprintf("clean domains unexpected error: %s", err))
	})

	updatedName := "test1"
	updatedMetadata := domains.Metadata{
		"test1": "test1",
	}
	updatedTags := []string{"test1"}
	updatedStatus := domains.DisabledStatus

	repo := postgres.NewRepository(database)

	domain := domains.Domain{
		ID:    domainID,
		Name:  "test",
		Route: "test",
		Tags:  []string{"test"},
		Metadata: map[string]interface{}{
			"test": "test",
		},
		CreatedBy: userID,
		UpdatedBy: userID,
		Status:    domains.EnabledStatus,
	}

	_, err := repo.SaveDomain(context.Background(), domain)
	require.Nil(t, err, fmt.Sprintf("failed to save domain %s", domain.ID))

	cases := []struct {
		desc     string
		domainID string
		d        domains.DomainReq
		response domains.Domain
		err      error
	}{
		{
			desc:     "update existing domain name and metadata",
			domainID: domain.ID,
			d: domains.DomainReq{
				Name:     &updatedName,
				Metadata: &updatedMetadata,
			},
			response: domains.Domain{
				ID:    domainID,
				Name:  "test1",
				Route: "test",
				Tags:  []string{"test"},
				Metadata: map[string]interface{}{
					"test1": "test1",
				},
				CreatedBy: userID,
				UpdatedBy: userID,
				Status:    domains.EnabledStatus,
				UpdatedAt: time.Now(),
			},
			err: nil,
		},
		{
			desc:     "update existing domain name, metadata, tags and status",
			domainID: domain.ID,
			d: domains.DomainReq{
				Name:     &updatedName,
				Metadata: &updatedMetadata,
				Tags:     &updatedTags,
				Status:   &updatedStatus,
			},
			response: domains.Domain{
				ID:    domainID,
				Name:  "test1",
				Route: "test",
				Tags:  []string{"test1"},
				Metadata: map[string]interface{}{
					"test1": "test1",
				},
				CreatedBy: userID,
				UpdatedBy: userID,
				Status:    domains.DisabledStatus,
				UpdatedAt: time.Now(),
			},
			err: nil,
		},
		{
			desc:     "update non-existing domain",
			domainID: invalid,
			d: domains.DomainReq{
				Name:     &updatedName,
				Metadata: &updatedMetadata,
			},
			response: domains.Domain{},
			err:      repoerr.ErrNotFound,
		},
		{
			desc:     "update domain with empty ID",
			domainID: "",
			d: domains.DomainReq{
				Name:     &updatedName,
				Metadata: &updatedMetadata,
			},
			response: domains.Domain{},
			err:      repoerr.ErrNotFound,
		},
		{
			desc:     "update domain with malformed metadata",
			domainID: domainID,
			d: domains.DomainReq{
				Name:     &updatedName,
				Metadata: &domains.Metadata{"key": make(chan int)},
			},
			response: domains.Domain{},
			err:      repoerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			d, err := repo.UpdateDomain(context.Background(), tc.domainID, tc.d)
			d.UpdatedAt = tc.response.UpdatedAt
			assert.Equal(t, tc.response, d, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, d))
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		})
	}
}

func TestDelete(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM domains")
		require.Nil(t, err, fmt.Sprintf("clean domains unexpected error: %s", err))
	})

	repo := postgres.NewRepository(database)

	domain := domains.Domain{
		ID:    domainID,
		Name:  "test",
		Route: "test",
		Tags:  []string{"test"},
		Metadata: map[string]interface{}{
			"test": "test",
		},
		CreatedBy: userID,
		UpdatedBy: userID,
		Status:    domains.EnabledStatus,
	}

	_, err := repo.SaveDomain(context.Background(), domain)
	require.Nil(t, err, fmt.Sprintf("failed to save domain %s", domain.ID))

	cases := []struct {
		desc     string
		domainID string
		err      error
	}{
		{
			desc:     "delete existing domain",
			domainID: domain.ID,
			err:      nil,
		},
		{
			desc:     "delete non-existing domain",
			domainID: invalid,
			err:      repoerr.ErrNotFound,
		},
		{
			desc:     "delete domain with empty ID",
			domainID: "",
			err:      repoerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := repo.DeleteDomain(context.Background(), tc.domainID)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		})
	}
}

func TestListDomains(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM domains")
		require.Nil(t, err, fmt.Sprintf("clean domains unexpected error: %s", err))
	})

	repo := postgres.NewRepository(database)

	items := []domains.Domain{}
	for i := 0; i < 10; i++ {
		domain := domains.Domain{
			ID:    testsutil.GenerateUUID(t),
			Name:  fmt.Sprintf(`"test%d"`, i),
			Route: fmt.Sprintf(`"test%d"`, i),
			Tags:  []string{"test"},
			Metadata: map[string]interface{}{
				"test": "test",
			},
			CreatedBy: userID,
			UpdatedBy: userID,
			Status:    domains.EnabledStatus,
		}
		if i%5 == 0 {
			domain.Status = domains.DisabledStatus
			domain.Tags = []string{"test", "admin"}
			domain.Metadata = map[string]interface{}{
				"test1": "test1",
			}
		}
		_, err := repo.SaveDomain(context.Background(), domain)
		require.Nil(t, err, fmt.Sprintf("save domain unexpected error: %s", err))
		items = append(items, domain)
	}
	cases := []struct {
		desc     string
		pm       domains.Page
		response domains.DomainsPage
		err      error
	}{
		{
			desc: "list all domains",
			pm: domains.Page{
				Offset: 0,
				Limit:  10,
				Status: domains.AllStatus,
			},
			response: domains.DomainsPage{
				Total:   10,
				Offset:  0,
				Limit:   10,
				Domains: items,
			},
			err: nil,
		},
		{
			desc: "list all domains with enabled status",
			pm: domains.Page{
				Offset: 0,
				Limit:  10,
				Status: domains.EnabledStatus,
			},
			response: domains.DomainsPage{
				Total:   8,
				Offset:  0,
				Limit:   10,
				Domains: []domains.Domain{items[1], items[2], items[3], items[4], items[6], items[7], items[8], items[9]},
			},
			err: nil,
		},
		{
			desc: "list all domains with name",
			pm: domains.Page{
				Offset: 0,
				Limit:  10,
				Name:   items[0].Name,
				Status: domains.AllStatus,
			},
			response: domains.DomainsPage{
				Total:   1,
				Offset:  0,
				Limit:   10,
				Domains: []domains.Domain{items[0]},
			},
			err: nil,
		},
		{
			desc: "list all domains with disabled status",
			pm: domains.Page{
				Offset: 0,
				Limit:  10,
				Status: domains.DisabledStatus,
			},
			response: domains.DomainsPage{
				Total:   2,
				Offset:  0,
				Limit:   10,
				Domains: []domains.Domain{items[0], items[5]},
			},
			err: nil,
		},
		{
			desc: "list all domains with tags",
			pm: domains.Page{
				Offset: 0,
				Limit:  10,
				Tag:    "admin",
				Status: domains.AllStatus,
			},
			response: domains.DomainsPage{
				Total:   2,
				Offset:  0,
				Limit:   10,
				Domains: []domains.Domain{items[0], items[5]},
			},
			err: nil,
		},
		{
			desc: "list all domains with invalid tag",
			pm: domains.Page{
				Offset: 0,
				Limit:  10,
				Tag:    "invalid",
				Status: domains.AllStatus,
			},
			response: domains.DomainsPage{
				Total:   0,
				Offset:  0,
				Limit:   10,
				Domains: []domains.Domain(nil),
			},
			err: nil,
		},
		{
			desc: "list all domains with metadata",
			pm: domains.Page{
				Offset: 0,
				Limit:  10,
				Metadata: map[string]interface{}{
					"test1": "test1",
				},
				Status: domains.AllStatus,
			},
			response: domains.DomainsPage{
				Total:   2,
				Offset:  0,
				Limit:   10,
				Domains: []domains.Domain{items[0], items[5]},
			},
			err: nil,
		},
		{
			desc: "list all domains with invalid metadata",
			pm: domains.Page{
				Offset: 0,
				Limit:  10,
				Metadata: map[string]interface{}{
					"key": make(chan int),
				},
				Status: domains.AllStatus,
			},
			response: domains.DomainsPage{},
			err:      repoerr.ErrViewEntity,
		},
		{
			desc: "list all domains with subject id",
			pm: domains.Page{
				Offset: 0,
				Limit:  10,
				UserID: userID,
				Status: domains.AllStatus,
			},
			response: domains.DomainsPage{
				Total:  0,
				Offset: 0,
				Limit:  10,
			},
			err: nil,
		},
		{
			desc: "list domains with id",
			pm: domains.Page{
				Offset: 0,
				Limit:  10,
				ID:     items[0].ID,
				Status: domains.AllStatus,
			},
			response: domains.DomainsPage{
				Total:   1,
				Offset:  0,
				Limit:   10,
				Domains: []domains.Domain{items[0]},
			},
			err: nil,
		},
		{
			desc: "list domains with invalid id",
			pm: domains.Page{
				Offset: 0,
				Limit:  10,
				ID:     invalid,
				Status: domains.AllStatus,
			},
			response: domains.DomainsPage{
				Total:  0,
				Offset: 0,
				Limit:  10,
			},
			err: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			dp, err := repo.ListDomains(context.Background(), tc.pm)
			assert.Equal(t, tc.response, dp, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, dp))
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.err, err))
		})
	}
}
