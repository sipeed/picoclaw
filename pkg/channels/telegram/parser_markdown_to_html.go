package telegram

import (
	"fmt"
	"html"
	"regexp"
	"strings"
)

var reRawURL = regexp.MustCompile(`https?://[^\s<]+`)

func markdownToTelegramHTML(text string) string {
	if text == "" {
		return ""
	}

	codeBlocks := extractCodeBlocks(text)
	text = codeBlocks.text

	inlineCodes := extractInlineCodes(text)
	text = inlineCodes.text

	links := extractLinks(text)
	text = links.text

	rawURLs := extractRawURLs(text)
	text = rawURLs.text

	text = reHeading.ReplaceAllString(text, "$1")

	text = reBlockquote.ReplaceAllString(text, "$1")

	text = escapeHTML(text)

	text = reBoldStar.ReplaceAllString(text, "<b>$1</b>")

	text = reBoldUnder.ReplaceAllString(text, "<b>$1</b>")

	text = reItalic.ReplaceAllStringFunc(text, func(s string) string {
		match := reItalic.FindStringSubmatch(s)
		if len(match) < 2 {
			return s
		}
		return "<i>" + match[1] + "</i>"
	})

	text = reStrike.ReplaceAllString(text, "<s>$1</s>")

	text = reListItem.ReplaceAllString(text, "• ")

	for i, lnk := range links.links {
		label := escapeHTML(lnk[0])
		url := escapeHTMLAttr(lnk[1])
		text = strings.ReplaceAll(text, fmt.Sprintf("\x00LK%d\x00", i), fmt.Sprintf(`<a href="%s">%s</a>`, url, label))
	}

	for i, rawURL := range rawURLs.urls {
		escaped := escapeHTML(rawURL)
		text = strings.ReplaceAll(
			text,
			fmt.Sprintf("\x00RU%d\x00", i),
			fmt.Sprintf(`<a href="%s">%s</a>`, escapeHTMLAttr(rawURL), escaped),
		)
	}

	for i, code := range inlineCodes.codes {
		escaped := escapeHTML(code)
		text = strings.ReplaceAll(text, fmt.Sprintf("\x00IC%d\x00", i), fmt.Sprintf("<code>%s</code>", escaped))
	}

	for i, code := range codeBlocks.codes {
		escaped := escapeHTML(code)
		text = strings.ReplaceAll(
			text,
			fmt.Sprintf("\x00CB%d\x00", i),
			fmt.Sprintf("<pre><code>%s</code></pre>", escaped),
		)
	}

	return text
}

type linkMatch struct {
	text  string
	links [][2]string // [label, url]
}

func extractLinks(text string) linkMatch {
	matches := reLink.FindAllStringSubmatch(text, -1)

	extracted := make([][2]string, 0, len(matches))
	for _, match := range matches {
		extracted = append(extracted, [2]string{match[1], match[2]})
	}

	i := 0
	text = reLink.ReplaceAllStringFunc(text, func(m string) string {
		placeholder := fmt.Sprintf("\x00LK%d\x00", i)
		i++
		return placeholder
	})

	return linkMatch{text: text, links: extracted}
}

type codeBlockMatch struct {
	text  string
	codes []string
}

type rawURLMatch struct {
	text string
	urls []string
}

func extractCodeBlocks(text string) codeBlockMatch {
	matches := reCodeBlock.FindAllStringSubmatch(text, -1)

	codes := make([]string, 0, len(matches))
	for _, match := range matches {
		codes = append(codes, match[1])
	}

	i := 0
	text = reCodeBlock.ReplaceAllStringFunc(text, func(m string) string {
		placeholder := fmt.Sprintf("\x00CB%d\x00", i)
		i++
		return placeholder
	})

	return codeBlockMatch{text: text, codes: codes}
}

func extractRawURLs(text string) rawURLMatch {
	matches := reRawURL.FindAllString(text, -1)

	urls := make([]string, 0, len(matches))
	for _, match := range matches {
		urls = append(urls, match)
	}

	i := 0
	text = reRawURL.ReplaceAllStringFunc(text, func(string) string {
		placeholder := fmt.Sprintf("\x00RU%d\x00", i)
		i++
		return placeholder
	})

	return rawURLMatch{text: text, urls: urls}
}

type inlineCodeMatch struct {
	text  string
	codes []string
}

func extractInlineCodes(text string) inlineCodeMatch {
	matches := reInlineCode.FindAllStringSubmatch(text, -1)

	codes := make([]string, 0, len(matches))
	for _, match := range matches {
		codes = append(codes, match[1])
	}

	i := 0
	text = reInlineCode.ReplaceAllStringFunc(text, func(m string) string {
		placeholder := fmt.Sprintf("\x00IC%d\x00", i)
		i++
		return placeholder
	})

	return inlineCodeMatch{text: text, codes: codes}
}

func escapeHTML(text string) string {
	text = strings.ReplaceAll(text, "&", "&amp;")
	text = strings.ReplaceAll(text, "<", "&lt;")
	text = strings.ReplaceAll(text, ">", "&gt;")
	return text
}

func escapeHTMLAttr(text string) string {
	return html.EscapeString(text)
}
