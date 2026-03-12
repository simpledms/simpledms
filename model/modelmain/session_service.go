package modelmain

import (
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain/session"
)

type SessionService struct{}

func NewSessionService() *SessionService {
	return &SessionService{}
}

func (qq *SessionService) DeleteByValue(ctx ctxx.Context, sessionValue string) error {
	_, err := ctx.MainCtx().MainTx.Session.Delete().Where(session.Value(sessionValue)).Exec(ctx)
	return err
}
