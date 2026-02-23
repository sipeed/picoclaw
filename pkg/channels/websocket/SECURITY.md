# WebSocket Chat Security - XSS Protection

## Issue
XSS vulnerability via markdown rendering where LLM responses containing HTML could be rendered directly without sanitization.

## Solution
Implemented comprehensive XSS protection using DOMPurify library with strict configuration.

## Implementation Details

### 1. DOMPurify Integration
- **Library**: DOMPurify v3.0.8 (loaded from CDN)
- **Location**: `/pkg/channels/websocket/chat.html`

### 2. Security Configuration
```javascript
const DOMPURIFY_CONFIG = {
    ALLOWED_TAGS: [
        'h1', 'h2', 'h3', 'h4', 'h5', 'h6',
        'p', 'br', 'hr',
        'strong', 'em', 'b', 'i', 'u', 'del', 's', 'code', 'pre',
        'ul', 'ol', 'li',
        'blockquote',
        'a', 'img',
        'table', 'thead', 'tbody', 'tr', 'th', 'td',
        'span', 'div'
    ],
    ALLOWED_ATTR: [
        'href', 'title', 'alt', 'src',
        'class', 'id'
    ],
    ALLOW_DATA_ATTR: false,
    ALLOWED_URI_REGEXP: /^(?:(?:(?:f|ht)tps?|mailto|tel|callto|cid|xmpp|data):|[^a-z]|[a-z+.\-]+(?:[^a-z+.\-:]|$))/i,
    FORBID_TAGS: ['script', 'style', 'iframe', 'object', 'embed', 'form', 'input', 'button'],
    FORBID_ATTR: ['onerror', 'onload', 'onclick', 'onmouseover', 'onfocus', 'onblur'],
    KEEP_CONTENT: true
};
```

### 3. Security Features

#### Blocked Elements
- `<script>` tags - Prevents JavaScript execution
- `<style>` tags - Prevents CSS injection
- `<iframe>`, `<object>`, `<embed>` - Prevents content embedding attacks
- `<form>`, `<input>`, `<button>` - Prevents form injection

#### Blocked Attributes
All event handler attributes are blocked:
- `onerror`, `onload`, `onclick`
- `onmouseover`, `onfocus`, `onblur`
- And any other `on*` attributes

#### Allowed Safe Elements
Only safe HTML tags for markdown rendering:
- Headings: `h1-h6`
- Text formatting: `p`, `br`, `hr`, `strong`, `em`, `b`, `i`, `u`, `del`, `s`
- Code: `code`, `pre`
- Lists: `ul`, `ol`, `li`
- Quotes: `blockquote`
- Links and images: `a`, `img`
- Tables: `table`, `thead`, `tbody`, `tr`, `th`, `td`
- Containers: `span`, `div`

### 4. Implementation Points

All markdown rendering goes through the `sanitizeHtml()` function:

1. **Assistant Messages** (Line 1345-1348)
```javascript
if (sender === 'assistant' || sender === 'system') {
    const rawHtml = marked.parse(content);
    const cleanHtml = sanitizeHtml(rawHtml);
    contentDiv.innerHTML = cleanHtml;
}
```

2. **System Messages Update** (Line 1368-1371)
```javascript
function updateReconnectMessage(newContent) {
    const rawHtml = marked.parse(newContent);
    const cleanHtml = sanitizeHtml(rawHtml);
    contentDiv.innerHTML = cleanHtml;
}
```

3. **Retry Button Messages** (Line 1393-1396)
```javascript
const rawHtml = marked.parse(
    t.clickToRetry + '\n\n**[' + t.retryButton + ']**'
);
const cleanHtml = sanitizeHtml(rawHtml);
contentDiv.innerHTML = cleanHtml;
```

### 5. User Input Protection
User messages are rendered as plain text using `textContent` instead of `innerHTML`:
```javascript
} else {
    // 用户消息保持纯文本
    contentDiv.textContent = content;
}
```

## Testing

A test suite is provided in `xss_test.html` that validates protection against:
- Script tag injection
- Event handler injection (onclick, onerror, etc.)
- iframe/object/embed injection
- JavaScript protocol URLs
- SVG-based attacks
- Style injection
- Form injection
- And various bypass attempts

To run tests:
1. Open `xss_test.html` in a browser
2. All tests should pass
3. Check browser console for any JavaScript errors

## Attack Vectors Mitigated

✅ Direct script injection: `<script>alert('XSS')</script>`
✅ Image onerror: `<img src="x" onerror="alert('XSS')">`
✅ JavaScript URLs: `[link](javascript:alert('XSS'))`
✅ Event handlers: `<a onclick="alert('XSS')">Click</a>`
✅ SVG attacks: `<svg onload="alert('XSS')"></svg>`
✅ Iframe injection: `<iframe src="javascript:alert('XSS')"></iframe>`
✅ Object tags: `<object data="javascript:alert('XSS')"></object>`
✅ Form injection: `<form><input type="text"></form>`
✅ Style injection: `<style>body { display: none; }</style>`

## Maintenance

When updating the chat interface:
1. Always use `sanitizeHtml()` for any HTML content from external sources
2. Never use `innerHTML` directly with user/LLM content
3. Keep DOMPurify library updated to latest stable version
4. Review `DOMPURIFY_CONFIG` when adding new markdown features

## References

- [DOMPurify GitHub](https://github.com/cure53/DOMPurify)
- [OWASP XSS Prevention Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Cross_Site_Scripting_Prevention_Cheat_Sheet.html)
- [Marked.js Documentation](https://marked.js.org/)
