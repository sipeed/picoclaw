const themeIcons= {
    light:"☀️",dark:"🌙",system:"💻"
},themeOrder=["system","light","dark"];
let currentThemeSetting=localStorage.getItem("picoclaw-theme")||"system";
function getSystemTheme() {
    return window.matchMedia("(prefers-color-scheme: dark)").matches?"dark":"light";

}function applyTheme() {
    const r=currentThemeSetting==="system"?getSystemTheme():currentThemeSetting;
    document.documentElement.setAttribute("data-theme",r);
    const b=document.getElementById("btnTheme");
    if(b)b.textContent=themeIcons[currentThemeSetting];

}function cycleTheme() {
    const i=themeOrder.indexOf(currentThemeSetting);
    currentThemeSetting=themeOrder[(i+1)%themeOrder.length];
    localStorage.setItem("picoclaw-theme",currentThemeSetting);
    applyTheme();

}window.matchMedia("(prefers-color-scheme: dark)").addEventListener("change",()=> {
    if(currentThemeSetting==="system")applyTheme();

});
applyTheme();
let i18nData = { en: {}, zh: {} };
const preferredLangOrder = ["en","zh","ja","pt-br","vi","fr"];
const langDisplayNames = {
    en: "English",
    zh: "中文",
    ja: "日本語",
    "pt-br": "Português (Brasil)",
    vi: "Tiếng Việt",
    fr: "Français"
};
async function loadI18nData() {
    try {
        const res = await fetch("./i18n.json", { cache: "no-store" });
        if(!res.ok)throw new Error("HTTP " + res.status);
        const data = await res.json();
        if(data && typeof data === "object") {
            i18nData = data;
        }
    } catch (e) {
        console.error("Failed to load i18n.json:", e);
        i18nData = { en: {}, zh: {} };
    }
}
function normalizeLangTag(tag) {
    return String(tag||"").toLowerCase();
}
function getAvailableLangs() {
    const keys=Object.keys(i18nData|| {

    }).map(normalizeLangTag).filter(Boolean);
    if(!keys.length)return ["en"];
    const ordered=preferredLangOrder.filter(k=>keys.includes(k));
    const extra=keys.filter(k=>!ordered.includes(k));
    return [...ordered,...extra];
}
function detectBrowserLanguage(available) {
    const avail=new Set(available.map(normalizeLangTag));
    const rawCandidates=[...(Array.isArray(navigator.languages)?navigator.languages:[]),navigator.language].filter(Boolean);
    for(const raw of rawCandidates) {
        const candidate=normalizeLangTag(raw);
        if(avail.has(candidate))return candidate;
        const base=candidate.split("-")[0];
        if(avail.has(base))return base;
        if(base==="pt"&&avail.has("pt-br"))return "pt-br";
    }
    return avail.has("en")?"en":available[0];
}
function resolveInitialLanguage() {
    const available=getAvailableLangs();
    const saved=normalizeLangTag(localStorage.getItem("picoclaw-lang"));
    if(saved&&available.includes(saved))return saved;
    return detectBrowserLanguage(available);
}
function populateLanguageSelect() {
    const sel=document.getElementById("langSelect");
    if(!sel)return;
    const available=getAvailableLangs();
    sel.innerHTML="";
    available.forEach(lang=> {
        const op=document.createElement("option");
        op.value=lang;
        op.textContent=langDisplayNames[lang]||lang;
        sel.appendChild(op);
    });
    sel.value=available.includes(currentLang)?currentLang:(available[0]||"en");
}
let currentLang="en";
function t(key,params) {
    let s=(i18nData[currentLang]&&i18nData[currentLang][key])||i18nData.en[key]||key;
    if(params)Object.keys(params).forEach(k=> {
        s=s.replace("{"+k+"}",params[k]);

    });
    return s;

}function applyI18n() {
    document.querySelectorAll("[data-i18n]").forEach(el=> {
        const key=el.dataset.i18n,val=t(key);
        if(el.tagName==="INPUT"||el.tagName==="TEXTAREA")el.placeholder=val;
        else el.textContent=val;

    });
    const d=document.getElementById("authDesc");
    if(d)d.innerHTML=t("auth.desc");
    const sel=document.getElementById("langSelect");
    if(sel&&sel.value!==currentLang)sel.value=currentLang;
    document.documentElement.lang=currentLang;

}function setLanguage(lang) {
    const normalized=normalizeLangTag(lang);
    const available=getAvailableLangs();
    if(!available.includes(normalized))return;
    currentLang=normalized;
    localStorage.setItem("picoclaw-lang",currentLang);
    applyI18n();
    updateRunStopButton(gatewayRunning);
    renderModels();
    const p=document.querySelector(".content-panel.active");
    if(p) {
        const id=p.id;
        if(id==="panelAuth")loadAuthStatus();
        if(id.startsWith("panelCh_"))renderChannelForm(id.replace("panelCh_",""));

    }

}function checkI18nCompleteness() {
    const enKeys=Object.keys(i18nData.en|| {

    });
    const report= {

    };
    Object.keys(i18nData|| {

    }).forEach(lang=> {
        if(lang==="en")return;
        const keys=Object.keys(i18nData[lang]|| {

        });
        const missing=enKeys.filter(k=>!keys.includes(k));
        if(missing.length)report[lang]=missing;
    });
    if(Object.keys(report).length)console.warn("[i18n] missing keys",report);

}function isMobileViewport() {
    return window.matchMedia("(max-width: 768px)").matches;

}function closeMobileSidebar() {
    document.body.classList.remove("mobile-sidebar-open");

}function toggleMobileSidebar() {
    document.body.classList.toggle("mobile-sidebar-open");

}function toggleNavbarMenu() {
    const burger=document.getElementById("navbarBurger"),menu=document.getElementById("navbarMenu");
    if(!burger||!menu)return;
    const expanded=burger.getAttribute("aria-expanded")==="true";
    burger.setAttribute("aria-expanded",String(!expanded));
    burger.classList.toggle("is-active");
    menu.classList.toggle("is-active");

}function setSwitchA11yState(el) {
    if(!el)return;
    el.setAttribute("role","switch");
    el.setAttribute("tabindex","0");
    el.setAttribute("aria-checked",el.classList.contains("on")?"true":"false");

}function toggleSwitchElement(el) {
    if(!el)return;
    el.classList.toggle("on");
    setSwitchA11yState(el);

}function clearFieldErrors(root) {
    if(!root)return;
    root.querySelectorAll(".is-danger").forEach(el=>el.classList.remove("is-danger"));
    root.querySelectorAll(".field-error").forEach(el=>el.remove());

}function addFieldError(input,msg) {
    if(!input)return;
    input.classList.add("is-danger");
    const p=document.createElement("p");
    p.className="help is-danger field-error";
    p.textContent=msg;
    const field=input.closest(".field");
    if(field)field.appendChild(p);

}function isLikelyURLField(key) {
    return /(^|_)(url|ws_url|bridge_url)$/.test(key)||key==="webhook_url";

}function isSensitiveFieldKey(key) {
    return /(token|secret|api_key|password|encrypt_key|verification_token|access_token)/i.test(key||"");

}function validateURLValue(value) {
    try {
        const u=new URL(value);
        return !!u.protocol&&!!u.host;

    }
    catch(_) {
        return false;

    }

}let configData=null,configPath="",authPollTimer=null,editingModelIndex=-1;
const channelSchemas= {
    telegram: {
        title:"Telegram",titleKey:"channel.telegram",configKey:"telegram",docSlug:"telegram",fields:[ {
            key:"token",label:"Bot Token",type:"password",placeholder:"Telegram bot token from @BotFather"
        }, {
            key:"proxy",label:"Proxy",type:"text",placeholder:"http://proxy:port"
        }
        ]
    },discord: {
        title:"Discord",titleKey:"channel.discord",configKey:"discord",docSlug:"discord",fields:[ {
            key:"token",label:"Bot Token",type:"password",placeholder:"Discord bot token"
        }, {
            key:"mention_only",label:"ch.mentionOnly",type:"toggle",hint:"ch.mentionOnlyHint",i18nLabel:true
        }
        ]
    },slack: {
        title:"Slack",titleKey:"channel.slack",configKey:"slack",docSlug:"slack",fields:[ {
            key:"bot_token",label:"Bot Token",type:"password",placeholder:"xoxb-..."
        }, {
            key:"app_token",label:"App Token",type:"password",placeholder:"xapp-..."
        }
        ]
    },wecom: {
        title:"WeCom (Bot)",titleKey:"channel.wecom",configKey:"wecom",docSlug:"wecom-bot",fields:[ {
            key:"token",label:"Token",type:"password",placeholder:"Verification token"
        }, {
            key:"encoding_aes_key",label:"Encoding AES Key",type:"password",placeholder:"43-char AES key"
        }, {
            key:"webhook_url",label:"Webhook URL",type:"text",placeholder:"https://qyapi.weixin.qq.com/..."
        }, {
            key:"webhook_host",label:"Webhook Host",type:"text",placeholder:"0.0.0.0"
        }, {
            key:"webhook_port",label:"Webhook Port",type:"number",placeholder:"18793"
        }, {
            key:"webhook_path",label:"Webhook Path",type:"text",placeholder:"/webhook/wecom"
        }, {
            key:"reply_timeout",label:"Reply Timeout (s)",type:"number",placeholder:"5"
        }
        ]
    },wecom_app: {
        title:"WeCom (App)",titleKey:"channel.wecom_app",configKey:"wecom_app",docSlug:"wecom-app",fields:[ {
            key:"corp_id",label:"Corp ID",type:"text",placeholder:"Corporation ID"
        }, {
            key:"corp_secret",label:"Corp Secret",type:"password",placeholder:"Corporation secret"
        }, {
            key:"agent_id",label:"Agent ID",type:"number",placeholder:"Agent ID (number)"
        }, {
            key:"token",label:"Token",type:"password",placeholder:"Verification token"
        }, {
            key:"encoding_aes_key",label:"Encoding AES Key",type:"password",placeholder:"43-char AES key"
        }, {
            key:"webhook_host",label:"Webhook Host",type:"text",placeholder:"0.0.0.0"
        }, {
            key:"webhook_port",label:"Webhook Port",type:"number",placeholder:"18792"
        }, {
            key:"webhook_path",label:"Webhook Path",type:"text",placeholder:"/webhook/wecom-app"
        }, {
            key:"reply_timeout",label:"Reply Timeout (s)",type:"number",placeholder:"5"
        }
        ]
    },dingtalk: {
        title:"DingTalk",titleKey:"channel.dingtalk",configKey:"dingtalk",docSlug:"dingtalk",fields:[ {
            key:"client_id",label:"Client ID",type:"text",placeholder:"App key / Client ID"
        }, {
            key:"client_secret",label:"Client Secret",type:"password",placeholder:"App secret"
        }
        ]
    },feishu: {
        title:"Feishu",titleKey:"channel.feishu",configKey:"feishu",docSlug:"feishu",fields:[ {
            key:"app_id",label:"App ID",type:"text",placeholder:"Feishu app ID"
        }, {
            key:"app_secret",label:"App Secret",type:"password",placeholder:"Feishu app secret"
        }, {
            key:"encrypt_key",label:"Encrypt Key",type:"password",placeholder:"Event encrypt key"
        }, {
            key:"verification_token",label:"Verification Token",type:"password",placeholder:"Event verification token"
        }
        ]
    },line: {
        title:"LINE",titleKey:"channel.line",configKey:"line",docSlug:"line",fields:[ {
            key:"channel_secret",label:"Channel Secret",type:"password",placeholder:"LINE channel secret"
        }, {
            key:"channel_access_token",label:"Channel Access Token",type:"password",placeholder:"LINE channel access token"
        }, {
            key:"webhook_host",label:"Webhook Host",type:"text",placeholder:"0.0.0.0"
        }, {
            key:"webhook_port",label:"Webhook Port",type:"number",placeholder:"18791"
        }, {
            key:"webhook_path",label:"Webhook Path",type:"text",placeholder:"/webhook/line"
        }
        ]
    },whatsapp: {
        title:"WhatsApp",titleKey:"channel.whatsapp",configKey:"whatsapp",docSlug:null,fields:[ {
            key:"bridge_url",label:"Bridge URL",type:"text",placeholder:"ws://localhost:3001"
        }
        ]
    },qq: {
        title:"QQ",titleKey:"channel.qq",configKey:"qq",docSlug:"qq",fields:[ {
            key:"app_id",label:"App ID",type:"text",placeholder:"QQ bot App ID"
        }, {
            key:"app_secret",label:"App Secret",type:"password",placeholder:"QQ bot App Secret"
        }
        ]
    },onebot: {
        title:"OneBot",titleKey:"channel.onebot",configKey:"onebot",docSlug:"onebot",fields:[ {
            key:"ws_url",label:"WebSocket URL",type:"text",placeholder:"ws://127.0.0.1:3001"
        }, {
            key:"access_token",label:"Access Token",type:"password",placeholder:"Access token"
        }, {
            key:"reconnect_interval",label:"Reconnect Interval (s)",type:"number",placeholder:"5"
        }, {
            key:"group_trigger_prefix",label:"ch.groupTrigger",type:"array",placeholder:"Trigger word",i18nLabel:true
        }
        ]
    },maixcam: {
        title:"MaixCAM",titleKey:"channel.maixcam",configKey:"maixcam",docSlug:"maixcam",fields:[ {
            key:"host",label:"Host",type:"text",placeholder:"0.0.0.0"
        }, {
            key:"port",label:"Port",type:"number",placeholder:"18790"
        }
        ]
    }

};
function toggleGroup(el) {
    const g=el.closest(".sidebar-group");
    if(g)g.classList.toggle("collapsed");

}function activatePanel(panelId) {
    document.querySelectorAll(".sidebar-item").forEach(i=>i.classList.remove("is-active"));
    const item=document.querySelector('.sidebar-item[data-panel="'+panelId+'"]');
    if(item)item.classList.add("is-active");
    document.querySelectorAll(".content-panel").forEach(p=>p.classList.remove("active"));
    const panel=document.getElementById(panelId);
    if(panel)panel.classList.add("active");
    if(panelId==="panelModels")renderModels();
    if(panelId==="panelAuth")loadAuthStatus();
    if(panelId==="panelRawJson")syncEditorFromConfig();
    if(panelId.startsWith("panelCh_"))renderChannelForm(panelId.replace("panelCh_",""));
    if(isMobileViewport())closeMobileSidebar();

}function showStatus(text,type) {
    const c=document.getElementById("toastContainer"),n=document.createElement("div");
    n.className="notification is-light "+(type==="success"?"is-success":"is-danger");
    n.textContent=(type==="success"?"✓ ":"✗ ")+text;
    c.appendChild(n);
    setTimeout(()=>n.remove(),3000);

}function hidePersistentNotice() {
    const box=document.getElementById("persistentNotice");
    if(box)box.classList.add("is-hidden");

}function showPersistentNotice(text,titleKey) {
    const box=document.getElementById("persistentNotice"),titleEl=document.getElementById("persistentNoticeTitle"),textEl=document.getElementById("persistentNoticeText");
    if(!box||!titleEl||!textEl)return;
    titleEl.textContent=t(titleKey||"process.startFailed");
    textEl.textContent=text||"";
    box.classList.remove("is-hidden");

}async function loadConfig() {
    try {
        const res=await fetch("/api/config");
        if(!res.ok)throw new Error("HTTP "+res.status+": "+await res.text());
        const data=await res.json();
        configData=data.config;
        configPath=data.path||"";
        document.getElementById("filePath").textContent=configPath||"-";
        renderModels();
        if(!pendingAction)updateRunStopButton(gatewayRunning);

    }
    catch(e) {
        showStatus(t("status.loadFailed")+": "+e.message,"error");

    }

}async function saveConfig() {
    if(!configData)return;
    try {
        const res=await fetch("/api/config", {
            method:"PUT",headers: {
                "Content-Type":"application/json"
            },body:JSON.stringify(configData)
        });
        if(!res.ok)throw new Error("HTTP "+res.status+": "+await res.text());
        showStatus(t("status.configSaved"),"success");

    }
    catch(e) {
        showStatus(t("status.saveFailed")+": "+e.message,"error");

    }
    if(!pendingAction)updateRunStopButton(gatewayRunning);

}let codeEditor=null,codeEditorTextarea=null,plainEditorTextarea=null,cm6LoadPromise=null;
function getEditorValue() {
    if(codeEditor&&typeof codeEditor.getCode==="function")return codeEditor.getCode();
    if(plainEditorTextarea)return plainEditorTextarea.value||"";
    const editor=document.getElementById("editor");
    return editor?editor.textContent||"":"";
}
function setEditorValue(v) {
    if(codeEditor&&typeof codeEditor.setCode==="function") {
        codeEditor.setCode(v||"");
        return;
    }
    if(plainEditorTextarea) {
        plainEditorTextarea.value=v||"";
        return;
    }
    const editor=document.getElementById("editor");
    if(editor)editor.textContent=v||"";
}
function getEditorCursorPos() {
    if(codeEditor&&typeof codeEditor.getCursor==="function")return codeEditor.getCursor();
    return codeEditorTextarea?codeEditorTextarea.selectionStart||0:0;
}
function setEditorCursorPos(pos) {
    if(codeEditor&&typeof codeEditor.setCursor==="function") {
        codeEditor.setCursor(pos);
        return;
    }
    if(codeEditorTextarea&&typeof codeEditorTextarea.setSelectionRange==="function") {
        codeEditorTextarea.setSelectionRange(pos,pos);
    }
}
function initPlainEditor(host) {
    const ta=document.createElement("textarea");
    ta.className="textarea app-scroll";
    ta.style.minHeight="clamp(280px, 46vh, 640px)";
    ta.style.resize="vertical";
    ta.spellcheck=false;
    host.replaceChildren(ta);
    codeEditor=null;
    plainEditorTextarea=ta;
    codeEditorTextarea=ta;
    ta.addEventListener("input",()=> {
        validateJson();
        scheduleAutoFormatJson();
    });
    ta.addEventListener("blur",()=> {
        runAutoFormatJson(true);
    });
    ta.addEventListener("paste",()=> {
        setTimeout(()=>runAutoFormatJson(true),0);
    });
}
function loadCodeMirror6() {
    if(cm6LoadPromise)return cm6LoadPromise;
    cm6LoadPromise=Promise.resolve().then(()=> {
        const cmGlobal=window.CM|| {};
        const statePkg=cmGlobal["@codemirror/state"];
        const viewPkg=cmGlobal["@codemirror/view"];
        const bundlePkg=cmGlobal["codemirror"];
        const jsonPkg=cmGlobal["@codemirror/lang-json"];
        const languagePkg=cmGlobal["@codemirror/language"];
        const lezerHighlightPkg=cmGlobal["@lezer/highlight"];
        if(!statePkg||!viewPkg||!bundlePkg||!jsonPkg||!languagePkg||!lezerHighlightPkg) {
            throw new Error("CodeMirror bundles not available on window.CM");
        }
        return {
            EditorState:statePkg.EditorState,
            EditorView:viewPkg.EditorView,
            basicSetup:bundlePkg.basicSetup,
            json:jsonPkg.json,
            HighlightStyle:languagePkg.HighlightStyle,
            syntaxHighlighting:languagePkg.syntaxHighlighting,
            tags:lezerHighlightPkg.tags
        };
    }).catch(err=> {
        cm6LoadPromise=null;
        throw err;
    });
    return cm6LoadPromise;
}
async function initJsonCodeEditor() {
    const host=document.getElementById("editor");
    if(!host)return;
    plainEditorTextarea=null;
    codeEditorTextarea=null;
    try {
        const cm=await loadCodeMirror6();
        const jsonHighlight=cm.HighlightStyle.define([
            { tag: cm.tags.propertyName, color: "#4f8fe8" },
            { tag: cm.tags.string, color: "#2f9d5d" },
            { tag: cm.tags.number, color: "#b8731f" },
            { tag: [cm.tags.bool, cm.tags.null], color: "#8d4fc9" },
            { tag: cm.tags.punctuation, color: "#6f7f9a" }
        ]);
        host.replaceChildren();
        const state=cm.EditorState.create({
            doc:"",
            extensions:[
                cm.basicSetup,
                cm.json(),
                cm.EditorView.lineWrapping,
                cm.syntaxHighlighting(jsonHighlight),
                cm.EditorView.updateListener.of(update=> {
                    if(!update.docChanged)return;
                    validateJson();
                    scheduleAutoFormatJson();
                })
            ]
        });
        const view=new cm.EditorView({
            state,
            parent:host
        });
        codeEditor={
            type:"cm6",
            view,
            getCode() {
                return view.state.doc.toString();
            },
            setCode(next) {
                const current=view.state.doc.toString();
                const value=next||"";
                if(current===value)return;
                view.dispatch({
                    changes:{
                        from:0,
                        to:current.length,
                        insert:value
                    }
                });
            },
            getCursor() {
                return view.state.selection.main.from;
            },
            setCursor(pos) {
                const p=Math.max(0,Math.min(pos,view.state.doc.length));
                view.dispatch({
                    selection:{
                        anchor:p,
                        head:p
                    },
                    scrollIntoView:true
                });
            }
        };
        codeEditorTextarea=view.contentDOM;
        if(codeEditorTextarea)codeEditorTextarea.addEventListener("blur",()=> {
            runAutoFormatJson(true);
        });
        if(codeEditorTextarea)codeEditorTextarea.addEventListener("paste",()=> {
            setTimeout(()=>runAutoFormatJson(true),0);
        });
        return;
    } catch(err) {
        console.error("Failed to initialize CodeMirror 6, fallback to textarea:",err);
        initPlainEditor(host);
    }

}function syncEditorFromConfig() {
    if(configData)setEditorValue(JSON.stringify(configData,null,2));
    document.getElementById("filePath").textContent=configPath||"-";
    validateJson();

}async function saveRawConfig() {
    const o=validateJson();
    if(o===null) {
        showStatus(t("status.invalidJson"),"error");
        return;

    }
    configData=o;
    await saveConfig();
    setEditorValue(JSON.stringify(configData,null,2));

}function formatJson() {
    const o=validateJson();
    if(o!==null) {
        setEditorValue(JSON.stringify(o,null,2));
        showStatus(t("status.formatted"),"success");

    }

}function validateJson() {
    const s=document.getElementById("jsonStatus"),w=document.getElementById("editorWrapper"),v=getEditorValue().trim();
    if(!v) {
        s.textContent="-";
        w.classList.remove("error");
        return null;

    }
    try {
        const o=JSON.parse(v);
        s.textContent="✓ "+t("status.jsonValid");
        s.style.color="var(--success)";
        w.classList.remove("error");
        return o;

    }
    catch(err) {
        s.textContent="✗ "+t("status.jsonInvalid")+": "+err.message;
        s.style.color="var(--error)";
        w.classList.add("error");
        return null;

    }

}function renderModels() {
    const g=document.getElementById("modelGrid");
    if(!configData||!configData.model_list) {
        g.innerHTML='<div class="column is-full"><p class="has-text-grey">'+t("models.noModels")+"</p></div>";
        return;

    }
    const p=(configData.agents&&configData.agents.defaults)?(configData.agents.defaults.model_name||configData.agents.defaults.model||""):"",idxs=configData.model_list.map((_,i)=>i);
    idxs.sort((a,b)=> {
        const ap=configData.model_list[a].model_name===p?0:1,bp=configData.model_list[b].model_name===p?0:1;
        return ap!==bp?ap-bp:a-b;

    });
    let h="";
    idxs.forEach(idx=> {
        const m=configData.model_list[idx],av=isModelAvailableGlobal(m),isP=m.model_name===p,proto=m.model?m.model.split("/")[0]:"";
        let details="";
        details+='<div class="model-detail-row"><span class="model-detail-key">Model</span><span class="model-detail-val">'+esc(m.model||"-")+"</span></div>";
        if(m.api_base)details+='<div class="model-detail-row"><span class="model-detail-key">API Base</span><span class="model-detail-val">'+esc(m.api_base)+"</span></div>";
        if(m.api_key)details+='<div class="model-detail-row"><span class="model-detail-key">API Key</span><span class="model-detail-val">'+maskKey(m.api_key)+"</span></div>";
        if(m.auth_method)details+='<div class="model-detail-row"><span class="model-detail-key">Auth</span><span class="model-detail-val">'+esc(m.auth_method)+"</span></div>";
        if(m.proxy)details+='<div class="model-detail-row"><span class="model-detail-key">Proxy</span><span class="model-detail-val">'+esc(m.proxy)+"</span></div>";
        h+='<div class="column is-full-tablet is-half-desktop"><div class="card model-card '+(av?"":"unavailable")+'"><div class="card-content"><div class="is-flex is-justify-content-space-between is-align-items-center mb-2"><strong>'+esc(m.model_name||"-")+'</strong><div class="is-flex is-align-items-center" style="gap:.4rem;">'+(proto?'<span class="tag is-light">'+esc(proto)+"</span>":"")+(isP?'<span class="badge badge-primary">'+t("models.primary")+"</span>":(!av?'<span class="badge badge-muted">'+t("models.noKey")+"</span>":""))+'</div></div><div class="model-details">'+details+'</div><div class="buttons mt-4"><button class="button is-small" data-action="edit-model" data-index="'+idx+'">'+t("edit")+'</button>'+(av&&!isP?'<button class="button is-small is-success is-light" data-action="set-primary-model" data-index="'+idx+'">'+t("models.setPrimary")+"</button>":"")+'<button class="button is-small is-danger is-light" data-action="delete-model" data-index="'+idx+'">'+t("delete")+"</button></div></div></div></div>";

    });
    g.innerHTML=h;

}function setPrimaryModel(idx) {
    if(!configData||!configData.model_list[idx])return;
    if(!configData.agents)configData.agents= {

    };
    if(!configData.agents.defaults)configData.agents.defaults= {

    };
    configData.agents.defaults.model_name=configData.model_list[idx].model_name;
    saveConfig().then(renderModels);

}function deleteModel(idx) {
    if(!configData||!configData.model_list)return;
    const n=configData.model_list[idx].model_name;
    if(!confirm(t("models.deleteConfirm", {
        name:n
    })))return;
    configData.model_list.splice(idx,1);
    saveConfig().then(renderModels);

}const modelFieldsRequired=[ {
    key:"model_name",labelKey:"field.modelName",type:"text",placeholder:"e.g. gpt-4o",required:true
}, {
    key:"model",labelKey:"field.modelId",type:"text",placeholder:"e.g. openai/gpt-4o",required:true,hintKey:"field.modelIdHint"
}, {
    key:"api_key",labelKey:"field.apiKey",type:"password",placeholder:"API key"
}, {
    key:"api_base",labelKey:"field.apiBase",type:"text",placeholder:"https://api.openai.com/v1"
}
],modelFieldsOptional=[ {
    key:"proxy",labelKey:"field.proxy",type:"text",placeholder:"http://proxy:port"
}, {
    key:"auth_method",labelKey:"field.authMethod",type:"text",placeholder:"oauth / token"
}, {
    key:"connect_mode",labelKey:"field.connectMode",type:"text",placeholder:"stdio / grpc"
}, {
    key:"workspace",labelKey:"field.workspace",type:"text",placeholder:"Workspace path"
}, {
    key:"rpm",labelKey:"field.rpm",type:"number",placeholder:"RPM"
}, {
    key:"request_timeout",labelKey:"field.requestTimeout",type:"number",placeholder:"Seconds"
}
],modelFields=[...modelFieldsRequired,...modelFieldsOptional];
function showEditModelModal(idx) {
    editingModelIndex=idx;
    const m=configData.model_list[idx];
    document.getElementById("modalTitle").textContent=t("models.editModel")+": "+m.model_name;
    renderModalBody(m);
    document.getElementById("modelModal").classList.add("is-active");

}function showAddModelModal() {
    editingModelIndex=-1;
    document.getElementById("modalTitle").textContent=t("models.addModel");
    renderModalBody( {

    });
    document.getElementById("modelModal").classList.add("is-active");

}function closeModelModal() {
    document.getElementById("modelModal").classList.remove("is-active");

}function renderModalBody(data) {
    function f(field) {
        const val=data[field.key]!==undefined&&data[field.key]!==null?data[field.key]:"";
        const resolvedType=(field.type==="password"||isSensitiveFieldKey(field.key))?"password":(field.type==="number"?"number":"text");
        let h='<div class="field"><label class="label">'+t(field.labelKey)+(field.required?" *":"")+'</label><div class="control"><input class="input ui-input-compact '+(resolvedType==="password"?"ui-input-password":"")+'" type="'+resolvedType+'" data-field="'+field.key+'" value="'+esc(String(val))+'" placeholder="'+(field.placeholder||"")+'"></div>';
        if(field.hintKey)h+='<p class="help">'+t(field.hintKey)+"</p>";
        return h+"</div>";

    }
    let html="";
    modelFieldsRequired.forEach(x=> {
        html+=f(x);

    });
    html+='<details class="mb-2"><summary>'+t("models.advancedOptions")+'</summary><div class="mt-3">';
    modelFieldsOptional.forEach(x=> {
        html+=f(x);

    });
    html+="</div></details>";
    document.getElementById("modalBody").innerHTML=html;

}function saveModelFromModal() {
    const modalBody=document.getElementById("modalBody"),inputs=document.querySelectorAll("#modalBody input[data-field]"),obj= {

    };
    clearFieldErrors(modalBody);
    inputs.forEach(input=> {
        const k=input.dataset.field;
        let v=input.value.trim();
        if(input.type==="number"&&v)v=parseInt(v,10)||0;
        if(v!==""&&v!==0)obj[k]=v;
        else if(k==="model_name"||k==="model")obj[k]=v;

    });
    let hasError=false;
    const modelNameInput=modalBody.querySelector('input[data-field="model_name"]');
    const modelIdInput=modalBody.querySelector('input[data-field="model"]');
    if(!obj.model_name) {
        addFieldError(modelNameInput,t("models.requiredFields"));
        hasError=true;

    }
    if(!obj.model) {
        addFieldError(modelIdInput,t("models.requiredFields"));
        hasError=true;

    }
    const rpmInput=modalBody.querySelector('input[data-field="rpm"]');
    if(rpmInput&&rpmInput.value.trim()!==""&&parseInt(rpmInput.value,10)<0) {
        addFieldError(rpmInput,"RPM must be >= 0");
        hasError=true;

    }
    const timeoutInput=modalBody.querySelector('input[data-field="request_timeout"]');
    if(timeoutInput&&timeoutInput.value.trim()!==""&&parseInt(timeoutInput.value,10)<=0) {
        addFieldError(timeoutInput,"Timeout must be > 0");
        hasError=true;

    }
    if(hasError) {
        showStatus(t("models.requiredFields"),"error");
        return;

    }
    if(!configData.model_list)configData.model_list=[];
    if(editingModelIndex>=0) {
        configData.model_list[editingModelIndex]= {
            ...configData.model_list[editingModelIndex],...obj
        };
        modelFields.forEach(x=> {
            if(!x.required&&(obj[x.key]===""||obj[x.key]===0))delete configData.model_list[editingModelIndex][x.key];

        });

    }
    else {
        configData.model_list.push(obj);
        if(!configData.agents)configData.agents= {

        };
        if(!configData.agents.defaults)configData.agents.defaults= {

        };
        configData.agents.defaults.model_name=obj.model_name;

    }
    closeModelModal();
    saveConfig().then(renderModels);

}function renderChannelForm(chKey) {
    const s=channelSchemas[chKey];
    if(!s)return;
    const displayTitle=s.titleKey?t(s.titleKey):s.title;
    const panel=document.getElementById("panelCh_"+chKey),d=(configData&&configData.channels&&configData.channels[s.configKey])|| {

    };
    let h='<div class="panel-box"><h2 class="title is-4">'+displayTitle+'</h2><p class="subtitle is-6">'+t("ch.configure", {
        name:displayTitle
    });
    if(s.docSlug) {
        const b=currentLang==="zh"?"https://docs.picoclaw.io/zh-Hans/docs/channels/":"https://docs.picoclaw.io/docs/channels/";
        h+=' <a href="'+b+s.docSlug+'" target="_blank" rel="noopener noreferrer">📖 '+t("ch.docLink")+"</a>";

    }
    h+='</p><div class="field is-grouped is-align-items-center"><div class="switch '+(d.enabled?"on":"")+'" data-action="toggle-channel-enabled" data-chkey="'+chKey+'" id="chToggle_'+chKey+'" role="switch" tabindex="0" aria-checked="'+(d.enabled?"true":"false")+'"></div><span class="ml-2">'+t("enabled")+'</span></div><div id="chForm_'+chKey+'">';
    s.fields.forEach(f=> {
        const label=f.i18nLabel?t(f.label):f.label;
        if(f.type==="toggle") {
            const hint=f.i18nLabel&&f.hint?t(f.hint):(f.hint||"");
            h+='<div class="field is-grouped is-align-items-center"><div class="switch '+(d[f.key]?"on":"")+'" data-action="toggle-field-switch" data-chfield="'+f.key+'" role="switch" tabindex="0" aria-checked="'+(d[f.key]?"true":"false")+'"></div><span class="ml-2">'+label+"</span>"+(hint?'<span class="help ml-2">'+hint+"</span>":"")+"</div>";

        }
        else if(f.type==="array") {
            const arr=d[f.key]||[];
            h+='<div class="field"><label class="label">'+label+'</label><div class="array-editor" data-chfield="'+f.key+'" data-placeholder="'+(f.placeholder||"")+'">';
            arr.forEach(v=> {
                h+='<div class="array-row"><input class="input" type="text" value="'+esc(String(v))+'" placeholder="'+(f.placeholder||"")+'"><button class="button is-small is-danger is-light" data-action="remove-array-row" type="button">×</button></div>';

            });
            h+='<button class="button is-small is-light" data-action="add-array-row" type="button">'+t("ch.addItem")+"</button></div></div>";

        }
        else {
            const val=d[f.key]!==undefined&&d[f.key]!==null?d[f.key]:"";
            const resolvedType=(f.type==="password"||isSensitiveFieldKey(f.key))?"password":(f.type==="number"?"number":"text");
            h+='<div class="field"><label class="label">'+label+'</label><div class="control"><input class="input ui-input-compact '+(resolvedType==="password"?"ui-input-password":"")+'" type="'+resolvedType+'" data-chfield="'+f.key+'" value="'+esc(String(val))+'" placeholder="'+(f.placeholder||"")+'"></div></div>';

        }

    });
    const af=d.allow_from||[];
    h+='<h3 class="title is-6 mt-4">'+t("ch.accessControl")+'</h3><div class="field"><label class="label">'+t("ch.allowFrom")+'</label><div class="array-editor" data-chfield="allow_from" data-placeholder="User / Chat ID">';
    af.forEach(v=> {
        h+='<div class="array-row"><input class="input" type="text" value="'+esc(String(v))+'" placeholder="User / Chat ID"><button class="button is-small is-danger is-light" data-action="remove-array-row" type="button">×</button></div>';

    });
    h+='<button class="button is-small is-light" data-action="add-array-row" type="button">'+t("ch.addItem")+'</button></div></div><button class="button is-primary mt-3" data-action="save-channel" data-chkey="'+chKey+'">'+t("save")+"</button></div></div>";
    panel.innerHTML=h;
    panel.querySelectorAll('.switch[data-action]').forEach(setSwitchA11yState);

}function addArrayRow(c) {
    const p=c.dataset.placeholder||"",a=c.querySelector('[data-action="add-array-row"]'),r=document.createElement("div");
    r.className="array-row";
    r.innerHTML='<input class="input" type="text" value="" placeholder="'+esc(p)+'"><button class="button is-small is-danger is-light" data-action="remove-array-row" type="button">×</button>';
    c.insertBefore(r,a);
    r.querySelector("input").focus();

}function validateChannelForm(schema,form) {
    clearFieldErrors(form);
    let hasError=false;
    schema.fields.forEach(f=> {
        if(f.type==="number") {
            const input=form.querySelector('input[data-chfield="'+f.key+'"]');
            if(input&&input.value.trim()!=="") {
                const n=Number(input.value.trim());
                if(Number.isNaN(n)||n<0) {
                    addFieldError(input,"Must be a number >= 0");
                    hasError=true;

                }

            }

        }
        else if(f.type!=="toggle"&&f.type!=="array"&&isLikelyURLField(f.key)) {
            const input=form.querySelector('input[data-chfield="'+f.key+'"]');
            if(input&&input.value.trim()!==""&&!validateURLValue(input.value.trim())) {
                addFieldError(input,"Invalid URL");
                hasError=true;

            }

        }

    });
    return !hasError;

}function saveChannelForm(chKey) {
    const s=channelSchemas[chKey];
    if(!s||!configData)return;
    if(!configData.channels)configData.channels= {

    };
    const o=configData.channels[s.configKey]|| {

    },t0=document.getElementById("chToggle_"+chKey);
    o.enabled=t0?t0.classList.contains("on"):false;
    const form=document.getElementById("chForm_"+chKey);
    if(!validateChannelForm(s,form)) {
        showStatus("Please fix invalid fields before saving","error");
        return;

    }
    s.fields.forEach(f=> {
        if(f.type==="toggle") {
            const e=form.querySelector('[data-chfield="'+f.key+'"].switch');
            if(e)o[f.key]=e.classList.contains("on");

        }
        else if(f.type==="array") {
            const c=form.querySelector('.array-editor[data-chfield="'+f.key+'"]');
            if(c) {
                const vals=[];
                c.querySelectorAll(".array-row input").forEach(i=> {
                    const v=i.value.trim();
                    if(v)vals.push(v);

                });
                o[f.key]=vals;

            }

        }
        else {
            const i=form.querySelector('input[data-chfield="'+f.key+'"]');
            if(i) {
                let v=i.value.trim();
                if(f.type==="number"&&v) {
                    v=parseInt(v,10);
                    if(isNaN(v))v=0;

                }
                o[f.key]=v===""?(f.type==="number"?0:""):v;

            }

        }

    });
    const af=form.querySelector('.array-editor[data-chfield="allow_from"]');
    if(af) {
        const vals=[];
        af.querySelectorAll(".array-row input").forEach(i=> {
            const v=i.value.trim();
            if(v)vals.push(v);

        });
        o.allow_from=vals;

    }
    configData.channels[s.configKey]=o;
    const displayTitle=s.titleKey?t(s.titleKey):s.title;
    saveConfig().then(()=>showStatus(t("status.saved", {
        name:displayTitle
    }),"success"));

}let authProviderMap= {

};
async function loadAuthStatus() {
    try {
        const res=await fetch("/api/auth/status");
        if(!res.ok)return;
        const data=await res.json(),providers=data.providers||[];
        authProviderMap= {

        };
        providers.forEach(p=> {
            authProviderMap[p.provider]=p;

        });
        renderAuthStatus(providers,data.pending_device);

    }
    catch(e) {
        console.error("Failed to load auth status:",e);

    }

}function renderAuthStatus(list,pending) {
    const map= {

    };
    list.forEach(p=> {
        map[p.provider]=p;

    });
    ["openai","anthropic","google-antigravity"].forEach(name=> {
        const b=document.getElementById("badge-"+name),d=document.getElementById("details-"+name),a=document.getElementById("actions-"+name),p=map[name];
        if(p) {
            const bc=p.status==="active"?"badge-active":(p.status==="expired"?"badge-expired":"badge-pending"),bt=p.status==="active"?t("auth.active"):(p.status==="expired"?t("auth.expired"):t("auth.needsRefresh"));
            b.className="provider-badge "+bc;
            b.textContent=bt;
            let h="";
            if(p.auth_method)h+="<p><strong>"+t("auth.method")+":</strong> "+esc(p.auth_method)+"</p>";
            if(p.email)h+="<p><strong>"+t("auth.email")+":</strong> "+esc(p.email)+"</p>";
            if(p.account_id)h+="<p><strong>"+t("auth.account")+":</strong> "+esc(p.account_id)+"</p>";
            if(p.project_id)h+="<p><strong>"+t("auth.project")+":</strong> "+esc(p.project_id)+"</p>";
            if(p.expires_at)h+="<p><strong>"+t("auth.expires")+":</strong> "+new Date(p.expires_at).toLocaleString()+"</p>";
            d.innerHTML=h;
            a.innerHTML='<button class="button is-small is-danger is-light" data-action="logout-provider" data-provider="'+name+'">'+t("auth.logout")+"</button>";

        }
        else {
            b.className="provider-badge badge-none";
            b.textContent=t("auth.notLoggedIn");
            d.innerHTML="";
            if(name==="openai")a.innerHTML='<button class="button is-small is-primary" data-action="login-provider" data-provider="openai">'+t("auth.loginDevice")+"</button>";
            else if(name==="anthropic")a.innerHTML='<button class="button is-small is-primary" data-action="show-token-input" data-provider="anthropic">'+t("auth.loginToken")+"</button>";
            else a.innerHTML='<button class="button is-small is-primary" data-action="login-provider" data-provider="google-antigravity">'+t("auth.loginOAuth")+"</button>";

        }

    });
    if(pending&&pending.status==="pending") {
        const n=pending.provider,b=document.getElementById("badge-"+n),d=document.getElementById("details-"+n),a=document.getElementById("actions-"+n);
        if(b) {
            b.className="provider-badge badge-pending";
            b.textContent=t("auth.authenticating");

        }
        if(pending.device_url&&pending.user_code&&d) {
            d.innerHTML='<div class="notification is-warning is-light"><p>'+t("auth.step1")+'</p><p><a href="'+pending.device_url+'" target="_blank" rel="noopener noreferrer">'+pending.device_url+' ↗</a></p><p>'+t("auth.step2")+': <strong>'+esc(pending.user_code)+'</strong></p><p>'+t("auth.step3")+"</p></div>";

        }
        if(pending.error&&d) {
            d.innerHTML='<p class="has-text-danger">'+esc(pending.error)+"</p>";
            if(a)a.innerHTML='<button class="button is-small is-primary" data-action="login-provider" data-provider="'+n+'">'+t("auth.retry")+"</button>";

        }
        else if(a) {
            a.innerHTML='<button class="button is-small" disabled>'+t("auth.waiting")+"</button>";

        }
        startAuthPolling();

    }
    else stopAuthPolling();

}function startAuthPolling() {
    stopAuthPolling();
    authPollTimer=setInterval(loadAuthStatus,3000);

}function stopAuthPolling() {
    if(authPollTimer) {
        clearInterval(authPollTimer);
        authPollTimer=null;

    }

}async function loginProvider(provider) {
    const a=document.getElementById("actions-"+provider),orig=a?a.innerHTML:"";
    if(a)a.querySelectorAll(".button").forEach(b=> {
        b.disabled=true;

    });
    try {
        const res=await fetch("/api/auth/login", {
            method:"POST",headers: {
                "Content-Type":"application/json"
            },body:JSON.stringify( {
                provider
            })
        });
        if(!res.ok)throw new Error(await res.text());
        const data=await res.json();
        if(data.status==="redirect"&&data.auth_url) {
            showStatus(t("status.openingBrowser"),"success");
            window.open(data.auth_url,"_blank");
            if(a)a.innerHTML=orig;
            return;

        }
        if(data.status==="pending") {
            showStatus(data.message||t("status.loginStarted"),"success");
            startAuthPolling();
            await loadAuthStatus();

        }
        else if(data.status==="success") {
            showStatus(data.message||t("status.loginSuccess"),"success");
            loadAuthStatus();

        }

    }
    catch(e) {
        showStatus(t("status.loginFailed")+": "+e.message,"error");
        if(a)a.innerHTML=orig;

    }

}function showTokenInput(provider) {
    const a=document.getElementById("actions-"+provider);
    a.innerHTML='<div class="field has-addons"><div class="control"><input class="input ui-input-compact ui-input-password" type="password" id="tokenInput-'+provider+'" placeholder="'+t("auth.pasteKey")+'"></div><div class="control"><button class="button is-primary is-small" data-action="submit-token" data-provider="'+provider+'">'+t("save")+'</button></div><div class="control"><button class="button is-small" data-action="cancel-token" data-provider="'+provider+'">'+t("cancel")+"</button></div></div>";
    document.getElementById("tokenInput-"+provider).focus();

}async function submitToken(provider) {
    const input=document.getElementById("tokenInput-"+provider),token=input.value.trim();
    if(!token) {
        showStatus(t("status.tokenEmpty"),"error");
        return;

    }
    try {
        const res=await fetch("/api/auth/login", {
            method:"POST",headers: {
                "Content-Type":"application/json"
            },body:JSON.stringify( {
                provider,token
            })
        });
        if(!res.ok)throw new Error(await res.text());
        showStatus(t("status.tokenSaved", {
            name:provider
        }),"success");
        loadAuthStatus();

    }
    catch(e) {
        showStatus(t("status.loginFailed")+": "+e.message,"error");

    }

}async function logoutProvider(provider) {
    try {
        const res=await fetch("/api/auth/logout", {
            method:"POST",headers: {
                "Content-Type":"application/json"
            },body:JSON.stringify( {
                provider
            })
        });
        if(!res.ok)throw new Error(await res.text());
        showStatus(t("status.loggedOut", {
            name:provider
        }),"success");
        loadAuthStatus();

    }
    catch(e) {
        showStatus(t("status.logoutFailed")+": "+e.message,"error");

    }

}let gatewayRunning=false,processPolling=null,pendingAction=null;
const MAX_RETRIES=10;
function isModelAvailableGlobal(m) {
    if(m.api_key)return true;
    if(m.auth_method==="oauth") {
        const p=m.model?m.model.split("/")[0]:"",name=p==="google-antigravity"?"google-antigravity":p,auth=authProviderMap[name];
        return !!(auth&&auth.status==="active");

    }
    if(m.auth_method)return true;
    return false;

}function checkStartPrereqs() {
    const p=configData&&configData.agents&&configData.agents.defaults?(configData.agents.defaults.model_name||""):"";
    const hasModel=!!(p&&configData&&configData.model_list&&configData.model_list.some(m=>m.model_name===p&&isModelAvailableGlobal(m)));
    const hasChannel=configData&&configData.channels&&Object.keys(channelSchemas).some(k=> {
        const c=configData.channels[channelSchemas[k].configKey];
        return c&&c.enabled;

    });
    return {
        hasModel,hasChannel,canStart:hasModel&&hasChannel
    };

}function updateRunStopButton(running) {
    gatewayRunning=running;
    const b=document.getElementById("btnRunStop"),i=document.getElementById("btnRunStopIcon"),t0=document.getElementById("btnRunStopText"),h=document.getElementById("processHint");
    if(running) {
        b.disabled=false;
        b.className="button is-danger";
        i.textContent="■";
        t0.textContent=t("stop");
        h.textContent="("+t("process.running")+")";
        h.style.color="";

    }
    else {
        const p=checkStartPrereqs();
        b.disabled=!p.canStart;
        b.className="button is-success";
        i.textContent="▶";
        t0.textContent=t("start");
        if(!p.canStart) {
            const r=!p.hasModel&&!p.hasChannel?t("process.needBoth"):(!p.hasModel?t("process.needModel"):t("process.needChannel"));
            h.textContent="("+r+")";
            h.style.color="var(--error)";

        }
        else {
            h.textContent="("+t("process.notRunning")+")";
            h.style.color="";

        }

    }

}function setButtonLoading(type) {
    const b=document.getElementById("btnRunStop"),i=document.getElementById("btnRunStopIcon"),t0=document.getElementById("btnRunStopText"),h=document.getElementById("processHint");
    b.disabled=true;
    b.className="button is-warning";
    i.textContent="⏳";
    t0.textContent=type==="start"?t("process.starting"):t("process.stopping");
    h.textContent="";

}async function checkProcessStatus() {
    let running=false;
    try {
        const p=new URLSearchParams( {
            log_offset:logOffset,log_run_id:logRunID
        }),res=await fetch("/api/process/status?"+p);
        if(res.ok) {
            const d=await res.json();
            running=d.process_status==="running";
            handleLogData(d);

        }

    }
    catch(_) {
        running=false;

    }
    if(pendingAction) {
        const e=pendingAction.type==="start";
        if(running===e) {
            showStatus(e?t("process.started"):t("process.stopped"),"success");
            if(e)hidePersistentNotice();
            pendingAction=null;
            restoreNormalPolling();
            updateRunStopButton(running);

        }
        else {
            pendingAction.retries++;
            if(pendingAction.retries>=MAX_RETRIES) {
                let m=pendingAction.type==="start"?t("process.startFailed"):t("process.stopFailed");
                if(pendingAction.type==="start")m+=" - "+t("process.checkLogs");
                showStatus(m,"error");
                pendingAction=null;
                restoreNormalPolling();
                updateRunStopButton(running);

            }

        }

    }
    else updateRunStopButton(running);

}function restoreNormalPolling() {
    clearInterval(processPolling);
    processPolling=setInterval(checkProcessStatus,5000);

}function isPicoclawNotFoundError(rawMessage) {
    const msg=String(rawMessage||"");
    return /exec:\s*"picoclaw".*not found in %PATH%|executable file not found/i.test(msg);

}function buildFriendlyStartError(rawMessage) {
    const msg=String(rawMessage||"");
    const missingPicoclaw=isPicoclawNotFoundError(msg);
    if(!missingPicoclaw)return msg;
    const hint=t("process.picoclawNotFoundHint");
    return msg+"\n"+hint;

}async function startGateway() {
    const p=checkStartPrereqs();
    if(!p.canStart) {
        const r=!p.hasModel&&!p.hasChannel?t("process.needBoth"):(!p.hasModel?t("process.needModel"):t("process.needChannel"));
        showStatus(r,"error");
        return;

    }
    setButtonLoading("start");
    pendingAction= {
        type:"start",retries:0
    };
    clearInterval(processPolling);
    processPolling=setInterval(checkProcessStatus,1000);
    try {
        const res=await fetch("/api/process/start", {
            method:"POST"
        });
        if(!res.ok)throw new Error(await res.text());

    }
    catch(e) {
        const raw=e&&e.message?e.message:"";
        const friendly=buildFriendlyStartError(raw);
        showStatus(t("process.startFailed")+": "+friendly+" - "+t("process.checkLogs"),"error");
        if(isPicoclawNotFoundError(raw))showPersistentNotice(friendly,"process.startFailed");
        pendingAction=null;
        restoreNormalPolling();
        updateRunStopButton(false);

    }

}async function stopGateway() {
    setButtonLoading("stop");
    pendingAction= {
        type:"stop",retries:0
    };
    clearInterval(processPolling);
    processPolling=setInterval(checkProcessStatus,1000);
    try {
        const res=await fetch("/api/process/stop", {
            method:"POST"
        });
        if(!res.ok)throw new Error(await res.text());

    }
    catch(e) {
        showStatus(t("process.stopFailed")+": "+e.message,"error");
        pendingAction=null;
        restoreNormalPolling();
        updateRunStopButton(true);

    }

}let logOffset=0,logRunID=-1,logAutoScrollEnabled=true,logHasContent=false;
let autoFormatTimer=null,isAutoFormatting=false;
function handleLogData(data) {
    const o=document.getElementById("logOutput"),p=document.getElementById("logPlaceholder");
    if(!o)return;
    const run=data.log_run_id,src=data.log_source,lines=data.logs||[],total=data.log_total||0;
    if(src==="none") {
        if(!logHasContent&&p) {
            p.textContent=gatewayRunning?t("logs.noCapture"):t("logs.noLogs");
            p.style.display="";

        }
        return;

    }
    if(run!==logRunID&&logRunID!==-1) {
        o.textContent="";
        logHasContent=false;

    }
    logRunID=run;
    if(lines.length>0) {
        if(p)p.style.display="none";
        if(!logHasContent)o.textContent="";
        o.textContent+=lines.join("\n")+"\n";
        logHasContent=true;
        if(logAutoScrollEnabled)o.scrollTop=o.scrollHeight;

    }
    logOffset=total;
    if(!logHasContent&&p) {
        p.textContent=t("logs.noLogs");
        p.style.display="";

    }

}function clearLogDisplay() {
    const o=document.getElementById("logOutput"),p=document.getElementById("logPlaceholder");
    if(o)o.textContent="";
    logHasContent=false;
    if(o&&p) {
        o.appendChild(p);
        p.textContent=t("logs.noLogs");
        p.style.display="";

    }

}function esc(s) {
    const d=document.createElement("div");
    d.textContent=s;
    return d.innerHTML;

}function maskKey(k) {
    if(!k||k.length<8)return"****";
    return k.substring(0,4)+"..."+k.substring(k.length-4);

}function scheduleAutoFormatJson() {
    if(!document.getElementById("editor"))return;
    if(autoFormatTimer)clearTimeout(autoFormatTimer);
    autoFormatTimer=setTimeout(()=> {
        runAutoFormatJson(false);
    },350);

}function runAutoFormatJson(force) {
    if(!document.getElementById("editor")||isAutoFormatting)return;
    const raw=getEditorValue();
    if(!raw||!raw.trim())return;
    try {
        const parsed=JSON.parse(raw);
        const formatted=JSON.stringify(parsed,null,2);
        if(force||formatted!==raw) {
            isAutoFormatting=true;
            const pos=getEditorCursorPos();
            setEditorValue(formatted);
            setEditorCursorPos(Math.min(pos,formatted.length));
            isAutoFormatting=false;
        }
        validateJson();
    }
    catch(_) {
        // parsing failed: keep current raw input and do not format/highlight
    }

}function initEventBindings() {
    const burger=document.getElementById("navbarBurger"),menu=document.getElementById("navbarMenu"),sidebarBackdrop=document.getElementById("sidebarBackdrop");
    if(burger)burger.addEventListener("click",()=> {
        toggleNavbarMenu();
        if(isMobileViewport())toggleMobileSidebar();

    });
    if(sidebarBackdrop)sidebarBackdrop.addEventListener("click",closeMobileSidebar);
    document.getElementById("btnTheme").addEventListener("click",cycleTheme);
    const langSelect=document.getElementById("langSelect");
    if(langSelect)langSelect.addEventListener("change",e=>setLanguage(e.target.value));
    const persistentNoticeClose=document.getElementById("persistentNoticeClose");
    if(persistentNoticeClose)persistentNoticeClose.addEventListener("click",hidePersistentNotice);
    document.getElementById("btnRunStop").addEventListener("click",()=> {
        if(gatewayRunning)stopGateway();
        else startGateway();

    });
    document.getElementById("btnAddModel").addEventListener("click",showAddModelModal);
    document.getElementById("btnReload").addEventListener("click",loadConfig);
    document.getElementById("btnFormat").addEventListener("click",formatJson);
    document.getElementById("btnSaveRaw").addEventListener("click",saveRawConfig);
    document.getElementById("btnClearLogs").addEventListener("click",clearLogDisplay);
    document.getElementById("logAutoScrollCb").addEventListener("change",e=> {
        logAutoScrollEnabled=e.target.checked;

    });
    document.getElementById("sidebar").addEventListener("click",e=> {
        const g=e.target.closest('[data-action="toggle-group"]');
        if(g) {
            e.preventDefault();
            toggleGroup(g);
            return;

        }
        const item=e.target.closest(".sidebar-item[data-panel]");
        if(item) {
            e.preventDefault();
            activatePanel(item.dataset.panel);
            if(isMobileViewport()&&menu&&menu.classList.contains("is-active"))toggleNavbarMenu();

        }

    });
    document.getElementById("sidebar").addEventListener("keydown",e=> {
        const actionable=e.target.closest('.sidebar-item[data-panel], [data-action="toggle-group"]');
        if(!actionable)return;
        if(e.key==="Enter"||e.key===" ") {
            e.preventDefault();
            actionable.click();

        }

    });
    document.getElementById("modelGrid").addEventListener("click",e=> {
        const b=e.target.closest("[data-action]");
        if(!b)return;
        const idx=parseInt(b.dataset.index||"-1",10);
        if(b.dataset.action==="edit-model")showEditModelModal(idx);
        if(b.dataset.action==="set-primary-model")setPrimaryModel(idx);
        if(b.dataset.action==="delete-model")deleteModel(idx);

    });
    document.getElementById("modelModal").addEventListener("click",e=> {
        const a=e.target.closest("[data-action]");
        if(!a)return;
        if(a.dataset.action==="close-model-modal")closeModelModal();
        if(a.dataset.action==="save-model-modal")saveModelFromModal();

    });
    document.getElementById("contentArea").addEventListener("click",e=> {
        const a=e.target.closest("[data-action]");
        if(!a)return;
        const x=a.dataset.action;
        if(x==="toggle-channel-enabled"||x==="toggle-field-switch") {
            toggleSwitchElement(a);
            return;

        }
        if(x==="add-array-row") {
            const c=a.closest(".array-editor");
            if(c)addArrayRow(c);
            return;

        }
        if(x==="remove-array-row") {
            const r=a.closest(".array-row");
            if(r)r.remove();
            return;

        }
        if(x==="save-channel") {
            saveChannelForm(a.dataset.chkey);
            return;

        }
        if(x==="login-provider")loginProvider(a.dataset.provider);
        if(x==="logout-provider")logoutProvider(a.dataset.provider);
        if(x==="show-token-input")showTokenInput(a.dataset.provider);
        if(x==="submit-token")submitToken(a.dataset.provider);
        if(x==="cancel-token")loadAuthStatus();

    });
    document.getElementById("contentArea").addEventListener("keydown",e=> {
        const sw=e.target.closest('.switch[data-action="toggle-channel-enabled"], .switch[data-action="toggle-field-switch"]');
        if(!sw)return;
        if(e.key==="Enter"||e.key===" ") {
            e.preventDefault();
            toggleSwitchElement(sw);

        }

    });
    window.addEventListener("resize",()=> {
        if(!isMobileViewport())closeMobileSidebar();

    });
    document.addEventListener("keydown",e=> {
        if((e.ctrlKey||e.metaKey)&&e.key==="s") {
            e.preventDefault();
            if(document.getElementById("panelRawJson").classList.contains("active"))saveRawConfig();
            else saveConfig();

        }
        if(e.key==="Escape") {
            closeMobileSidebar();
            if(menu&&menu.classList.contains("is-active")&&isMobileViewport())toggleNavbarMenu();

        }

    });

}async function init() {
    await loadI18nData();
    currentLang=resolveInitialLanguage();
    localStorage.setItem("picoclaw-lang",currentLang);
    checkI18nCompleteness();
    populateLanguageSelect();
    await initJsonCodeEditor();
    applyI18n();
    document.querySelectorAll('.sidebar-item[data-panel]').forEach(el=> {
        el.setAttribute("role","button");
        el.setAttribute("tabindex","0");

    });
    document.querySelectorAll('[data-action="toggle-group"]').forEach(el=> {
        el.setAttribute("role","button");
        el.setAttribute("tabindex","0");

    });
    initEventBindings();
    loadConfig();
    loadAuthStatus().then(()=>renderModels());
    checkProcessStatus();
    processPolling=setInterval(checkProcessStatus,5000);
    if(window.location.hash==="#auth") {
        activatePanel("panelAuth");
        window.location.hash="";

    }

}if(document.readyState==="loading")document.addEventListener("DOMContentLoaded",()=> {
    init();
});
else init();
