package user

import (
	"strings"
	"time"

	"github.com/marcobeierer/go-core/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	enttenantprivacy "github.com/simpledms/simpledms/db/enttenant/privacy"
	"github.com/simpledms/simpledms/db/enttenant/spaceuserassignment"
)

type User struct {
	Data *enttenant.User
}

func NewUser(data *enttenant.User) *User {
	return &User{
		Data: data,
	}
}

// same in Account.Name()
func (qq *User) Name() string {
	var elems []string
	if qq.Data.FirstName != "" {
		elems = append(elems, qq.Data.FirstName)
	}
	if qq.Data.LastName != "" {
		elems = append(elems, qq.Data.LastName)
	}
	if len(elems) > 0 {
		return strings.Join(elems, " ")
	}
	// TODO does this expose to many details?
	return qq.Data.Email.String()
}

// returns email address if FirstName or LastName is set, otherwise empty string
func (qq *User) NameSecondLine() string {
	name := qq.Name()
	if name == qq.Data.Email.String() {
		return ""
	}
	return qq.Data.Email.String()
}

func (qq *User) Delete(ctx ctxx.Context, deletedBy int64) error {
	ctxWithPrivacyOverride := enttenantprivacy.DecisionContext(ctx, enttenantprivacy.Allow)

	_, err := ctx.TenantCtx().TTx.SpaceUserAssignment.Delete().
		Where(spaceuserassignment.UserID(qq.Data.ID)).
		Exec(ctxWithPrivacyOverride)
	if err != nil {
		return err
	}

	userx, err := qq.Data.Update().
		SetDeletedAt(time.Now()).
		SetDeletedBy(deletedBy).
		Save(ctx)
	if err != nil {
		return err
	}

	qq.Data = userx

	return nil
}
