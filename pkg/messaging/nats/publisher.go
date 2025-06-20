// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package nats

import (
	"context"
	"fmt"

	"github.com/absmach/supermq/pkg/events"
	"github.com/absmach/supermq/pkg/messaging"
	broker "github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"google.golang.org/protobuf/proto"
)

const (
	// A maximum number of reconnect attempts before NATS connection closes permanently.
	// Value -1 represents an unlimited number of reconnect retries, i.e. the client
	// will never give up on retrying to re-establish connection to NATS server.
	maxReconnects = -1

	// reconnectBufSize is obtained from the maximum number of unpublished events
	// multiplied by the approximate maximum size of a single event.
	reconnectBufSize = events.MaxUnpublishedEvents * (1024 * 1024)
)

var _ messaging.Publisher = (*publisher)(nil)

type publisher struct {
	js   jetstream.JetStream
	conn *broker.Conn
	options
}

// NewPublisher returns NATS message Publisher.
func NewPublisher(ctx context.Context, url string, opts ...messaging.Option) (messaging.Publisher, error) {
	pub := &publisher{
		options: defaultOptions(),
	}

	for _, opt := range opts {
		if err := opt(pub); err != nil {
			return nil, err
		}
	}

	conn, err := broker.Connect(url, broker.MaxReconnects(maxReconnects), broker.ReconnectBufSize(int(reconnectBufSize)))
	if err != nil {
		return nil, err
	}
	pub.conn = conn

	js, err := jetstream.New(conn)
	if err != nil {
		return nil, err
	}
	if _, err := js.CreateStream(ctx, pub.jsStreamConfig); err != nil {
		return nil, err
	}
	pub.js = js

	return pub, nil
}

func (pub *publisher) Publish(ctx context.Context, topic string, msg *messaging.Message) error {
	if topic == "" {
		return ErrEmptyTopic
	}

	data, err := proto.Marshal(msg)
	if err != nil {
		return err
	}

	subject := fmt.Sprintf("%s.%s", pub.prefix, topic)
	if _, err = pub.js.Publish(ctx, subject, data); err != nil {
		return err
	}

	return nil
}

func (pub *publisher) Close() error {
	pub.conn.Close()
	return nil
}
