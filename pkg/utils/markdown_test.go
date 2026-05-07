package utils

import (
	"testing"

	"github.com/sipeed/picoclaw/pkg/logger"
)

func TestHtmlToMarkdown(t *testing.T) {
	// Define our test cases
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Removes scripts and styles",
			input:    `<script>alert("hello");</script><style>body { color: red; }</style><p>Clean text</p>`,
			expected: "Clean text",
		},
		{
			name:     "Extracts links correctly",
			input:    `Visit my <a href="https://example.com">website</a> for info.`,
			expected: "Visit my [website](https://example.com) for info.",
		},
		{
			name:     "Converts headers (H1, H2, H3)",
			input:    `<h1>Main Title</h1><h2>Subtitle</h2><h3>Section</h3>`,
			expected: "# Main Title\n\n## Subtitle\n\n### Section",
		},
		{
			name:     "Handles bold and italics",
			input:    `Text <b>bold</b> and <strong>strong</strong>, then <i>italic</i> and <em>em</em>.`,
			expected: "Text **bold** and **strong**, then *italic* and *em*.",
		},
		{
			name:     "Converts lists",
			input:    `<ul><li>First element</li><li>Second element</li></ul>`,
			expected: "- First element\n- Second element",
		},
		{
			name:     "Handles paragraphs and line breaks (<br>)",
			input:    `<p>First paragraph</p><p>Second paragraph with<br>a line break.</p>`,
			expected: "First paragraph\n\nSecond paragraph with\na line break.",
		},
		{
			name:     "Decodes HTML entities",
			input:    `Math: 5 &gt; 3 &amp; 2 &lt; 4. A &quot;quote&quot;.`,
			expected: "Math: 5 > 3 & 2 < 4. A \"quote\".",
		},
		{
			name:     "Cleans up residual HTML tags",
			input:    `<div><span>Text inside div and span</span></div>`,
			expected: "Text inside div and span",
		},
		{
			name:     "Removes multiple spaces and excessive empty lines",
			input:    `This   text    has too many spaces. <br><br><br><br> And too many newlines.`,
			expected: "This text has too many spaces.\n\nAnd too many newlines.",
		},
		{
			name:  "Nested lists with indentation",
			input: "<ul><li>One<ul><li>Two</li></ul></li></ul>",
			// Expect the sub-element to have 4 spaces of indentation
			expected: "- One\n    - Two",
		},
		{
			name:  "Image support",
			input: `<img src="image.jpg" alt="alternative text">`,
			// Correct Markdown syntax for images
			expected: "![alternative text](image.jpg)",
		},
		{
			name:  "Image support without alt-text",
			input: `<img src="image.jpg">`,
			// If alt is missing, square brackets remain empty
			expected: "![](image.jpg)",
		},
		{
			name: "XSS Bypass on Links (Obfuscated HTML entities)",
			// The Go HTML parser resolves entities, so this becomes "javascript:alert(1)"
			input: `<a href="jav&#x09;ascript:alert(1)">Click here</a>`,
			// Our isSafeHref (if updated with net/url) should neutralize it to "#"
			expected: "[Click here](#)",
		},
		{
			name:  "Empty link or used as anchor",
			input: `<a name="top"></a>`,
			// With no text or href, it shouldn't print anything (not even empty brackets)
			expected: "",
		},
		{
			name:  "Link without href but with text (Textual anchor)",
			input: `<a id="top">Back to top</a>`,
			// Should extract only plain text, without generating a broken Markdown link like [Back to top](#) or [Back to top]()
			expected: "Back to top",
		},
		{
			name:  "Badly spaced bold and italics (Edge Case)",
			input: `<b> Text </b>`,
			// In Markdown `** Text **` is often not formatted correctly. The ideal is `**Text**`
			expected: "**Text**",
		},
		{
			name: "Complex Test - Real Article",
			input: `
             <h1>Article Title</h1>
             <p>This is an <strong>introductory text</strong> with a <a href="http://link.com">link</a>.</p>
             <h2>Subtitle</h2>
             <ul>
                <li>Point one</li>
                <li>Point two</li>
             </ul>
             <script>console.log("do not show me")</script>
          `,
			// Note: The indentation of the real HTML test will generate spaces that
			// regex will clean up.
			expected: "# Article Title\n\nThis is an **introductory text** with a [link](http://link.com).\n\n## Subtitle\n\n- Point one\n- Point two",
		},
		{
			name:     "Ordered list (OL)",
			input:    `<ol><li>First</li><li>Second</li><li>Third</li></ol>`,
			expected: "1. First\n2. Second\n3. Third",
		},
		{
			name:     "Ordered list nested in unordered list",
			input:    `<ul><li>Fruits<ol><li>Apples</li><li>Pears</li></ol></li><li>Vegetables</li></ul>`,
			expected: "- Fruits\n    1. Apples\n    2. Pears\n- Vegetables",
		},
		{
			name:     "Code block (pre/code)",
			input:    "<pre><code>func main() {\n    fmt.Println(\"hello\")\n}</code></pre>",
			expected: "```\nfunc main() {\n    fmt.Println(\"hello\")\n}\n```",
		},
		{
			name:     "Inline code",
			input:    `<p>Use the command <code>go test ./...</code> to run the tests.</p>`,
			expected: "Use the command `go test ./...` to run the tests.",
		},
		{
			name:     "Simple blockquote",
			input:    `<blockquote><p>An important quote.</p></blockquote>`,
			expected: "> An important quote.",
		},
		{
			name:     "Multiline blockquote",
			input:    `<blockquote><p>First line of the quote.</p><p>Second line of the quote.</p></blockquote>`,
			expected: "> First line of the quote.\n>\n> Second line of the quote.",
		},
		{
			name:     "Strikethrough text (del/s)",
			input:    `This text is <del>deleted</del> and this is <s>crossed out</s>.`,
			expected: "This text is ~~deleted~~ and this is ~~crossed out~~.",
		},
		{
			name:     "Horizontal separator (HR)",
			input:    `<p>Above the line</p><hr><p>Below the line</p>`,
			expected: "Above the line\n\n---\n\nBelow the line",
		},
		{
			name:     "Bold nested in link",
			input:    `<a href="https://example.com"><strong>Linked bold text</strong></a>`,
			expected: "[**Linked bold text**](https://example.com)",
		},
		{
			name:     "data-src Image (lazy loading)",
			input:    `<img data-src="lazy.jpg" alt="Lazy image">`,
			expected: "![Lazy image](lazy.jpg)",
		},
		{
			name:  "Image with javascript: src blocked",
			input: `<img src="javascript:alert(1)" alt="XSS">`,
			// src is not safe, so the image is not emitted
			expected: "",
		},
		{
			name:     "Link with data: href blocked",
			input:    `<a href="data:text/html,<script>alert(1)</script>">Click</a>`,
			expected: "[Click](#)",
		},
		{
			name:     "Deeply nested divs",
			input:    `<div><div><div><div><p>Deeply nested text</p></div></div></div></div>`,
			expected: "Deeply nested text",
		},
		{
			name:     "Non-consecutive headers (H1, H3, H5)",
			input:    `<h1>Title</h1><h3>Subsection</h3><h5>Sub-subsection</h5>`,
			expected: "# Title\n\n### Subsection\n\n##### Sub-subsection",
		},
		{
			name:     "Paragraph with mixed multiple emphasis",
			input:    `<p><strong>Important:</strong> read the <strong><em>critical instructions</em></strong> <em>carefully</em>.</p>`,
			expected: "**Important:** read the ***critical instructions*** *carefully*.",
		},
		{
			name: "Article with nav and aside sections (noise to filter)",
			input: `
        <nav><a href="/home">Home</a><a href="/about-us">About us</a></nav>
        <article>
            <h2>Article title</h2>
            <p>This is the body of the article.</p>
        </article>
        <aside><p>Advertisement</p></aside>
       `,
			expected: "## Article title\n\nThis is the body of the article.",
		},
		{
			name:     "Text with mixed special HTML entities",
			input:    `Copyright &copy; 2024 &mdash; All rights reserved &reg;`,
			expected: "Copyright © 2024 — All rights reserved ®",
		},
		{
			name:     "Mailto link",
			input:    `Write to us at <a href="mailto:info@example.com">info@example.com</a>`,
			expected: "Write to us at [info@example.com](mailto:info@example.com)",
		},
		{
			name:  "Image inside a link (clickable figure)",
			input: `<a href="https://example.com"><img src="photo.jpg" alt="Photo"></a>`,
			// The image-link without text must not generate broken markup
			expected: "[![Photo](photo.jpg)](https://example.com)",
		},
		{
			name:     "Empty content or only whitespace",
			input:    `   <p>  </p>  <div>   </div>  `,
			expected: "",
		},
	}

	// Iterate over all test cases
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := HtmlToMarkdown(tt.input)
			if err != nil {
				logger.ErrorCF("tool", "Failed to parse html to markdown: %s", map[string]any{"error": err.Error()})
			}

			if got != tt.expected {
				t.Errorf("\nTest case failed: %s\nInput:    %q\nGot:      %q\nExpected: %q",
					tt.name, tt.input, got, tt.expected)
			}
		})
	}
}
