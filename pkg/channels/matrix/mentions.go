package matrix

import (
	"html"
	"net/url"
	"regexp"
	"strings"

	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
)

var matrixMentionHrefRegexp = regexp.MustCompile(`(?i)<a[^>]+href=["']([^"']+)["']`)

func (c *MatrixChannel) isBotMentioned(msgEvt *event.MessageEventContent) bool {
	if msgEvt == nil {
		return false
	}

	if msgEvt.Mentions != nil && msgEvt.Mentions.Has(c.client.UserID) {
		return true
	}

	userID := c.client.UserID.String()
	if userID != "" && strings.Contains(msgEvt.Body, userID) {
		return true
	}
	if mentionsUserInFormattedBody(msgEvt.FormattedBody, c.client.UserID) {
		return true
	}

	mentionR := c.localpartMentionR
	if mentionR == nil {
		mentionR = localpartMentionRegexp(matrixLocalpart(c.client.UserID))
	}
	if mentionR == nil {
		return false
	}

	// Matrix users are addressed as MXID "@localpart:server", but many clients
	// emit plain-text mentions as "@localpart". Both forms are handled here.
	return mentionR.MatchString(msgEvt.Body) || mentionR.MatchString(msgEvt.FormattedBody)
}

func mentionsUserInFormattedBody(formattedBody string, userID id.UserID) bool {
	target := strings.ToLower(strings.TrimSpace(userID.String()))
	if target == "" {
		return false
	}

	formattedBody = strings.TrimSpace(formattedBody)
	if formattedBody == "" {
		return false
	}

	if strings.Contains(strings.ToLower(formattedBody), target) {
		return true
	}

	matches := matrixMentionHrefRegexp.FindAllStringSubmatch(formattedBody, -1)
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		decoded := decodeMatrixMentionHref(match[1])
		if strings.Contains(strings.ToLower(decoded), target) {
			return true
		}

		u, err := url.Parse(decoded)
		if err != nil {
			continue
		}

		if strings.Contains(strings.ToLower(u.Path), target) || strings.Contains(strings.ToLower(u.Fragment), target) {
			return true
		}
		if strings.Contains(strings.ToLower(decodeMatrixMentionHref(u.Fragment)), target) {
			return true
		}
	}

	return false
}

func decodeMatrixMentionHref(v string) string {
	decoded := html.UnescapeString(strings.TrimSpace(v))
	if decoded == "" {
		return ""
	}

	for i := 0; i < 2; i++ {
		next, err := url.QueryUnescape(decoded)
		if err != nil || next == decoded {
			break
		}
		decoded = next
	}
	return decoded
}

func (c *MatrixChannel) stripSelfMention(text string) string {
	return stripUserMentionWithRegexp(text, c.client.UserID, c.localpartMentionR)
}

func matrixLocalpart(userID id.UserID) string {
	s := strings.TrimPrefix(userID.String(), "@")
	localpart, _, _ := strings.Cut(s, ":")
	return strings.TrimSpace(localpart)
}

func localpartMentionRegexp(localpart string) *regexp.Regexp {
	localpart = strings.TrimSpace(localpart)
	if localpart == "" {
		return nil
	}

	// Match Matrix mentions in plain text while avoiding false positives:
	//   "@picoclaw" and "@picoclaw:matrix.org" should match,
	//   "test@example.com" and "hellopicoclawworld" should not.
	pattern := `(?i)(^|[^[:alnum:]_])@` + regexp.QuoteMeta(localpart) + `(?::[A-Za-z0-9._:-]+)?([^[:alnum:]_]|$)`
	return regexp.MustCompile(pattern)
}

func stripUserMention(text string, userID id.UserID) string {
	return stripUserMentionWithRegexp(text, userID, localpartMentionRegexp(matrixLocalpart(userID)))
}

func stripUserMentionWithRegexp(text string, userID id.UserID, mentionR *regexp.Regexp) string {
	cleaned := strings.ReplaceAll(text, userID.String(), "")

	if mentionR != nil {
		cleaned = mentionR.ReplaceAllString(cleaned, "$1$2")
	}

	cleaned = strings.TrimSpace(cleaned)
	cleaned = strings.TrimLeft(cleaned, ",:; ")
	return strings.TrimSpace(cleaned)
}
