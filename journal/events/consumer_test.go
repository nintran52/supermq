// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package events_test

import (
	"context"
	"encoding/json"
	"errors"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/absmach/supermq/internal/testsutil"
	"github.com/absmach/supermq/journal"
	aevents "github.com/absmach/supermq/journal/events"
	"github.com/absmach/supermq/journal/mocks"
	repoerr "github.com/absmach/supermq/pkg/errors/repository"
	"github.com/absmach/supermq/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	operation = "users.create"
	payload   = map[string]interface{}{
		"temperature": rand.Float64(),
		"humidity":    float64(rand.Intn(1000)),
		"locations": []interface{}{
			strings.Repeat("a", 100),
			strings.Repeat("a", 100),
		},
		"status": "active",
	}
	idProvider = uuid.New()
)

type testEvent struct {
	data map[string]interface{}
	err  error
}

func (e testEvent) Encode() (map[string]interface{}, error) {
	return e.data, e.err
}

func NewTestEvent(data map[string]interface{}, err error) testEvent {
	return testEvent{data: data, err: err}
}

func TestHandle(t *testing.T) {
	repo := new(mocks.Repository)
	svc := journal.NewService(idProvider, repo)

	cases := []struct {
		desc      string
		event     map[string]interface{}
		encodeErr error
		repoErr   error
		err       error
	}{
		{
			desc: "success",
			event: map[string]interface{}{
				"operation":   operation,
				"occurred_at": float64(time.Now().UnixNano()),
				"id":          testsutil.GenerateUUID(t),
				"tags":        []interface{}{testsutil.GenerateUUID(t), testsutil.GenerateUUID(t)},
				"number":      float64(rand.Intn(1000)),
				"metadata":    payload,
			},
			err: nil,
		},
		{
			desc: "with encode error",
			event: map[string]interface{}{
				"operation":   operation,
				"occurred_at": float64(time.Now().UnixNano()),
				"id":          testsutil.GenerateUUID(t),
				"tags":        []interface{}{testsutil.GenerateUUID(t), testsutil.GenerateUUID(t)},
				"number":      float64(rand.Intn(1000)),
				"metadata":    payload,
			},
			encodeErr: errors.New("encode error"),
			err:       errors.New("encode error"),
		},
		{
			desc: "with missing operation",
			event: map[string]interface{}{
				"occurred_at": float64(time.Now().UnixNano()),
				"id":          testsutil.GenerateUUID(t),
				"tags":        []interface{}{testsutil.GenerateUUID(t), testsutil.GenerateUUID(t)},
				"number":      float64(rand.Intn(1000)),
				"metadata":    payload,
			},
			err: nil,
		},
		{
			desc: "with empty operation",
			event: map[string]interface{}{
				"operation":   "",
				"occurred_at": float64(time.Now().UnixNano()),
				"id":          testsutil.GenerateUUID(t),
				"tags":        []interface{}{testsutil.GenerateUUID(t), testsutil.GenerateUUID(t)},
				"number":      float64(rand.Intn(1000)),
				"metadata":    payload,
			},
			err: nil,
		},
		{
			desc: "with invalid operation",
			event: map[string]interface{}{
				"operation":   1,
				"occurred_at": float64(time.Now().UnixNano()),
				"id":          testsutil.GenerateUUID(t),
				"tags":        []interface{}{testsutil.GenerateUUID(t), testsutil.GenerateUUID(t)},
				"number":      float64(rand.Intn(1000)),
				"metadata":    payload,
			},
			err: nil,
		},
		{
			desc: "with missing occurred_at",
			event: map[string]interface{}{
				"operation": operation,
				"id":        testsutil.GenerateUUID(t),
				"tags":      []interface{}{testsutil.GenerateUUID(t), testsutil.GenerateUUID(t)},
				"number":    float64(rand.Intn(1000)),
				"metadata":  payload,
			},
			err: nil,
		},
		{
			desc: "with empty occurred_at",
			event: map[string]interface{}{
				"operation":   operation,
				"occurred_at": float64(0),
				"id":          testsutil.GenerateUUID(t),
				"tags":        []interface{}{testsutil.GenerateUUID(t), testsutil.GenerateUUID(t)},
				"number":      float64(rand.Intn(1000)),
				"metadata":    payload,
			},
			err: nil,
		},
		{
			desc: "with invalid occurred_at",
			event: map[string]interface{}{
				"operation":   operation,
				"occurred_at": "invalid",
				"id":          testsutil.GenerateUUID(t),
				"tags":        []interface{}{testsutil.GenerateUUID(t), testsutil.GenerateUUID(t)},
				"number":      float64(rand.Intn(1000)),
				"metadata":    payload,
			},
			err: nil,
		},
		{
			desc: "with missing metadata",
			event: map[string]interface{}{
				"operation":   operation,
				"occurred_at": float64(time.Now().UnixNano()),
				"id":          testsutil.GenerateUUID(t),
				"tags":        []interface{}{testsutil.GenerateUUID(t), testsutil.GenerateUUID(t)},
				"number":      float64(rand.Intn(1000)),
			},
			err: nil,
		},
		{
			desc: "with empty metadata",
			event: map[string]interface{}{
				"operation":   operation,
				"occurred_at": float64(time.Now().UnixNano()),
				"id":          testsutil.GenerateUUID(t),
				"tags":        []interface{}{testsutil.GenerateUUID(t), testsutil.GenerateUUID(t)},
				"number":      float64(rand.Intn(1000)),
				"metadata":    map[string]interface{}{},
			},
			err: nil,
		},
		{
			desc: "with invalid metadata",
			event: map[string]interface{}{
				"operation":   operation,
				"occurred_at": float64(time.Now().UnixNano()),
				"id":          testsutil.GenerateUUID(t),
				"tags":        []interface{}{testsutil.GenerateUUID(t), testsutil.GenerateUUID(t)},
				"number":      float64(rand.Intn(1000)),
				"metadata":    1,
			},
			err: nil,
		},
		{
			desc: "with missing attributes",
			event: map[string]interface{}{
				"operation":   operation,
				"occurred_at": float64(time.Now().UnixNano()),
				"metadata":    payload,
			},
			err: nil,
		},
		{
			desc: "with empty attributes",
			event: map[string]interface{}{
				"operation":   operation,
				"occurred_at": float64(time.Now().UnixNano()),
				"id":          "",
				"tags":        []interface{}{},
				"number":      float64(0),
				"metadata":    payload,
			},
			err: nil,
		},
		{
			desc: "with invalid attributes",
			event: map[string]interface{}{
				"operation":   operation,
				"occurred_at": float64(time.Now().UnixNano()),
				"nested": map[string]interface{}{
					"key": float64(rand.Intn(1000)),
					"nested": map[string]interface{}{
						"key": float64(rand.Intn(1000)),
						"nested": map[string]interface{}{
							"key": float64(rand.Intn(1000)),
							"nested": map[string]interface{}{
								"key": float64(rand.Intn(1000)),
								"nested": map[string]interface{}{
									"key": float64(rand.Intn(1000)),
									"nested": map[string]interface{}{
										"key": float64(rand.Intn(1000)),
									},
								},
							},
						},
					},
				},
				"metadata": payload,
			},
			err: nil,
		},
		{
			desc: "success",
			event: map[string]interface{}{
				"operation":   operation,
				"occurred_at": float64(time.Now().UnixNano()),
				"id":          testsutil.GenerateUUID(t),
				"tags":        []interface{}{testsutil.GenerateUUID(t), testsutil.GenerateUUID(t)},
				"number":      float64(rand.Intn(1000)),
				"metadata":    payload,
			},
			repoErr: repoerr.ErrCreateEntity,
			err:     nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			data, err := json.Marshal(tc.event)
			assert.NoError(t, err)

			event := map[string]interface{}{}
			err = json.Unmarshal(data, &event)
			assert.NoError(t, err)

			repoCall := repo.On("Save", context.Background(), mock.Anything).Return(tc.repoErr)
			err = aevents.Handle(svc)(context.Background(), NewTestEvent(event, tc.encodeErr))
			switch {
			case tc.err == nil:
				assert.NoError(t, err)
			default:
				assert.ErrorContains(t, err, tc.err.Error())
			}
			repoCall.Unset()
		})
	}
}
