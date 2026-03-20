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

const defaultTimeout = "5s"

// Server is a server.
type Server struct {
	nc *nats.Conn
}

// NewServer returns Server.
func NewServer(nc *nats.Conn) *Server {
	return &Server{nc: nc}
}

// V1RequestReplyPost implements interface.
// nolint:ireturn
func (s Server) V1RequestReplyPost(
	_ context.Context,
	req *restapi.V1RequestReplyPostReqWithContentType,
	params restapi.V1RequestReplyPostParams,
) (restapi.V1RequestReplyPostRes, error) {
	timeout := params.NatsReplyTimeout.Value
	if timeout == "" {
		timeout = defaultTimeout
	}
	timeoutDuration, err := time.ParseDuration(timeout)
	if err != nil {
		// nolint:nilerr
		return &restapi.V1RequestReplyPostBadRequest{}, nil
	}
	msg := nats.NewMsg(params.Subject)
	log := slog.With("subject", msg.Subject, "request_id", params.XRequestID)

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
