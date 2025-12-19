package mailer

import (
	"fmt"
	"strings"

	"github.com/simpledms/simpledms/app/simpledms/ctxx"
)

// Common CSS styles for all email templates
const emailCSS = `
    body {
        font-family: Arial, sans-serif;
        line-height: 1.6;
        color: #333;
        max-width: 600px;
        margin: 0 auto;
        padding: 20px;
    }
    .container {
        background-color: #f9f9f9;
        border: 1px solid #ddd;
        border-radius: 5px;
        padding: 20px;
    }
    .header {
        text-align: center;
        margin-bottom: 20px;
    }
    .content {
        margin-bottom: 20px;
    }
    .password {
        background-color: #eee;
        padding: 10px;
        border-radius: 3px;
        font-family: monospace;
        font-size: 16px;
        text-align: center;
        margin: 15px 0;
    }
    .note {
        background-color: #fffde7;
        padding: 10px;
        border-left: 4px solid #ffd600;
        margin: 15px 0;
    }
    .footer {
        font-size: 12px;
        color: #777;
        text-align: center;
        margin-top: 30px;
        border-top: 1px solid #ddd;
        padding-top: 10px;
    }
`

// *EmailTemplate represents the structure of an HTML email
type EmailTemplate struct {
	Title   string
	Heading string
	Content []ContentBlock
	Footer  string
}

// RenderHTML generates the complete HTML email from the template
func (qq *EmailTemplate) RenderHTML(ctx ctxx.Context) string {
	var contentHTML strings.Builder
	for _, block := range qq.Content {
		contentHTML.WriteString(block.ToHTML(ctx))
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>%s</title>
    <style>%s</style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>%s</h1>
        </div>
        <div class="content">
            %s
        </div>
        <div class="footer">
            <p>%s</p>
        </div>
    </div>
</body>
</html>`, qq.Title, emailCSS, qq.Heading, contentHTML.String(), qq.Footer)
}

// RenderPlainText generates the plain text version of the email
func (qq *EmailTemplate) RenderPlainText(ctx ctxx.Context) string {
	var plainText strings.Builder

	// Add heading
	plainText.WriteString(qq.Heading)
	plainText.WriteString("\n\n")

	// Add content blocks
	for _, block := range qq.Content {
		plainText.WriteString(block.ToPlainText(ctx))
		plainText.WriteString("\n\n")
	}

	// Add footer
	plainText.WriteString(qq.Footer)

	return plainText.String()
}
