import "./style.css";

const app = document.querySelector<HTMLDivElement>("#app");
if (!app) {
	throw new Error("Missing #app element");
}

const appEl = app;

const THEME_STORAGE_KEY = "picoclaw.theme";

type ThemeMode = "system" | "light" | "dark";

function getStoredTheme(): ThemeMode {
	const v = localStorage.getItem(THEME_STORAGE_KEY);
	if (v === "light" || v === "dark" || v === "system") return v;
	return "system";
}

function applyTheme(mode: ThemeMode): void {
	const root = document.documentElement;
	if (mode === "system") {
		root.removeAttribute("data-theme");
	} else {
		root.setAttribute("data-theme", mode);
	}
}

let themeMode: ThemeMode = getStoredTheme();
applyTheme(themeMode);

const ADMIN_TOKEN_STORAGE_KEY = "picoclaw.admin_token";

type JSONSchema = {
	$ref?: string;
	title?: string;
	description?: string;
	type?: unknown;
	properties?: Record<string, JSONSchema>;
	required?: string[];
	items?: JSONSchema;
	enum?: unknown[];
	default?: unknown;
	examples?: unknown[];
	anyOf?: JSONSchema[];
	allOf?: JSONSchema[];
	oneOf?: JSONSchema[];
	$defs?: Record<string, JSONSchema>;
};

function isRecord(v: unknown): v is Record<string, unknown> {
	return typeof v === "object" && v !== null && !Array.isArray(v);
}

function resolveRef(root: JSONSchema, ref: string): JSONSchema | null {
	if (!ref.startsWith("#/$defs/")) return null;
	const name = ref.slice("#/$defs/".length);
	const defs = root.$defs;
	if (!defs) return null;
	return defs[name] ?? null;
}

function mergeSchemas(a: JSONSchema, b: JSONSchema): JSONSchema {
	return {
		...a,
		...b,
		properties: {
			...(a.properties ?? {}),
			...(b.properties ?? {}),
		},
		required: Array.from(
			new Set([...(a.required ?? []), ...(b.required ?? [])]),
		),
	};
}

function normalizeSchema(root: JSONSchema, schema: JSONSchema): JSONSchema {
	let s = schema;
	if (s.$ref) {
		const r = resolveRef(root, s.$ref);
		if (r) s = mergeSchemas(r, { ...s, $ref: undefined });
	}
	if (s.allOf && s.allOf.length > 0) {
		let out: JSONSchema = { ...s, allOf: undefined };
		for (const part of s.allOf)
			out = mergeSchemas(out, normalizeSchema(root, part));
		s = out;
	}
	return s;
}

function getSchemaType(schema: JSONSchema): string | null {
	const t = schema.type;
	if (typeof t === "string") return t;
	return null;
}

function deepGet(obj: unknown, path: string[]): unknown {
	let cur: unknown = obj;
	for (const p of path) {
		if (!isRecord(cur)) return undefined;
		cur = cur[p];
	}
	return cur;
}

function deepSet(obj: unknown, path: string[], value: unknown): void {
	if (!isRecord(obj)) return;
	let cur: Record<string, unknown> = obj;
	for (let i = 0; i < path.length - 1; i++) {
		const p = path[i];
		const next = cur[p];
		if (!isRecord(next)) {
			cur[p] = {};
		}
		cur = cur[p] as Record<string, unknown>;
	}
	cur[path[path.length - 1]] = value;
}

function renderAdminUI(): void {
	appEl.innerHTML = `
		<div class="wrap">
			<header class="header">
				<div class="title">PicoClaw Admin</div>
				<div class="header-right">
					<label class="theme" for="theme">
						<span class="theme-label">Theme</span>
						<select class="theme-select" id="theme">
							<option value="system">System</option>
							<option value="light">Light</option>
							<option value="dark">Dark</option>
						</select>
					</label>
					<div class="status" id="status">idle</div>
				</div>
			</header>

			<main class="admin" id="admin">
				<section class="panel">
					<div class="panel-title">Connection</div>
					<div class="grid">
						<label class="field">
							<div class="field-label">Admin token</div>
							<input class="input" id="adminToken" placeholder="gateway.admin_token" autocomplete="off" />
							<div class="field-help">Used as Authorization: Bearer &lt;token&gt; for /admin/* endpoints.</div>
						</label>
						<div class="row">
							<button class="btn" id="load">Load</button>
							<button class="btn" id="save">Save</button>
						</div>
					</div>
				</section>

				<section class="panel">
					<div class="panel-title">Mode</div>
					<div class="row">
						<label class="radio"><input type="radio" name="mode" value="form" checked /> Form</label>
						<label class="radio"><input type="radio" name="mode" value="json" /> Raw JSON</label>
					</div>
				</section>

				<section class="panel" id="jsonPanel" style="display:none">
					<div class="panel-title">Config JSON</div>
					<textarea class="textarea" id="raw"></textarea>
				</section>

				<section class="panel" id="formPanel">
					<div class="panel-title">Config Form</div>
					<div class="form" id="formRoot"></div>
				</section>
			</main>
		</div>
	`;

	const statusEl = document.querySelector<HTMLDivElement>("#status");
	const themeSelect = document.querySelector<HTMLSelectElement>("#theme");
	const adminTokenInput =
		document.querySelector<HTMLInputElement>("#adminToken");
	const loadBtn = document.querySelector<HTMLButtonElement>("#load");
	const saveBtn = document.querySelector<HTMLButtonElement>("#save");
	const rawTextarea = document.querySelector<HTMLTextAreaElement>("#raw");
	const modeInputs =
		document.querySelectorAll<HTMLInputElement>("input[name=mode]");
	const jsonPanel = document.querySelector<HTMLDivElement>("#jsonPanel");
	const formPanel = document.querySelector<HTMLDivElement>("#formPanel");
	const formRoot = document.querySelector<HTMLDivElement>("#formRoot");

	if (
		!statusEl ||
		!themeSelect ||
		!adminTokenInput ||
		!loadBtn ||
		!saveBtn ||
		!rawTextarea ||
		!jsonPanel ||
		!formPanel ||
		!formRoot
	) {
		throw new Error("Missing admin UI elements");
	}

	const statusDiv = statusEl;
	const themeSelectEl = themeSelect;
	const adminTokenInputEl = adminTokenInput;
	const loadBtnEl = loadBtn;
	const saveBtnEl = saveBtn;
	const rawTextareaEl = rawTextarea;
	const jsonPanelEl = jsonPanel;
	const formPanelEl = formPanel;
	const formRootEl = formRoot;
	const setStatus = (s: string): void => {
		statusDiv.textContent = s;
		statusDiv.dataset.state = s;
	};

	themeSelectEl.value = themeMode;
	themeSelectEl.addEventListener("change", () => {
		const v = themeSelectEl.value;
		if (v === "light" || v === "dark" || v === "system") {
			themeMode = v;
		} else {
			themeMode = "system";
		}
		localStorage.setItem(THEME_STORAGE_KEY, themeMode);
		applyTheme(themeMode);
	});

	adminTokenInputEl.value = localStorage.getItem(ADMIN_TOKEN_STORAGE_KEY) || "";
	adminTokenInputEl.addEventListener("input", () => {
		localStorage.setItem(ADMIN_TOKEN_STORAGE_KEY, adminTokenInputEl.value);
	});

	let schemaRoot: JSONSchema | null = null;
	let configObj: unknown = null;

	function authHeaders(): HeadersInit {
		const token = adminTokenInputEl.value.trim();
		return token ? { Authorization: `Bearer ${token}` } : {};
	}

	async function loadAll(): Promise<void> {
		setStatus("loading");
		try {
			const [schemaResp, cfgResp] = await Promise.all([
				fetch("/admin/schema", { headers: authHeaders() }),
				fetch("/admin/config", { headers: authHeaders() }),
			]);
			if (!schemaResp.ok)
				throw new Error(
					`schema: ${schemaResp.status} ${schemaResp.statusText}`,
				);
			if (!cfgResp.ok)
				throw new Error(`config: ${cfgResp.status} ${cfgResp.statusText}`);
			schemaRoot = (await schemaResp.json()) as JSONSchema;
			configObj = (await cfgResp.json()) as unknown;
			rawTextareaEl.value = JSON.stringify(configObj, null, 2);
			renderForm();
			setStatus("loaded");
		} catch (e) {
			setStatus(e instanceof Error ? e.message : "load failed");
		}
	}

	async function saveAll(): Promise<void> {
		setStatus("saving");
		try {
			const raw = rawTextareaEl.value.trim();
			const body =
				raw.length > 0 ? raw : JSON.stringify(configObj ?? {}, null, 2);
			const resp = await fetch("/admin/config", {
				method: "PUT",
				headers: {
					...authHeaders(),
					"Content-Type": "application/json",
				},
				body,
			});
			if (!resp.ok) {
				const text = await resp.text().catch(() => "");
				throw new Error(text || `${resp.status} ${resp.statusText}`);
			}
			setStatus("saved");
		} catch (e) {
			setStatus(e instanceof Error ? e.message : "save failed");
		}
	}

	function renderForm(): void {
		if (!schemaRoot || !configObj) {
			formRootEl.innerHTML = "";
			return;
		}
		const root = normalizeSchema(schemaRoot, schemaRoot);
		formRootEl.innerHTML = "";
		const node = renderSchemaNode(schemaRoot, root, [], configObj);
		formRootEl.appendChild(node);
	}

	function schemaLabel(schema: JSONSchema, key: string | null): string {
		if (typeof schema.title === "string" && schema.title.trim() !== "")
			return schema.title;
		if (key) return key;
		return "value";
	}

	function schemaHelp(schema: JSONSchema): string {
		const parts: string[] = [];
		if (
			typeof schema.description === "string" &&
			schema.description.trim() !== ""
		) {
			parts.push(schema.description.trim());
		}
		if (schema.default !== undefined) {
			parts.push(`default: ${JSON.stringify(schema.default)}`);
		}
		if (Array.isArray(schema.examples) && schema.examples.length > 0) {
			parts.push(`example: ${JSON.stringify(schema.examples[0])}`);
		}
		return parts.join(" · ");
	}

	function renderSchemaNode(
		schemaRoot0: JSONSchema,
		schema0: JSONSchema,
		path: string[],
		obj: unknown,
		key: string | null = null,
	): HTMLElement {
		const schema = normalizeSchema(schemaRoot0, schema0);
		const t = getSchemaType(schema);
		const label = schemaLabel(schema, key);
		const help = schemaHelp(schema);

		const wrap = document.createElement("div");
		wrap.className = "node";

		const header = document.createElement("div");
		header.className = "node-header";
		header.textContent = label;
		wrap.appendChild(header);

		if (help) {
			const helpEl = document.createElement("div");
			helpEl.className = "node-help";
			helpEl.textContent = help;
			wrap.appendChild(helpEl);
		}

		if (schema.enum && Array.isArray(schema.enum)) {
			const select = document.createElement("select");
			select.className = "input";
			for (const v of schema.enum) {
				const opt = document.createElement("option");
				opt.value = String(v);
				opt.textContent = String(v);
				select.appendChild(opt);
			}
			const cur = deepGet(obj, path);
			if (cur !== undefined) select.value = String(cur);
			select.addEventListener("change", () => {
				deepSet(obj, path, select.value);
				rawTextareaEl.value = JSON.stringify(configObj ?? {}, null, 2);
			});
			wrap.appendChild(select);
			return wrap;
		}

		if (t === "object" && schema.properties) {
			const req = new Set(schema.required ?? []);
			const props = schema.properties;
			const keys = Object.keys(props);
			keys.sort();
			for (const k of keys) {
				const childSchema = props[k];
				const row = document.createElement("div");
				row.className = "row";

				const child = renderSchemaNode(
					schemaRoot0,
					childSchema,
					[...path, k],
					obj,
					k,
				);
				if (req.has(k)) {
					child.classList.add("required");
				}
				row.appendChild(child);
				wrap.appendChild(row);
			}
			return wrap;
		}

		if (t === "boolean") {
			const input = document.createElement("input");
			input.type = "checkbox";
			input.className = "checkbox";
			const cur = deepGet(obj, path);
			input.checked = Boolean(cur ?? schema.default ?? false);
			input.addEventListener("change", () => {
				deepSet(obj, path, input.checked);
				rawTextareaEl.value = JSON.stringify(configObj ?? {}, null, 2);
			});
			wrap.appendChild(input);
			return wrap;
		}

		if (t === "number" || t === "integer") {
			const input = document.createElement("input");
			input.type = "number";
			input.className = "input";
			if (t === "integer") input.step = "1";
			const cur = deepGet(obj, path);
			if (typeof cur === "number") input.value = String(cur);
			else if (typeof schema.default === "number")
				input.value = String(schema.default);
			input.addEventListener("input", () => {
				const v = input.value.trim();
				if (v === "") {
					deepSet(obj, path, null);
				} else {
					const n = Number(v);
					deepSet(obj, path, Number.isFinite(n) ? n : null);
				}
				rawTextareaEl.value = JSON.stringify(configObj ?? {}, null, 2);
			});
			wrap.appendChild(input);
			return wrap;
		}

		if (t === "array" && schema.items) {
			const textarea = document.createElement("textarea");
			textarea.className = "textarea";
			const cur = deepGet(obj, path);
			textarea.value = JSON.stringify(cur ?? schema.default ?? [], null, 2);
			textarea.addEventListener("input", () => {
				try {
					const v = JSON.parse(textarea.value);
					deepSet(obj, path, v);
					rawTextareaEl.value = JSON.stringify(configObj ?? {}, null, 2);
				} catch {
					// ignore
				}
			});
			wrap.appendChild(textarea);
			return wrap;
		}

		// string or unknown: simple text input
		const input = document.createElement("input");
		input.type = "text";
		input.className = "input";
		const cur = deepGet(obj, path);
		if (typeof cur === "string") input.value = cur;
		else if (typeof schema.default === "string") input.value = schema.default;
		input.addEventListener("input", () => {
			deepSet(obj, path, input.value);
			rawTextareaEl.value = JSON.stringify(configObj ?? {}, null, 2);
		});
		wrap.appendChild(input);
		return wrap;
	}

	function switchMode(mode: "form" | "json"): void {
		if (mode === "json") {
			jsonPanelEl.style.display = "block";
			formPanelEl.style.display = "none";
		} else {
			jsonPanelEl.style.display = "none";
			formPanelEl.style.display = "block";
			// Try to parse the JSON editor back into config when switching to form.
			try {
				configObj = JSON.parse(rawTextareaEl.value);
				renderForm();
			} catch {
				// ignore
			}
		}
	}

	modeInputs.forEach((i) => {
		i.addEventListener("change", () => {
			const v = i.value === "json" ? "json" : "form";
			switchMode(v);
		});
	});

	loadBtnEl.addEventListener("click", (e) => {
		e.preventDefault();
		void loadAll();
	});
	saveBtnEl.addEventListener("click", (e) => {
		e.preventDefault();
		void saveAll();
	});

	void loadAll();
}

function renderChatUI(): void {
	appEl.innerHTML = `
  <div class="wrap">
    <header class="header">
      <div class="title">PicoClaw</div>
      <div class="header-right">
        <label class="theme" for="theme">
          <span class="theme-label">Theme</span>
          <select class="theme-select" id="theme">
            <option value="system">System</option>
            <option value="light">Light</option>
            <option value="dark">Dark</option>
          </select>
        </label>
        <div class="status" id="status">disconnected</div>
      </div>
    </header>

    <main class="chat" id="chat"></main>

    <form class="composer" id="form">
      <input class="input" id="input" placeholder="Type a message..." autocomplete="off" />
      <button class="btn" id="send" type="submit">Send</button>
    </form>
  </div>
`;

	const chat = document.querySelector<HTMLDivElement>("#chat");
	const statusEl = document.querySelector<HTMLDivElement>("#status");
	const themeSelect = document.querySelector<HTMLSelectElement>("#theme");
	const form = document.querySelector<HTMLFormElement>("#form");
	const input = document.querySelector<HTMLInputElement>("#input");

	if (!chat || !statusEl || !themeSelect || !form || !input) {
		throw new Error("Missing UI elements");
	}

	const chatEl = chat as HTMLDivElement;
	const statusDiv = statusEl as HTMLDivElement;

	const chatId = "browser";
	const TOKEN_STORAGE_KEY = "picoclaw.gateway_token";

	const urlParams = new URLSearchParams(location.search);
	const tokenFromUrl = urlParams.get("token") || "";
	const tokenFromStorage = localStorage.getItem(TOKEN_STORAGE_KEY) || "";
	const gatewayToken = tokenFromUrl || tokenFromStorage;
	let tokenCameFromUrl = Boolean(tokenFromUrl);

	function removeTokenFromUrl(): void {
		const p = new URLSearchParams(location.search);
		if (!p.has("token")) return;
		p.delete("token");
		const qs = p.toString();
		const newUrl = `${location.pathname}${qs ? `?${qs}` : ""}${location.hash || ""}`;
		history.replaceState(null, "", newUrl);
	}

	themeSelect.value = themeMode;

	themeSelect.addEventListener("change", () => {
		const v = themeSelect.value;
		if (v === "light" || v === "dark" || v === "system") {
			themeMode = v;
		} else {
			themeMode = "system";
		}
		localStorage.setItem(THEME_STORAGE_KEY, themeMode);
		applyTheme(themeMode);
	});

	const media = window.matchMedia("(prefers-color-scheme: dark)");
	media.addEventListener("change", () => {
		if (themeMode === "system") applyTheme("system");
	});

	function addMessage(role: "user" | "assistant", text: string): void {
		const item = document.createElement("div");
		item.className = `msg ${role}`;

		const bubble = document.createElement("div");
		bubble.className = "bubble";
		bubble.textContent = text;

		item.appendChild(bubble);
		chatEl.appendChild(item);
		chatEl.scrollTop = chatEl.scrollHeight;
	}

	function wsUrl(): string {
		const proto = location.protocol === "https:" ? "wss:" : "ws:";
		const tokenPart = gatewayToken
			? `&token=${encodeURIComponent(gatewayToken)}`
			: "";
		return `${proto}//${location.host}/ws?chat_id=${encodeURIComponent(chatId)}${tokenPart}`;
	}

	type WSMessage = { type?: unknown; content?: unknown };

	let ws: WebSocket | null = null;
	let reconnectTimer: number | null = null;

	function setStatus(s: string): void {
		statusDiv.textContent = s;
		statusDiv.dataset.state = s;
	}

	function connect(): void {
		if (
			ws &&
			(ws.readyState === WebSocket.OPEN ||
				ws.readyState === WebSocket.CONNECTING)
		)
			return;

		setStatus("connecting");
		ws = new WebSocket(wsUrl());

		ws.addEventListener("open", () => {
			setStatus("connected");

			if (tokenCameFromUrl && gatewayToken) {
				localStorage.setItem(TOKEN_STORAGE_KEY, gatewayToken);
				removeTokenFromUrl();
				tokenCameFromUrl = false;
			}
		});

		ws.addEventListener("close", () => {
			setStatus("disconnected");
			if (reconnectTimer == null) {
				reconnectTimer = window.setTimeout(() => {
					reconnectTimer = null;
					connect();
				}, 1000);
			}
		});

		ws.addEventListener("message", (ev: MessageEvent<string>) => {
			try {
				const msg = JSON.parse(ev.data) as WSMessage;
				if (msg && msg.type === "message" && typeof msg.content === "string") {
					addMessage("assistant", msg.content);
				}
			} catch {
				// ignore
			}
		});
	}

	form.addEventListener("submit", (e) => {
		e.preventDefault();
		const text = input.value.trim();
		if (!text) return;
		input.value = "";

		addMessage("user", text);

		if (!ws || ws.readyState !== WebSocket.OPEN) {
			connect();
		}

		const payload = { chat_id: chatId, content: text };
		try {
			ws?.send(JSON.stringify(payload));
		} catch {
			// ignore
		}
	});

	connect();
}

if (location.pathname.startsWith("/admin")) {
	renderAdminUI();
} else {
	renderChatUI();
}
