package server

import (
	"errors"
	"net"
)

const closeErrorListenerMessage = "injected listener close error"

type closeErrorListener struct {
	net.Listener
}

func newCloseErrorListener(listener net.Listener) *closeErrorListener {
	return &closeErrorListener{
		Listener: listener,
	}
}

func (qq *closeErrorListener) Close() error {
	return errors.Join(qq.Listener.Close(), errors.New(closeErrorListenerMessage))
}
