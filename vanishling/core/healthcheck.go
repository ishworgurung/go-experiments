package core

import (
	"context"
	"net/http"

	"github.com/rs/zerolog"
)

type healthCheck struct {
	ctx  context.Context
	zlog zerolog.Logger
}

func (h healthCheck) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.zlog.Debug().Msgf("health request from %s", r.RemoteAddr)
	w.WriteHeader(http.StatusOK)
}

func NewHealthCheck(ctx context.Context, zlog zerolog.Logger) (*healthCheck, error) {
	return &healthCheck{
		zlog: zlog,
		ctx:  ctx,
	}, nil
}
