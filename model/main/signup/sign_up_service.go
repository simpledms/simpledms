package signup

import (
	"log"
	"net/http"
	"time"

	"entgo.io/ent/privacy"

	"github.com/simpledms/simpledms/ctxx"
	accountdb "github.com/simpledms/simpledms/db/entmain/account"
	"github.com/simpledms/simpledms/db/entx"
	"github.com/simpledms/simpledms/model/main/account"
	"github.com/simpledms/simpledms/model/main/common/country"
	"github.com/simpledms/simpledms/model/main/common/language"
	"github.com/simpledms/simpledms/model/main/common/mainrole"
	"github.com/simpledms/simpledms/model/main/common/plan"
	"github.com/simpledms/simpledms/model/main/common/tenantrole"
	"github.com/simpledms/simpledms/model/main/mailer"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/timex"
)

type SignUpService struct{}

func NewSignUpService() *SignUpService {
	return &SignUpService{}
}

func (qq *SignUpService) SignUp(
	ctx ctxx.Context,
	email string,
	tenantName string,
	firstName string,
	lastName string,
	country country.Country,
	language language.Language,
	subscribeToNewsletter bool,
	skipSendingMail bool,
	signInURL string,
) (*account.Account, error) {
	// Email uniqueness is global, including accounts hidden from the current admin.
	// Scope the privacy bypass to this lookup and reject before creating a tenant.
	emailExists, err := ctx.VisitorCtx().MainTx.Account.Query().
		Where(accountdb.EmailEQ(entx.NewCIText(email)), accountdb.DeletedAtIsNil()).
		Exist(privacy.DecisionContext(ctx, privacy.Allow))
	if err != nil {
		log.Println(err)
		return nil, err
	}
	if emailExists {
		return nil, e.NewHTTPErrorf(http.StatusBadRequest,
			"An account with this email address already exists. Please sign in or reset your password.")
	}

	tenantxQuery := ctx.VisitorCtx().MainTx.Tenant.
		Create().
		SetName(tenantName).
		SetCountry(country).
		SetPlan(plan.Trial)
	/* FIXME
	if data.AcceptPrivacyPolicy {
		tenantxQuery = tenantxQuery.SetPrivacyPolicyAccepted(time.Now())
	}
	if data.AcceptTermsOfService {
		tenantxQuery = tenantxQuery.SetTermsOfServiceAccepted(time.Now())
	}
	*/
	tenantxQuery = tenantxQuery.SetPrivacyPolicyAccepted(timex.NewDateTimeZero().Time)
	tenantxQuery = tenantxQuery.SetTermsOfServiceAccepted(timex.NewDateTimeZero().Time)
	tenantx := tenantxQuery.SaveX(ctx)

	accountQuery := ctx.VisitorCtx().MainTx.Account.
		Create().
		SetFirstName(firstName).
		SetLastName(lastName).
		SetLanguage(language).
		SetEmail(entx.NewCIText(email)).
		SetRole(mainrole.User)
	if subscribeToNewsletter {
		accountQuery.SetSubscribedToNewsletterAt(time.Now())
	}
	accountx := accountQuery.SaveX(ctx)

	_ = ctx.VisitorCtx().MainTx.TenantAccountAssignment.
		Create().
		SetTenant(tenantx).
		SetAccount(accountx).
		SetIsContactPerson(true).
		SetRole(tenantrole.Owner).
		SetIsOwningTenant(true).
		SetIsDefault(true).
		SaveX(ctx)

	// Tenant.User is created in tenant initialization

	accountm := account.NewAccount(accountx)
	// The new account is not visible to the authenticated admin under account privacy rules when a tenant is created from the manageTenant screen
	ctxWithPrivacyBypass := privacy.DecisionContext(ctx, privacy.Allow)
	password, expiresAt, err := accountm.GenerateTemporaryPassword(ctxWithPrivacyBypass)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	if !skipSendingMail {
		mailer.NewMailer().SignUp(ctx, accountx, password, expiresAt, signInURL)
	}

	return accountm, nil
}
