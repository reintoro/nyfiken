// Package strip containts functions to remove false positives from comparisons
// of new and last scrape.
//
// Example: Number of posts or number of comments are very commonly changed.
// A solution is to compare the requests while ignoring numbers.
// This package seeks to solve these kind of problems.
package strip

import "strings"
import "unicode"

import "code.google.com/p/go.net/html"
import "github.com/karlek/nyfiken/settings"
import "github.com/mewkiz/pkg/htmlutil"

// Returns a number free string.
func Numbers(doc *html.Node) (newSel string) {
	var f func(*html.Node)
	f = func(node *html.Node) {
		if node.Type == html.TextNode {
			text := strings.TrimSpace(node.Data)
			var newSel string
			for _, chr := range text {
				if !unicode.IsDigit(chr) {
					newSel += string(chr)
				}
			}
			node.Data = newSel
		}

		for c := node.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)

	return htmlutil.RenderToString(doc)
}

// Returns a string with empty HTML attributes.
func Attrs(doc *html.Node) (newSel string) {
	var f func(*html.Node)
	f = func(node *html.Node) {
		if node.Type == html.ElementNode {
			node.Attr = nil
		}

		for c := node.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)

	return htmlutil.RenderToString(doc)
}

// Returns a HTML free string.
func HTML(doc *html.Node) (newSel string) {
	var f func(*html.Node, *string)
	f = func(node *html.Node, newSel *string) {
		if node.Type == html.TextNode {
			*newSel += strings.TrimSpace(node.Data) + settings.Global.Newline
		}

		for c := node.FirstChild; c != nil; c = c.NextSibling {
			f(c, newSel)
		}
	}
	f(doc, &newSel)

	return newSel
}