// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package private

import (
	"context"

	"github.com/absmach/supermq/groups"
)

type Service interface {
	RetrieveById(ctx context.Context, id string) (groups.Group, error)
}

var _ Service = (*service)(nil)

func New(repo groups.Repository) Service {
	return service{repo}
}

type service struct {
	repo groups.Repository
}

func (svc service) RetrieveById(ctx context.Context, ids string) (groups.Group, error) {
	return svc.repo.RetrieveByID(ctx, ids)
}
