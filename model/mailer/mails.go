package mailer

import (
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/model/account"
	"github.com/simpledms/simpledms/model/tenant"
	wx "github.com/simpledms/simpledms/ui/widget"
)

// CreateSignUpTemplate creates a template for the signup email
func CreateSignUpTemplate(ctx ctxx.Context, tmpPassword string, expiresAt string) EmailTemplate {
	title := wx.T("Welcome to SimpleDMS").String(ctx)
	heading := title
	footer := wx.T("This is an automated message, please do not reply.").String(ctx)

	content := []ContentBlock{
		TextBlock{Text: wx.T("Your account has been created successfully.").String(ctx)},
		NewPasswordBlock(tmpPassword),
		NewExpiryBlock(expiresAt),
		TextBlock{Text: wx.T("Please log in and change your password as soon as possible.").String(ctx)},
	}

	return EmailTemplate{
		Title:   title,
		Heading: heading,
		Content: content,
		Footer:  footer,
	}
}

// CreateResetPasswordTemplate creates a template for the password reset email
func CreateResetPasswordTemplate(ctx ctxx.Context, tmpPassword string, expiresAt string) EmailTemplate {
	title := wx.T("SimpleDMS Password Reset").String(ctx)
	heading := title
	footer := wx.T("This is an automated message, please do not reply.").String(ctx)

	content := []ContentBlock{
		TextBlock{Text: wx.T("A password reset has been requested for your account.").String(ctx)},
		PasswordBlock{Password: tmpPassword},
		ExpiryBlock{ExpiresAt: expiresAt},
		NoteBlock{Text: wx.T("Your old password will still work until you change it.").String(ctx)},
		TextBlock{Text: wx.T("Please log in and change your password as soon as possible.").String(ctx)},
	}

	return EmailTemplate{
		Title:   title,
		Heading: heading,
		Content: content,
		Footer:  footer,
	}
}

func CreateUserTemplate(ctx ctxx.Context, tmpPassword string, expiresAt string) EmailTemplate {
	title := wx.T("Welcome to SimpleDMS").String(ctx)
	heading := title
	footer := wx.T("This is an automated message, please do not reply.").String(ctx)

	accountm := account.NewAccount(ctx.MainCtx().Account)

	content := []ContentBlock{}

	if ctx.IsTenantCtx() {
		content = append(content,
			TextBlock{Text: wx.Tf(
				"«%s» invited you to the tenant «%s».",
				accountm.Name(),
				tenant.NewTenant(ctx.TenantCtx().Tenant).Name(),
			).String(ctx)},
		)
	} else {
		content = append(content,
			TextBlock{Text: wx.Tf(
				"«%s» invited you.",
				accountm.Name(),
			).String(ctx)},
		)
	}

	/*
		if customMessage != "" {
			content = append(content, TextBlock{Text: customMessage})
		}
	*/

	content = append(content,
		NewPasswordBlock(tmpPassword),
		NewExpiryBlock(expiresAt),
		TextBlock{Text: wx.T("Please log in and change your password as soon as possible.").String(ctx)},
		// TODO hint if not signed up yourself, report as abuse...
	)

	return EmailTemplate{
		Title:   title,
		Heading: heading,
		Content: content,
		Footer:  footer,
	}
}
