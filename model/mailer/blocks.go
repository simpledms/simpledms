package mailer

import (
	"fmt"

	"github.com/simpledms/simpledms/ctxx"
	wx "github.com/simpledms/simpledms/ui/widget"
)

// ContentBlock represents a block of content in the email
type ContentBlock interface {
	ToHTML(ctx ctxx.Context) string
	ToPlainText(ctx ctxx.Context) string
}

// *TextBlock is a simple text paragraph
type TextBlock struct {
	Text string
}

func (qq TextBlock) ToHTML(ctx ctxx.Context) string {
	return fmt.Sprintf("<p>%s</p>", qq.Text)
}

func (qq TextBlock) ToPlainText(ctx ctxx.Context) string {
	return qq.Text
}

// PasswordBlock displays a password in a styled box
type PasswordBlock struct {
	Password string
}

func NewPasswordBlock(password string) PasswordBlock {
	return PasswordBlock{
		Password: password,
	}
}

func (qq PasswordBlock) ToHTML(ctx ctxx.Context) string {
	return fmt.Sprintf(`<p>%s:</p>
<div class="password">%s</div>`, wx.T("Your temporary password is").String(ctx), qq.Password)
}

func (qq PasswordBlock) ToPlainText(ctx ctxx.Context) string {
	return fmt.Sprintf("%s: %s", wx.T("Your temporary password is").String(ctx), qq.Password)
}

// ExpiryBlock displays when the password expires
type ExpiryBlock struct {
	ExpiresAt string
}

func NewExpiryBlock(expiresAt string) ExpiryBlock {
	return ExpiryBlock{
		ExpiresAt: expiresAt,
	}
}

func (qq ExpiryBlock) ToHTML(ctx ctxx.Context) string {
	return fmt.Sprintf(`<p>%s: <strong>%s</strong></p>`, wx.T("It expires at").String(ctx), qq.ExpiresAt)
}

func (qq ExpiryBlock) ToPlainText(ctx ctxx.Context) string {
	return fmt.Sprintf("%s: %s", wx.T("It expires at").String(ctx), qq.ExpiresAt)
}

// NoteBlock displays a highlighted note
type NoteBlock struct {
	Text string
}

func (qq NoteBlock) ToHTML(ctx ctxx.Context) string {
	return fmt.Sprintf(`<div class="note">
    <p><strong>%s:</strong> %s</p>
</div>`, wx.T("Note").String(ctx), qq.Text)
}

func (qq NoteBlock) ToPlainText(ctx ctxx.Context) string {
	return fmt.Sprintf("%s: %s", wx.T("Note").String(ctx), qq.Text)
}
