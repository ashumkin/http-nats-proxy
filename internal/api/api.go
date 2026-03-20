// Package api is a package.
package api

import (
	"bytes"
	"context"
	"http-nats-proxy/api/restapi"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/nats-io/nats.go"
)

const defaultTimeout = time.Second * 5

// Server is a server.
type Server struct {
	nc             *nats.Conn
	defaultTimeout time.Duration
}

// ServerOpt is the functional option.
type ServerOpt func(*Server)

// NewServer returns Server.
func NewServer(nc *nats.Conn, opts ...ServerOpt) *Server {
	s := Server{
		nc:             nc,
		defaultTimeout: defaultTimeout,
	}
	for _, opt := range opts {
		opt(&s)
	}

	return &s
}

// WithDefaultTimeout sets default timeout.
func WithDefaultTimeout(d time.Duration) ServerOpt {
	return func(s *Server) {
		s.defaultTimeout = d
	}
}

// V1RequestReplyPost implements interface.
// nolint:ireturn
func (s Server) V1RequestReplyPost(
	_ context.Context,
	req *restapi.V1RequestReplyPostReqWithContentType,
	params restapi.V1RequestReplyPostParams,
) (restapi.V1RequestReplyPostRes, error) {
	var timeoutDuration time.Duration
	var err error
	timeout := params.ReplyTimeout.Value
	if timeout == "" {
		timeoutDuration = s.defaultTimeout
	} else {
		timeoutDuration, err = time.ParseDuration(timeout)
		if err != nil {
			// nolint:nilerr
			return &restapi.V1RequestReplyPostBadRequest{}, nil
		}
	}
	msg := nats.NewMsg(params.Subject)
	log := slog.With("subject", msg.Subject, "request_id", params.XRequestID.Value)

	msg.Data, err = io.ReadAll(req.Content.Data)
	if err != nil {
		log.Error("failed to read message body", "err", err)

		return &restapi.V1RequestReplyPostBadRequest{}, nil
	}

	log.Debug("requesting", "timeout", timeoutDuration.String())
	start := time.Now()
	rMsg, err := s.nc.RequestMsg(msg, timeoutDuration)
	if err != nil {
		log.Error("error receiving reply", "err", err)

		return nil, err
	}
	replyRTT := time.Since(start)
	log.Debug("got reply", "rtt", replyRTT.String())

	data := bytes.NewReader(rMsg.Data)

	return &restapi.V1RequestReplyPostOKHeaders{
		Rtt:      replyRTT.String(),
		Response: restapi.V1RequestReplyPostOK{Data: data}}, nil
}

// NewError returns error.
func (s Server) NewError(_ context.Context, err error) *restapi.ErrorStatusCode {
	return &restapi.ErrorStatusCode{
		StatusCode: http.StatusInternalServerError,
		Response: restapi.Error{
			Message: err.Error(),
		},
	}
}
