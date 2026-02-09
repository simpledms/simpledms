package staticpage

import (
	"bytes"
	"embed"
	"html/template"
	"io/fs"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

type MarkdownRenderer struct {
	contentFS fs.FS
	markdown  goldmark.Markdown
}

func NewMarkdownRenderer(contentFS embed.FS) *MarkdownRenderer {
	return &MarkdownRenderer{
		contentFS: contentFS,
		markdown: goldmark.New(
			goldmark.WithExtensions(
				extension.GFM,
				extension.Linkify,
			),
			goldmark.WithParserOptions(
				parser.WithAutoHeadingID(),
			),
		),
	}
}

var defaultMarkdownRenderer = NewMarkdownRenderer(contentFS)

func RenderMarkdown(path string) (template.HTML, error) {
	return defaultMarkdownRenderer.Render(path)
}

func (qq *MarkdownRenderer) Render(path string) (template.HTML, error) {
	markdownContent, err := fs.ReadFile(qq.contentFS, path)
	if err != nil {
		return "", err
	}

	var buffer bytes.Buffer
	if err = qq.markdown.Convert(markdownContent, &buffer); err != nil {
		return "", err
	}

	htmlContent, err := qq.styleMarkdown(buffer.String())
	if err != nil {
		return "", err
	}

	return template.HTML(htmlContent), nil
}

func (qq *MarkdownRenderer) styleMarkdown(raw string) (string, error) {
	container := &html.Node{Type: html.ElementNode, DataAtom: atom.Div, Data: "div"}

	nodes, err := html.ParseFragment(strings.NewReader(raw), container)
	if err != nil {
		return "", err
	}

	for _, node := range nodes {
		container.AppendChild(node)
	}

	qq.visitNodes(container, func(node *html.Node) {
		if node.Type != html.ElementNode {
			return
		}

		switch node.Data {
		case "h1":
			qq.appendAttr(node, "class", "headline-md text-on-surface mt-7 mb-3")
		case "h2":
			qq.appendAttr(node, "class", "headline-sm text-on-surface mt-6 mb-3")
		case "h3", "h4", "h5", "h6":
			qq.appendAttr(node, "class", "title-lg text-on-surface mt-5 mb-2")
		case "p":
			qq.appendAttr(node, "class", "text-on-surface my-2")
		case "ul":
			qq.appendAttr(node, "class", "text-on-surface list-disc ml-6 my-3")
		case "ol":
			qq.appendAttr(node, "class", "text-on-surface list-decimal ml-6 my-3")
		case "li":
			qq.appendAttr(node, "class", "my-1")
		case "a":
			qq.appendAttr(node, "class", "text-tertiary")
			qq.appendAttr(node, "class", "underline")
			href := qq.getAttr(node, "href")
			if strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://") {
				qq.setAttr(node, "target", "_blank")
				qq.setAttr(node, "rel", "noopener noreferrer")
			}
		case "pre":
			qq.appendAttr(node, "class", "body-sm my-3 p-3 rounded-xl overflow-x-auto bg-surface-container-low")
		case "code":
			if node.Parent != nil && node.Parent.Data == "pre" {
				qq.appendAttr(node, "class", "body-sm")
				break
			}

			qq.appendAttr(node, "class", "body-sm px-1 py-[2px] rounded bg-surface-container-high")
		case "blockquote":
			qq.appendAttr(node, "class", "text-on-surface my-3 py-1 pl-4 border-l-4 border-outline-variant")
		case "hr":
			qq.appendAttr(node, "class", "my-4 border-t border-outline-variant")
		}
	})

	var out bytes.Buffer
	for child := container.FirstChild; child != nil; child = child.NextSibling {
		if err = html.Render(&out, child); err != nil {
			return "", err
		}
	}

	return out.String(), nil
}

func (qq *MarkdownRenderer) visitNodes(node *html.Node, fn func(*html.Node)) {
	fn(node)
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		qq.visitNodes(child, fn)
	}
}

func (qq *MarkdownRenderer) appendAttr(node *html.Node, key, value string) {
	currentValue := strings.TrimSpace(qq.getAttr(node, key))
	if currentValue == "" {
		qq.setAttr(node, key, value)
		return
	}

	qq.setAttr(node, key, currentValue+" "+value)
}

func (qq *MarkdownRenderer) getAttr(node *html.Node, key string) string {
	for _, attr := range node.Attr {
		if attr.Key == key {
			return attr.Val
		}
	}

	return ""
}

func (qq *MarkdownRenderer) setAttr(node *html.Node, key, value string) {
	for i, attr := range node.Attr {
		if attr.Key == key {
			node.Attr[i].Val = value
			return
		}
	}

	node.Attr = append(node.Attr, html.Attribute{Key: key, Val: value})
}
