// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/absmach/supermq/journal"
	"github.com/absmach/supermq/pkg/errors"
	repoerr "github.com/absmach/supermq/pkg/errors/repository"
	"github.com/absmach/supermq/pkg/postgres"
)

type repository struct {
	db postgres.Database
}

func NewRepository(db postgres.Database) journal.Repository {
	return &repository{db: db}
}

func (repo *repository) Save(ctx context.Context, j journal.Journal) (err error) {
	domain, ok := j.Attributes["domain"].(string)
	if ok {
		j.Domain = domain
	}
	if strings.HasPrefix(j.Operation, "domain.") {
		domain, ok := j.Attributes["id"].(string)
		if ok {
			j.Domain = domain
		}
	}

	q := `INSERT INTO journal (id, operation, occurred_at, attributes, metadata, domain)
		VALUES (:id, :operation, :occurred_at, :attributes, :metadata, :domain);`

	dbJournal, err := toDBJournal(j)
	if err != nil {
		return errors.Wrap(repoerr.ErrCreateEntity, err)
	}

	if _, err = repo.db.NamedExecContext(ctx, q, dbJournal); err != nil {
		return postgres.HandleError(repoerr.ErrCreateEntity, err)
	}

	return nil
}

func (repo *repository) RetrieveAll(ctx context.Context, page journal.Page) (journal.JournalsPage, error) {
	query := pageQuery(page)

	sq := "operation, occurred_at, domain"
	if page.WithAttributes {
		sq += ", attributes"
	}
	if page.WithMetadata {
		sq += ", metadata"
	}
	if page.Direction == "" {
		page.Direction = "ASC"
	}
	q := fmt.Sprintf("SELECT %s FROM journal %s ORDER BY occurred_at %s LIMIT :limit OFFSET :offset;", sq, query, page.Direction)

	rows, err := repo.db.NamedQueryContext(ctx, q, page)
	if err != nil {
		return journal.JournalsPage{}, postgres.HandleError(repoerr.ErrViewEntity, err)
	}
	defer rows.Close()

	var items []journal.Journal
	for rows.Next() {
		var item dbJournal
		if err = rows.StructScan(&item); err != nil {
			return journal.JournalsPage{}, postgres.HandleError(repoerr.ErrViewEntity, err)
		}
		j, err := toJournal(item)
		if err != nil {
			return journal.JournalsPage{}, err
		}
		items = append(items, j)
	}

	tq := fmt.Sprintf(`SELECT COUNT(*) FROM journal %s;`, query)

	total, err := postgres.Total(ctx, repo.db, tq, page)
	if err != nil {
		return journal.JournalsPage{}, postgres.HandleError(repoerr.ErrViewEntity, err)
	}

	journalsPage := journal.JournalsPage{
		Total:    total,
		Offset:   page.Offset,
		Limit:    page.Limit,
		Journals: items,
	}

	return journalsPage, nil
}

func pageQuery(pm journal.Page) string {
	var query []string
	var emq string
	if pm.Operation != "" {
		query = append(query, "operation = :operation")
	}
	if !pm.From.IsZero() {
		query = append(query, "occurred_at >= :from")
	}
	if !pm.To.IsZero() {
		query = append(query, "occurred_at <= :to")
	}
	if pm.EntityID != "" {
		query = append(query, pm.EntityType.Query())
	}

	if len(query) > 0 {
		emq = fmt.Sprintf("WHERE %s", strings.Join(query, " AND "))
	}

	return emq
}

type dbJournal struct {
	ID         string    `db:"id"`
	Operation  string    `db:"operation"`
	Domain     string    `db:"domain"`
	OccurredAt time.Time `db:"occurred_at"`
	Attributes []byte    `db:"attributes"`
	Metadata   []byte    `db:"metadata"`
}

func toDBJournal(j journal.Journal) (dbJournal, error) {
	if j.OccurredAt.IsZero() {
		j.OccurredAt = time.Now().UTC()
	}

	attributes := []byte("{}")
	if len(j.Attributes) > 0 {
		b, err := json.Marshal(j.Attributes)
		if err != nil {
			return dbJournal{}, errors.Wrap(repoerr.ErrMalformedEntity, err)
		}
		attributes = b
	}

	metadata := []byte("{}")
	if len(j.Metadata) > 0 {
		b, err := json.Marshal(j.Metadata)
		if err != nil {
			return dbJournal{}, errors.Wrap(repoerr.ErrMalformedEntity, err)
		}
		metadata = b
	}

	return dbJournal{
		ID:         j.ID,
		Operation:  j.Operation,
		Domain:     j.Domain,
		OccurredAt: j.OccurredAt,
		Attributes: attributes,
		Metadata:   metadata,
	}, nil
}

func toJournal(dbj dbJournal) (journal.Journal, error) {
	var attributes map[string]interface{}
	if dbj.Attributes != nil {
		if err := json.Unmarshal(dbj.Attributes, &attributes); err != nil {
			return journal.Journal{}, errors.Wrap(repoerr.ErrMalformedEntity, err)
		}
	}

	var metadata map[string]interface{}
	if dbj.Metadata != nil {
		if err := json.Unmarshal(dbj.Metadata, &metadata); err != nil {
			return journal.Journal{}, errors.Wrap(repoerr.ErrMalformedEntity, err)
		}
	}

	return journal.Journal{
		Operation:  dbj.Operation,
		Domain:     dbj.Domain,
		OccurredAt: dbj.OccurredAt,
		Attributes: attributes,
		Metadata:   metadata,
	}, nil
}
