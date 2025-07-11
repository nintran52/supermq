// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/absmach/supermq"
	grpcChannelsV1 "github.com/absmach/supermq/api/grpc/channels/v1"
	"github.com/absmach/supermq/coap"
	"github.com/absmach/supermq/pkg/errors"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	"github.com/absmach/supermq/pkg/messaging"
	"github.com/go-chi/chi/v5"
	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/plgd-dev/go-coap/v3/message/pool"
	"github.com/plgd-dev/go-coap/v3/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	protocol     = "coap"
	authQuery    = "auth"
	startObserve = 0 // observe option value that indicates start of observation
)

var (
	errBadOptions       = errors.New("bad options")
	errMethodNotAllowed = errors.New("method not allowed")
)

// MakeHandler returns a HTTP handler for API endpoints.
func MakeHandler(instanceID string) http.Handler {
	b := chi.NewRouter()
	b.Get("/health", supermq.Health(protocol, instanceID))
	b.Handle("/metrics", promhttp.Handler())

	return b
}

type CoAPHandler struct {
	logger   *slog.Logger
	service  coap.Service
	channels grpcChannelsV1.ChannelsServiceClient
	resolver messaging.TopicResolver
}

// MakeCoAPHandler creates handler for CoAP messages.
func MakeCoAPHandler(svc coap.Service, channelsClient grpcChannelsV1.ChannelsServiceClient, resolver messaging.TopicResolver, l *slog.Logger) mux.Handler {
	return &CoAPHandler{
		logger:   l,
		service:  svc,
		channels: channelsClient,
		resolver: resolver,
	}
}

// ServeCOAP implements the mux.Handler interface for handling CoAP messages.
func (h *CoAPHandler) ServeCOAP(w mux.ResponseWriter, m *mux.Message) {
	resp := pool.NewMessage(w.Conn().Context())
	resp.SetToken(m.Token())
	for _, opt := range m.Options() {
		resp.AddOptionBytes(opt.ID, opt.Value)
	}
	defer h.sendResp(w, resp)

	msg, err := h.decodeMessage(m)
	if err != nil {
		h.logger.Warn(fmt.Sprintf("Error decoding message: %s", err))
		resp.SetCode(codes.BadRequest)
		return
	}
	key, err := parseKey(m)
	if err != nil {
		h.logger.Warn(fmt.Sprintf("Error parsing auth: %s", err))
		resp.SetCode(codes.Unauthorized)
		return
	}

	switch m.Code() {
	case codes.GET:
		resp.SetCode(codes.Content)
		err = h.handleGet(m, w, msg, key)
	case codes.POST:
		resp.SetCode(codes.Created)
		err = h.service.Publish(m.Context(), key, msg)
	default:
		err = errMethodNotAllowed
	}

	if err != nil {
		switch {
		case err == errBadOptions:
			resp.SetCode(codes.BadOption)
		case err == errMethodNotAllowed:
			resp.SetCode(codes.MethodNotAllowed)
		case errors.Contains(err, svcerr.ErrAuthorization):
			resp.SetCode(codes.Forbidden)
		case errors.Contains(err, svcerr.ErrAuthentication):
			resp.SetCode(codes.Unauthorized)
		default:
			resp.SetCode(codes.InternalServerError)
		}
	}
}

func (h *CoAPHandler) handleGet(m *mux.Message, w mux.ResponseWriter, msg *messaging.Message, key string) error {
	var obs uint32
	obs, err := m.Options().Observe()
	if err != nil {
		h.logger.Warn(fmt.Sprintf("Error reading observe option: %s", err))
		return errBadOptions
	}
	if obs == startObserve {
		c := coap.NewClient(w.Conn(), m.Token(), h.logger)
		w.Conn().AddOnClose(func() {
			_ = h.service.DisconnectHandler(context.Background(), msg.GetDomain(), msg.GetChannel(), msg.GetSubtopic(), c.Token())
		})
		return h.service.Subscribe(w.Conn().Context(), key, msg.GetDomain(), msg.GetChannel(), msg.GetSubtopic(), c)
	}
	return h.service.Unsubscribe(w.Conn().Context(), key, msg.GetDomain(), msg.GetChannel(), msg.GetSubtopic(), m.Token().String())
}

func (h *CoAPHandler) decodeMessage(msg *mux.Message) (*messaging.Message, error) {
	if msg.Options() == nil {
		return &messaging.Message{}, errBadOptions
	}
	path, err := msg.Path()
	if err != nil {
		return &messaging.Message{}, err
	}

	var domain, channel, subTopic string
	switch msg.Code() {
	case codes.GET:
		domain, channel, subTopic, err = messaging.ParseSubscribeTopic(path)
	case codes.POST:
		domain, channel, subTopic, err = messaging.ParsePublishTopic(path)
	}
	if err != nil {
		return &messaging.Message{}, err
	}

	domainID, channelID, err := h.resolver.Resolve(msg.Context(), domain, channel)
	if err != nil {
		return &messaging.Message{}, err
	}

	ret := &messaging.Message{
		Protocol: protocol,
		Domain:   domainID,
		Channel:  channelID,
		Subtopic: subTopic,
		Payload:  []byte{},
		Created:  time.Now().UnixNano(),
	}

	if msg.Body() != nil {
		buff, err := io.ReadAll(msg.Body())
		if err != nil {
			return ret, err
		}
		ret.Payload = buff
	}
	return ret, nil
}

func (h *CoAPHandler) sendResp(w mux.ResponseWriter, resp *pool.Message) {
	if err := w.Conn().WriteMessage(resp); err != nil {
		h.logger.Warn(fmt.Sprintf("Can't set response: %s", err))
	}
}

func parseKey(msg *mux.Message) (string, error) {
	authKey, err := msg.Options().GetString(message.URIQuery)
	if err != nil {
		return "", err
	}
	vars := strings.Split(authKey, "=")
	if len(vars) != 2 || vars[0] != authQuery {
		return "", svcerr.ErrAuthorization
	}
	return vars[1], nil
}
