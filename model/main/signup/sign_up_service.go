package signup

import (
	"log"
	"time"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entx"
	"github.com/simpledms/simpledms/model/main/account"
	"github.com/simpledms/simpledms/model/main/common/country"
	"github.com/simpledms/simpledms/model/main/common/language"
	"github.com/simpledms/simpledms/model/main/common/mainrole"
	"github.com/simpledms/simpledms/model/main/common/plan"
	"github.com/simpledms/simpledms/model/main/common/tenantrole"
	"github.com/simpledms/simpledms/model/main/mailer"
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
	password, expiresAt, err := accountm.GenerateTemporaryPassword(ctx)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	if !skipSendingMail {
		mailer.NewMailer().SignUp(ctx, accountx, password, expiresAt, signInURL)
	}

	return accountm, nil
}
