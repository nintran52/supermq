// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package rabbitmq_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/absmach/supermq/pkg/messaging"
	"github.com/absmach/supermq/pkg/messaging/rabbitmq"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

const (
	topic        = "topic"
	msgPrefix    = "m"
	channel      = "9b7b1b3f-b1b0-46a8-a717-b8213f9eda3b"
	subtopic     = "engine"
	clientID     = "9b7b1b3f-b1b0-46a8-a717-b8213f9eda3b"
	exchangeName = "messages"
)

var (
	msgChan = make(chan *messaging.Message)
	data    = []byte("payload")
)

var errFailedHandleMessage = errors.New("failed to handle supermq message")

func TestPublisher(t *testing.T) {
	// Subscribing with topic, and with subtopic, so that we can publish messages.
	conn, ch, err := newConn()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	topicChan := subscribe(t, ch, fmt.Sprintf("%s.%s", msgPrefix, topic))
	subtopicChan := subscribe(t, ch, fmt.Sprintf("%s.%s.%s", msgPrefix, topic, subtopic))

	go rabbitHandler(topicChan, handler{})
	go rabbitHandler(subtopicChan, handler{})

	t.Cleanup(func() {
		conn.Close()
		ch.Close()
	})

	cases := []struct {
		desc     string
		channel  string
		subtopic string
		payload  []byte
	}{
		{
			desc:    "publish message with nil payload",
			payload: nil,
		},
		{
			desc:    "publish message with string payload",
			payload: data,
		},
		{
			desc:    "publish message with channel",
			payload: data,
			channel: channel,
		},
		{
			desc:     "publish message with subtopic",
			payload:  data,
			subtopic: subtopic,
		},
		{
			desc:     "publish message with channel and subtopic",
			payload:  data,
			channel:  channel,
			subtopic: subtopic,
		},
	}

	for _, tc := range cases {
		expectedMsg := messaging.Message{
			Publisher: clientID,
			Channel:   tc.channel,
			Subtopic:  tc.subtopic,
			Payload:   tc.payload,
		}
		err = pubsub.Publish(context.TODO(), topic, &expectedMsg)
		assert.Nil(t, err, fmt.Sprintf("%s: got unexpected error: %s", tc.desc, err))

		receivedMsg := <-msgChan
		assert.Equal(t, expectedMsg.Channel, receivedMsg.Channel, fmt.Sprintf("%s: expected %+v got %+v\n", tc.desc, &expectedMsg, receivedMsg))
		assert.Equal(t, expectedMsg.Created, receivedMsg.Created, fmt.Sprintf("%s: expected %+v got %+v\n", tc.desc, &expectedMsg, receivedMsg))
		assert.Equal(t, expectedMsg.Protocol, receivedMsg.Protocol, fmt.Sprintf("%s: expected %+v got %+v\n", tc.desc, &expectedMsg, receivedMsg))
		assert.Equal(t, expectedMsg.Publisher, receivedMsg.Publisher, fmt.Sprintf("%s: expected %+v got %+v\n", tc.desc, &expectedMsg, receivedMsg))
		assert.Equal(t, expectedMsg.Subtopic, receivedMsg.Subtopic, fmt.Sprintf("%s: expected %+v got %+v\n", tc.desc, &expectedMsg, receivedMsg))
		assert.Equal(t, expectedMsg.Payload, receivedMsg.Payload, fmt.Sprintf("%s: expected %+v got %+v\n", tc.desc, &expectedMsg, receivedMsg))
	}
}

func TestSubscribe(t *testing.T) {
	// Creating rabbitmq connection and channel, so that we can publish messages.
	conn, ch, err := newConn()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	t.Cleanup(func() {
		conn.Close()
		ch.Close()
	})

	cases := []struct {
		desc     string
		topic    string
		clientID string
		err      error
		handler  messaging.MessageHandler
	}{
		{
			desc:     "Subscribe to a topic with an ID",
			topic:    topic,
			clientID: "clientid1",
			err:      nil,
			handler:  handler{false, "clientid1"},
		},
		{
			desc:     "Subscribe to the same topic with a different ID",
			topic:    topic,
			clientID: "clientid2",
			err:      nil,
			handler:  handler{false, "clientid2"},
		},
		{
			desc:     "Subscribe to an already subscribed topic with an ID",
			topic:    topic,
			clientID: "clientid1",
			err:      nil,
			handler:  handler{false, "clientid1"},
		},
		{
			desc:     "Subscribe to a topic with a subtopic with an ID",
			topic:    fmt.Sprintf("%s.%s", topic, subtopic),
			clientID: "clientid1",
			err:      nil,
			handler:  handler{false, "clientid1"},
		},
		{
			desc:     "Subscribe to an already subscribed topic with a subtopic with an ID",
			topic:    fmt.Sprintf("%s.%s", topic, subtopic),
			clientID: "clientid1",
			err:      nil,
			handler:  handler{false, "clientid1"},
		},
		{
			desc:     "Subscribe to an empty topic with an ID",
			topic:    "",
			clientID: "clientid1",
			err:      rabbitmq.ErrEmptyTopic,
			handler:  handler{false, "clientid1"},
		},
		{
			desc:     "Subscribe to a topic with empty id",
			topic:    topic,
			clientID: "",
			err:      rabbitmq.ErrEmptyID,
			handler:  handler{false, ""},
		},
	}
	for _, tc := range cases {
		subCfg := messaging.SubscriberConfig{
			ID:      tc.clientID,
			Topic:   tc.topic,
			Handler: tc.handler,
		}
		err := pubsub.Subscribe(context.TODO(), subCfg)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected: %s, but got: %s", tc.desc, tc.err, err))

		if tc.err == nil {
			expectedMsg := messaging.Message{
				Publisher: "CLIENTID",
				Channel:   channel,
				Subtopic:  subtopic,
				Payload:   data,
			}

			data, err := proto.Marshal(&expectedMsg)
			assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

			err = ch.PublishWithContext(
				context.Background(),
				exchangeName,
				tc.topic,
				false,
				false,
				amqp.Publishing{
					Headers:     amqp.Table{},
					ContentType: "application/octet-stream",
					AppId:       "supermq-publisher",
					Body:        data,
				})
			assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

			receivedMsg := <-msgChan
			assert.Equal(t, expectedMsg.Channel, receivedMsg.Channel, fmt.Sprintf("%s: expected %+v got %+v\n", tc.desc, &expectedMsg, receivedMsg))
			assert.Equal(t, expectedMsg.Created, receivedMsg.Created, fmt.Sprintf("%s: expected %+v got %+v\n", tc.desc, &expectedMsg, receivedMsg))
			assert.Equal(t, expectedMsg.Protocol, receivedMsg.Protocol, fmt.Sprintf("%s: expected %+v got %+v\n", tc.desc, &expectedMsg, receivedMsg))
			assert.Equal(t, expectedMsg.Publisher, receivedMsg.Publisher, fmt.Sprintf("%s: expected %+v got %+v\n", tc.desc, &expectedMsg, receivedMsg))
			assert.Equal(t, expectedMsg.Subtopic, receivedMsg.Subtopic, fmt.Sprintf("%s: expected %+v got %+v\n", tc.desc, &expectedMsg, receivedMsg))
			assert.Equal(t, expectedMsg.Payload, receivedMsg.Payload, fmt.Sprintf("%s: expected %+v got %+v\n", tc.desc, &expectedMsg, receivedMsg))
		}
	}
}

func TestUnsubscribe(t *testing.T) {
	// Test Subscribe and Unsubscribe
	cases := []struct {
		desc      string
		topic     string
		clientID  string
		err       error
		subscribe bool // True for subscribe and false for unsubscribe.
		handler   messaging.MessageHandler
	}{
		{
			desc:      "Subscribe to a topic with an ID",
			topic:     fmt.Sprintf("%s.%s", msgPrefix, topic),
			clientID:  "clientid4",
			err:       nil,
			subscribe: true,
			handler:   handler{false, "clientid4"},
		},
		{
			desc:      "Subscribe to the same topic with a different ID",
			topic:     fmt.Sprintf("%s.%s", msgPrefix, topic),
			clientID:  "clientid9",
			err:       nil,
			subscribe: true,
			handler:   handler{false, "clientid9"},
		},
		{
			desc:      "Unsubscribe from a topic with an ID",
			topic:     fmt.Sprintf("%s.%s", msgPrefix, topic),
			clientID:  "clientid4",
			err:       nil,
			subscribe: false,
			handler:   handler{false, "clientid4"},
		},
		{
			desc:      "Unsubscribe from same topic with different ID",
			topic:     fmt.Sprintf("%s.%s", msgPrefix, topic),
			clientID:  "clientid9",
			err:       nil,
			subscribe: false,
			handler:   handler{false, "clientid9"},
		},
		{
			desc:      "Unsubscribe from a non-existent topic with an ID",
			topic:     "h",
			clientID:  "clientid4",
			err:       rabbitmq.ErrNotSubscribed,
			subscribe: false,
			handler:   handler{false, "clientid4"},
		},
		{
			desc:      "Unsubscribe from an already unsubscribed topic with an ID",
			topic:     fmt.Sprintf("%s.%s", msgPrefix, topic),
			clientID:  "clientid4",
			err:       rabbitmq.ErrNotSubscribed,
			subscribe: false,
			handler:   handler{false, "clientid4"},
		},
		{
			desc:      "Subscribe to a topic with a subtopic with an ID",
			topic:     fmt.Sprintf("%s.%s.%s", msgPrefix, topic, subtopic),
			clientID:  "clientidd4",
			err:       nil,
			subscribe: true,
			handler:   handler{false, "clientidd4"},
		},
		{
			desc:      "Unsubscribe from a topic with a subtopic with an ID",
			topic:     fmt.Sprintf("%s.%s.%s", msgPrefix, topic, subtopic),
			clientID:  "clientidd4",
			err:       nil,
			subscribe: false,
			handler:   handler{false, "clientidd4"},
		},
		{
			desc:      "Unsubscribe from an already unsubscribed topic with a subtopic with an ID",
			topic:     fmt.Sprintf("%s.%s.%s", msgPrefix, topic, subtopic),
			clientID:  "clientid4",
			err:       rabbitmq.ErrNotSubscribed,
			subscribe: false,
			handler:   handler{false, "clientid4"},
		},
		{
			desc:      "Unsubscribe from an empty topic with an ID",
			topic:     "",
			clientID:  "clientid4",
			err:       rabbitmq.ErrEmptyTopic,
			subscribe: false,
			handler:   handler{false, "clientid4"},
		},
		{
			desc:      "Unsubscribe from a topic with empty ID",
			topic:     fmt.Sprintf("%s.%s", msgPrefix, topic),
			clientID:  "",
			err:       rabbitmq.ErrEmptyID,
			subscribe: false,
			handler:   handler{false, ""},
		},
		{
			desc:      "Subscribe to a new topic with an ID",
			topic:     fmt.Sprintf("%s.%s", msgPrefix, topic+"2"),
			clientID:  "clientid55",
			err:       nil,
			subscribe: true,
			handler:   handler{true, "clientid5"},
		},
		{
			desc:      "Unsubscribe from a topic with an ID with failing handler",
			topic:     fmt.Sprintf("%s.%s", msgPrefix, topic+"2"),
			clientID:  "clientid55",
			err:       errFailedHandleMessage,
			subscribe: false,
			handler:   handler{true, "clientid5"},
		},
		{
			desc:      "Subscribe to a new topic with subtopic with an ID",
			topic:     fmt.Sprintf("%s.%s.%s", msgPrefix, topic+"2", subtopic),
			clientID:  "clientid55",
			err:       nil,
			subscribe: true,
			handler:   handler{true, "clientid5"},
		},
		{
			desc:      "Unsubscribe from a topic with subtopic with an ID with failing handler",
			topic:     fmt.Sprintf("%s.%s.%s", msgPrefix, topic+"2", subtopic),
			clientID:  "clientid55",
			err:       errFailedHandleMessage,
			subscribe: false,
			handler:   handler{true, "clientid5"},
		},
	}

	for _, tc := range cases {
		subCfg := messaging.SubscriberConfig{
			ID:      tc.clientID,
			Topic:   tc.topic,
			Handler: tc.handler,
		}
		switch tc.subscribe {
		case true:
			err := pubsub.Subscribe(context.TODO(), subCfg)
			assert.Equal(t, err, tc.err, fmt.Sprintf("%s: expected: %s, but got: %s", tc.desc, tc.err, err))
		default:
			err := pubsub.Unsubscribe(context.TODO(), tc.clientID, tc.topic)
			assert.Equal(t, err, tc.err, fmt.Sprintf("%s: expected: %s, but got: %s", tc.desc, tc.err, err))
		}
	}
}

func TestPubSub(t *testing.T) {
	cases := []struct {
		desc     string
		topic    string
		clientID string
		err      error
		handler  messaging.MessageHandler
	}{
		{
			desc:     "Subscribe to a topic with an ID",
			topic:    topic,
			clientID: clientID,
			err:      nil,
			handler:  handler{false, clientID},
		},
		{
			desc:     "Subscribe to the same topic with a different ID",
			topic:    topic,
			clientID: clientID + "1",
			err:      nil,
			handler:  handler{false, clientID + "1"},
		},
		{
			desc:     "Subscribe to a topic with a subtopic with an ID",
			topic:    fmt.Sprintf("%s.%s", topic, subtopic),
			clientID: clientID + "2",
			err:      nil,
			handler:  handler{false, clientID + "2"},
		},
		{
			desc:     "Subscribe to an empty topic with an ID",
			topic:    "",
			clientID: clientID,
			err:      rabbitmq.ErrEmptyTopic,
			handler:  handler{false, clientID},
		},
		{
			desc:     "Subscribe to a topic with empty id",
			topic:    topic,
			clientID: "",
			err:      rabbitmq.ErrEmptyID,
			handler:  handler{false, ""},
		},
	}
	for _, tc := range cases {
		subject := ""
		if tc.topic != "" {
			subject = fmt.Sprintf("%s.%s", msgPrefix, tc.topic)
		}
		subCfg := messaging.SubscriberConfig{
			ID:      tc.clientID,
			Topic:   subject,
			Handler: tc.handler,
		}
		err := pubsub.Subscribe(context.TODO(), subCfg)

		switch tc.err {
		case nil:
			// If no error, publish message, and receive after subscribing.
			expectedMsg := messaging.Message{
				Channel: channel,
				Payload: data,
			}

			err = pubsub.Publish(context.TODO(), tc.topic, &expectedMsg)
			assert.Nil(t, err, fmt.Sprintf("%s got unexpected error: %s", tc.desc, err))

			receivedMsg := <-msgChan
			assert.Equal(t, expectedMsg.Channel, receivedMsg.Channel, fmt.Sprintf("%s: expected %+v got %+v\n", tc.desc, &expectedMsg, receivedMsg))
			assert.Equal(t, expectedMsg.Payload, receivedMsg.Payload, fmt.Sprintf("%s: expected %+v got %+v\n", tc.desc, &expectedMsg, receivedMsg))

			err = pubsub.Unsubscribe(context.TODO(), tc.clientID, fmt.Sprintf("%s.%s", msgPrefix, tc.topic))
			assert.Nil(t, err, fmt.Sprintf("%s got unexpected error: %s", tc.desc, err))
		default:
			assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected: %s, but got: %s", tc.desc, err, tc.err))
		}
	}
}

type handler struct {
	fail      bool
	publisher string
}

func (h handler) Handle(msg *messaging.Message) error {
	if msg.GetPublisher() != h.publisher {
		msgChan <- msg
	}
	return nil
}

func (h handler) Cancel() error {
	if h.fail {
		return errFailedHandleMessage
	}
	return nil
}
