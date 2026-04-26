package tenantuser

import (
	"github.com/marcobeierer/go-core/db/entmain"
	"github.com/marcobeierer/go-core/db/entmain/account"
	"github.com/marcobeierer/go-core/db/entx"
	"github.com/marcobeierer/go-core/model/common/language"
	"github.com/marcobeierer/go-core/model/common/mainrole"
	"github.com/marcobeierer/go-core/model/common/tenantrole"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/user"
)

type TenantUserRepository interface {
	AccountExistsByEmail(ctx ctxx.Context, email string) (bool, error)
	CreateAccount(
		ctx ctxx.Context,
		firstName string,
		lastName string,
		language language.Language,
		email string,
	) (*entmain.Account, error)
	CreateTenantUser(ctx ctxx.Context, accountx *entmain.Account, role tenantrole.TenantRole) (*enttenant.User, error)
	UserByPublicID(ctx ctxx.Context, userPublicID string) (*enttenant.User, error)
}

type EntTenantUserRepository struct{}

var _ TenantUserRepository = (*EntTenantUserRepository)(nil)

func NewEntTenantUserRepository() *EntTenantUserRepository {
	return &EntTenantUserRepository{}
}

func (qq *EntTenantUserRepository) AccountExistsByEmail(ctx ctxx.Context, email string) (bool, error) {
	return ctx.TenantCtx().MainTx.Account.Query().
		Where(account.EmailEQ(entx.NewCIText(email))).
		Exist(ctx)
}

func (qq *EntTenantUserRepository) CreateAccount(
	ctx ctxx.Context,
	firstName string,
	lastName string,
	language language.Language,
	email string,
) (*entmain.Account, error) {
	return ctx.TenantCtx().MainTx.Account.Create().
		SetFirstName(firstName).
		SetLastName(lastName).
		SetLanguage(language).
		SetEmail(entx.NewCIText(email)).
		SetRole(mainrole.User).
		Save(ctx)
}

func (qq *EntTenantUserRepository) CreateTenantUser(
	ctx ctxx.Context,
	accountx *entmain.Account,
	role tenantrole.TenantRole,
) (*enttenant.User, error) {
	return ctx.AppCtx().TTx.User.Create().
		SetAccountID(accountx.ID).
		SetRole(role).
		SetEmail(accountx.Email).
		SetFirstName(accountx.FirstName).
		SetLastName(accountx.LastName).
		Save(ctx)
}

func (qq *EntTenantUserRepository) UserByPublicID(ctx ctxx.Context, userPublicID string) (*enttenant.User, error) {
	return ctx.AppCtx().TTx.User.Query().
		Where(user.PublicID(entx.NewCIText(userPublicID))).
		Only(ctx)
}
