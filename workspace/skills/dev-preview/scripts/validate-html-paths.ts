/**
 * Validate rendered HTML from a dev server for absolute paths in src/href.
 *
 * Fetches the page from the running dev server and checks the actual DOM
 * output — works with any framework/template engine.
 *
 * Usage: bun run validate-html-paths.ts <url> [url2] ...
 *   e.g. bun run validate-html-paths.ts http://localhost:3000
 *        bun run validate-html-paths.ts http://localhost:3000 http://localhost:3000/about
 *
 * Checks:
 *  1. Absolute paths in src/href (breaks under /miniapp/dev/ reverse proxy)
 *  2. External domain URLs in src/href (blocked by Telegram Mini App sandbox)
 *
 * Exit code: 0 = pass, 1 = violations found, 2 = usage error / fetch failure
 */

const ATTRS = ["src", "href", "action", "poster", "data"] as const;

// Matches attr="value" or attr='value' in HTML tags
const TAG_ATTR_RE = new RegExp(
  `\\b(${ATTRS.join("|")})\\s*=\\s*(?:"([^"]*?)"|'([^']*?)')`,
  "gi",
);

type ViolationKind = "absolute-path" | "external-domain";

interface Violation {
  url: string;
  line: number;
  attr: string;
  value: string;
  kind: ViolationKind;
}

function classifyViolation(value: string): ViolationKind | null {
  // External URLs: https://..., http://..., //cdn.example.com/...
  if (/^(https?:)?\/\//.test(value)) return "external-domain";
  // Absolute path: /foo (but not /miniapp/dev/...)
  if (value.startsWith("/") && !value.startsWith("/miniapp/dev")) return "absolute-path";
  return null;
}

async function validate(url: string): Promise<Violation[]> {
  const resp = await fetch(url);
  if (!resp.ok) {
    throw new Error(`${url} returned ${resp.status}`);
  }
  const html = await resp.text();
  const lines = html.split("\n");
  const violations: Violation[] = [];

  for (let i = 0; i < lines.length; i++) {
    TAG_ATTR_RE.lastIndex = 0;
    let match: RegExpExecArray | null;
    while ((match = TAG_ATTR_RE.exec(lines[i])) !== null) {
      const value = match[2] ?? match[3];
      const kind = classifyViolation(value);
      if (kind) {
        violations.push({ url, line: i + 1, attr: match[1], value, kind });
      }
    }
  }
  return violations;
}

// --- main ---
const urls = process.argv.slice(2);
if (urls.length === 0) {
  console.error("Usage: bun run validate-html-paths.ts <url> [url2] ...");
  console.error("  e.g. bun run validate-html-paths.ts http://localhost:3000");
  process.exit(2);
}

let total = 0;
for (const url of urls) {
  try {
    const vs = await validate(url);
    for (const v of vs) {
      if (v.kind === "absolute-path") {
        console.error(
          `ERROR ${v.url} line ${v.line}: ${v.attr}="${v.value}" is absolute.`,
        );
        console.error(
          `  FIX: Change to ${v.attr}=".${v.value}"`,
        );
      } else {
        console.error(
          `ERROR ${v.url} line ${v.line}: ${v.attr}="${v.value}" loads from external domain.`,
        );
        console.error(
          `  FIX: Telegram Mini App blocks cross-origin loading. Download and serve locally, bundle with the app, or inline the content.`,
        );
      }
    }
    total += vs.length;
  } catch (e: any) {
    console.error(`FETCH ERROR: ${e.message}`);
    process.exit(2);
  }
}

if (total > 0) {
  console.error(
    `\n${total} violation(s). Fix absolute paths (use ./) and remove external domain URLs.`,
  );
  process.exit(1);
} else {
  console.log(`OK — ${urls.length} URL(s) checked, no absolute path violations.`);
}
