package mailer

import (
	"time"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/entmain"
	wx "github.com/simpledms/simpledms/ui/widget"
)

// TODO good location?
type Mailer struct {
}

func NewMailer() *Mailer {
	return &Mailer{}
}

func (qq *Mailer) SignUp(ctx ctxx.Context, accountx *entmain.Account, tmpPassword string, expiresAt time.Time) {
	// wx.T for translations
	subject := wx.T("Welcome to SimpleDMS").String(ctx)

	// Format the expiration date
	formattedExpiresAt := expiresAt.In(ctx.VisitorCtx().Location).Format(wx.T("02-01-2006 at 15:04 o'clock").String(ctx))

	// Create email template
	template := CreateSignUpTemplate(ctx, tmpPassword, formattedExpiresAt)

	// Generate HTML and plain text versions
	htmlBody := template.RenderHTML(ctx)
	plainBody := template.RenderPlainText(ctx)

	ctx.VisitorCtx().MainTx.Mail.Create().
		SetSubject(subject).
		SetBody(plainBody).
		SetHTMLBody(htmlBody).
		SetReceiver(accountx).
		SaveX(ctx)
}

func (qq *Mailer) ResetPassword(ctx ctxx.Context, accountx *entmain.Account, tmpPassword string, expiresAt time.Time) {
	// wx.T for translations
	subject := wx.T("SimpleDMS password reset").String(ctx)

	// Format the expiration date
	formattedExpiresAt := expiresAt.In(ctx.VisitorCtx().Location).Format(wx.T("02-01-2006 at 15:04 o'clock").String(ctx))

	// Create email template
	template := CreateResetPasswordTemplate(ctx, tmpPassword, formattedExpiresAt)

	// Generate HTML and plain text versions
	htmlBody := template.RenderHTML(ctx)
	plainBody := template.RenderPlainText(ctx)

	ctx.VisitorCtx().MainTx.Mail.Create().
		SetSubject(subject).
		SetBody(plainBody).
		SetHTMLBody(htmlBody).
		SetReceiver(accountx).
		SaveX(ctx)
}

func (qq *Mailer) CreateUser(ctx ctxx.Context, accountx *entmain.Account, tmpPassword string, expiresAt time.Time) {
	subject := wx.T("Welcome to SimpleDMS").String(ctx)
	formattedExpiresAt := expiresAt.In(ctx.VisitorCtx().Location).Format(wx.T("02-01-2006 at 15:04 o'clock").String(ctx))

	// Create email template
	template := CreateUserTemplate(ctx, tmpPassword, formattedExpiresAt)

	// Generate HTML and plain text versions
	htmlBody := template.RenderHTML(ctx)
	plainBody := template.RenderPlainText(ctx)

	ctx.VisitorCtx().MainTx.Mail.Create().
		SetSubject(subject).
		SetBody(plainBody).
		SetHTMLBody(htmlBody).
		SetReceiver(accountx).
		SaveX(ctx)
}
