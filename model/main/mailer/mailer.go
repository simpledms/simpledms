package mailer

import (
	"time"

	wx "github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/i18n"
)

// TODO good location?
type Mailer struct {
}

func NewMailer() *Mailer {
	return &Mailer{}
}

func (qq *Mailer) TenantRegistered(
	ctx ctxx.Context,
	adminx *entmain.Account,
	tenantName, accountEmail string,
) {
	subject := wx.T("New organization registration").String(ctx)
	template := EmailTemplate{
		Title:   subject,
		Heading: subject,
		Content: []ContentBlock{
			TextBlock{Text: wx.Tuf("The organization «%s» was registered by %s.", tenantName, accountEmail).String(ctx)},
		},
		Footer: wx.T("This is an automated message, please do not reply.").String(ctx),
	}

	qq.queueMail(ctx, adminx, subject, template)
}

func (qq *Mailer) SignUp(
	ctx ctxx.Context,
	accountx *entmain.Account,
	tmpPassword string,
	expiresAt time.Time,
	signInURL string,
) {
	// Use the recipient's selected language without changing the request's language.
	mailCtx := *ctx.VisitorCtx()
	mailCtx.Printer = i18n.NewI18n().Printer(accountx.Language.Tag())
	mailCtx.LanguageBCP47 = accountx.Language.Tag().String()
	ctx = &mailCtx

	// wx.T for translations
	subject := wx.T("Welcome to SimpleDMS").String(ctx)
	template := CreateSignUpTemplate(
		ctx,
		tmpPassword,
		qq.formatExpiresAt(ctx, expiresAt),
		signInURL,
	)
	qq.queueMail(ctx, accountx, subject, template)
}

func (qq *Mailer) ResetPassword(
	ctx ctxx.Context,
	accountx *entmain.Account,
	tmpPassword string,
	expiresAt time.Time,
	signInURL string,
) {
	// wx.T for translations
	subject := wx.T("SimpleDMS password reset").String(ctx)
	template := CreateResetPasswordTemplate(
		ctx,
		tmpPassword,
		qq.formatExpiresAt(ctx, expiresAt),
		signInURL,
	)
	qq.queueMail(ctx, accountx, subject, template)
}

func (qq *Mailer) CreateUser(
	ctx ctxx.Context,
	accountx *entmain.Account,
	tmpPassword string,
	expiresAt time.Time,
	signInURL string,
) {
	subject := wx.T("Welcome to SimpleDMS").String(ctx)
	template := CreateUserTemplate(ctx, tmpPassword, qq.formatExpiresAt(ctx, expiresAt), signInURL)
	qq.queueMail(ctx, accountx, subject, template)
}

func (qq *Mailer) formatExpiresAt(ctx ctxx.Context, expiresAt time.Time) string {
	layout := wx.T("02-01-2006 at 15:04 o'clock").String(ctx)
	return expiresAt.In(ctx.VisitorCtx().Location).Format(layout)
}

// queueMail stores the mail with HTML and plain text versions of template for sending.
func (qq *Mailer) queueMail(
	ctx ctxx.Context,
	receiver *entmain.Account,
	subject string,
	template EmailTemplate,
) {
	ctx.VisitorCtx().MainTx.Mail.Create().
		SetSubject(subject).
		SetBody(template.RenderPlainText(ctx)).
		SetHTMLBody(template.RenderHTML(ctx)).
		SetReceiver(receiver).
		SaveX(ctx)
}
