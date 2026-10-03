package session

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

// DeleteOtherSessionsOfAccount signs out all other devices, for example after a password
// change, so that a compromised session does not survive the change.
func (qq *SessionService) DeleteOtherSessionsOfAccount(
	ctx ctxx.Context,
	accountID int64,
	currentSessionValue string,
) error {
	_, err := ctx.MainCtx().MainTx.Session.Delete().
		Where(
			session.AccountID(accountID),
			session.ValueNEQ(currentSessionValue),
		).
		Exec(ctx)
	return err
}
