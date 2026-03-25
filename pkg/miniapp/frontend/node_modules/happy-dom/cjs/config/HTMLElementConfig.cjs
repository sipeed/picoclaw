"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const HTMLElementConfigContentModelEnum_js_1 = __importDefault(require("./HTMLElementConfigContentModelEnum.cjs"));
/**
 * @see https://html.spec.whatwg.org/multipage/indices.html
 */
exports.default = {
    a: {
        className: 'HTMLAnchorElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.noSelfDescendants
    },
    abbr: {
        className: 'HTMLElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.anyDescendants
    },
    address: {
        className: 'HTMLElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.anyDescendants
    },
    area: {
        className: 'HTMLAreaElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.noDescendants
    },
    article: {
        className: 'HTMLElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.anyDescendants
    },
    aside: {
        className: 'HTMLElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.anyDescendants
    },
    audio: {
        className: 'HTMLAudioElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.anyDescendants
    },
    b: {
        className: 'HTMLElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.anyDescendants
    },
    base: {
        className: 'HTMLBaseElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.noDescendants
    },
    bdi: {
        className: 'HTMLElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.anyDescendants
    },
    bdo: {
        className: 'HTMLElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.anyDescendants
    },
    body: {
        className: 'HTMLBodyElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.anyDescendants
    },
    template: {
        className: 'HTMLTemplateElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.anyDescendants
    },
    form: {
        className: 'HTMLFormElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.anyDescendants
    },
    input: {
        className: 'HTMLInputElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.noDescendants
    },
    textarea: {
        className: 'HTMLTextAreaElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.anyDescendants
    },
    script: {
        className: 'HTMLScriptElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.rawText
    },
    img: {
        className: 'HTMLImageElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.noDescendants
    },
    link: {
        className: 'HTMLLinkElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.noDescendants
    },
    style: {
        className: 'HTMLStyleElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.rawText
    },
    label: {
        className: 'HTMLLabelElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.anyDescendants
    },
    slot: {
        className: 'HTMLSlotElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.anyDescendants
    },
    meta: {
        className: 'HTMLMetaElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.noDescendants
    },
    blockquote: {
        className: 'HTMLQuoteElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.anyDescendants
    },
    br: {
        className: 'HTMLBRElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.noDescendants
    },
    button: {
        className: 'HTMLButtonElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.anyDescendants
    },
    canvas: {
        className: 'HTMLCanvasElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.anyDescendants
    },
    caption: {
        className: 'HTMLTableCaptionElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.textOrComments
    },
    cite: {
        className: 'HTMLElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.anyDescendants
    },
    code: {
        className: 'HTMLElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.anyDescendants
    },
    col: {
        className: 'HTMLTableColElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.noDescendants,
        permittedParents: ['colgroup']
    },
    colgroup: {
        className: 'HTMLTableColElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.permittedDescendants,
        permittedDescendants: ['col']
    },
    data: {
        className: 'HTMLDataElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.anyDescendants
    },
    datalist: {
        className: 'HTMLDataListElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.anyDescendants
    },
    dd: {
        className: 'HTMLElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.noForbiddenFirstLevelDescendants,
        forbiddenDescendants: ['dt', 'dd']
    },
    del: {
        className: 'HTMLModElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.anyDescendants
    },
    details: {
        className: 'HTMLDetailsElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.anyDescendants
    },
    dfn: {
        className: 'HTMLElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.anyDescendants
    },
    dialog: {
        className: 'HTMLDialogElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.anyDescendants
    },
    div: {
        className: 'HTMLDivElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.anyDescendants
    },
    dl: {
        className: 'HTMLDListElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.anyDescendants
    },
    dt: {
        className: 'HTMLElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.noForbiddenFirstLevelDescendants,
        forbiddenDescendants: ['dt', 'dd']
    },
    em: {
        className: 'HTMLElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.anyDescendants
    },
    embed: {
        className: 'HTMLEmbedElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.noDescendants
    },
    fieldset: {
        className: 'HTMLFieldSetElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.anyDescendants
    },
    figcaption: {
        className: 'HTMLElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.anyDescendants
    },
    figure: {
        className: 'HTMLElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.anyDescendants
    },
    footer: {
        className: 'HTMLElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.anyDescendants
    },
    h1: {
        className: 'HTMLHeadingElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.noFirstLevelSelfDescendants
    },
    h2: {
        className: 'HTMLHeadingElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.noFirstLevelSelfDescendants
    },
    h3: {
        className: 'HTMLHeadingElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.noFirstLevelSelfDescendants
    },
    h4: {
        className: 'HTMLHeadingElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.noFirstLevelSelfDescendants
    },
    h5: {
        className: 'HTMLHeadingElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.noFirstLevelSelfDescendants
    },
    h6: {
        className: 'HTMLHeadingElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.noFirstLevelSelfDescendants
    },
    head: {
        className: 'HTMLHeadElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.anyDescendants
    },
    header: {
        className: 'HTMLElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.anyDescendants
    },
    hgroup: {
        className: 'HTMLElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.anyDescendants
    },
    hr: {
        className: 'HTMLHRElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.noDescendants
    },
    html: {
        className: 'HTMLHtmlElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.anyDescendants
    },
    i: {
        className: 'HTMLElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.anyDescendants
    },
    iframe: {
        className: 'HTMLIFrameElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.anyDescendants
    },
    ins: {
        className: 'HTMLModElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.anyDescendants
    },
    kbd: {
        className: 'HTMLElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.anyDescendants
    },
    legend: {
        className: 'HTMLLegendElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.anyDescendants
    },
    li: {
        className: 'HTMLLIElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.noFirstLevelSelfDescendants
    },
    main: {
        className: 'HTMLElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.anyDescendants
    },
    map: {
        className: 'HTMLMapElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.anyDescendants
    },
    mark: {
        className: 'HTMLElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.anyDescendants
    },
    menu: {
        className: 'HTMLMenuElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.anyDescendants
    },
    meter: {
        className: 'HTMLMeterElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.anyDescendants
    },
    nav: {
        className: 'HTMLElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.anyDescendants
    },
    noscript: {
        className: 'HTMLElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.anyDescendants
    },
    object: {
        className: 'HTMLObjectElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.anyDescendants
    },
    ol: {
        className: 'HTMLOListElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.anyDescendants
    },
    optgroup: {
        className: 'HTMLOptGroupElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.noFirstLevelSelfDescendants
    },
    option: {
        className: 'HTMLOptionElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.noForbiddenFirstLevelDescendants,
        forbiddenDescendants: ['option', 'optgroup']
    },
    output: {
        className: 'HTMLOutputElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.anyDescendants
    },
    p: {
        className: 'HTMLParagraphElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.anyDescendants
    },
    param: {
        className: 'HTMLParamElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.noDescendants
    },
    picture: {
        className: 'HTMLPictureElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.anyDescendants
    },
    pre: {
        className: 'HTMLPreElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.anyDescendants
    },
    progress: {
        className: 'HTMLProgressElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.anyDescendants
    },
    q: {
        className: 'HTMLQuoteElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.anyDescendants
    },
    rb: {
        className: 'HTMLElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.anyDescendants
    },
    rp: {
        className: 'HTMLElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.noForbiddenFirstLevelDescendants,
        forbiddenDescendants: ['rp', 'rt']
    },
    rt: {
        className: 'HTMLElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.noForbiddenFirstLevelDescendants,
        forbiddenDescendants: ['rp', 'rt']
    },
    rtc: {
        className: 'HTMLElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.anyDescendants
    },
    ruby: {
        className: 'HTMLElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.anyDescendants
    },
    s: {
        className: 'HTMLElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.anyDescendants
    },
    samp: {
        className: 'HTMLElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.anyDescendants
    },
    section: {
        className: 'HTMLElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.anyDescendants
    },
    select: {
        className: 'HTMLSelectElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.anyDescendants
    },
    small: {
        className: 'HTMLElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.anyDescendants
    },
    source: {
        className: 'HTMLSourceElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.noDescendants
    },
    span: {
        className: 'HTMLSpanElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.anyDescendants
    },
    strong: {
        className: 'HTMLElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.anyDescendants
    },
    sub: {
        className: 'HTMLElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.anyDescendants
    },
    summary: {
        className: 'HTMLElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.anyDescendants
    },
    sup: {
        className: 'HTMLElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.anyDescendants
    },
    table: {
        className: 'HTMLTableElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.permittedDescendants,
        permittedDescendants: ['caption', 'colgroup', 'thead', 'tfoot', 'tbody'],
        moveForbiddenDescendant: { exclude: [] }
    },
    tbody: {
        className: 'HTMLTableSectionElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.permittedDescendants,
        permittedDescendants: ['tr'],
        permittedParents: ['table'],
        moveForbiddenDescendant: { exclude: ['caption', 'colgroup', 'thead', 'tfoot', 'tbody'] }
    },
    td: {
        className: 'HTMLTableCellElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.noForbiddenFirstLevelDescendants,
        forbiddenDescendants: ['td', 'th', 'tr', 'tbody', 'tfoot', 'thead'],
        permittedParents: ['tr']
    },
    tfoot: {
        className: 'HTMLTableSectionElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.permittedDescendants,
        permittedDescendants: ['tr'],
        permittedParents: ['table'],
        moveForbiddenDescendant: { exclude: ['caption', 'colgroup', 'thead', 'tfoot', 'tbody'] }
    },
    th: {
        className: 'HTMLTableCellElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.noForbiddenFirstLevelDescendants,
        forbiddenDescendants: ['td', 'th', 'tr', 'tbody', 'tfoot', 'thead'],
        permittedParents: ['tr']
    },
    thead: {
        className: 'HTMLTableSectionElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.permittedDescendants,
        permittedDescendants: ['tr'],
        permittedParents: ['table'],
        moveForbiddenDescendant: { exclude: ['caption', 'colgroup', 'thead', 'tfoot', 'tbody'] }
    },
    time: {
        className: 'HTMLTimeElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.anyDescendants
    },
    title: {
        className: 'HTMLTitleElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.anyDescendants
    },
    tr: {
        className: 'HTMLTableRowElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.permittedDescendants,
        permittedDescendants: ['td', 'th'],
        permittedParents: ['tbody', 'tfoot', 'thead'],
        addPermittedParent: 'tbody',
        moveForbiddenDescendant: { exclude: ['caption', 'colgroup', 'thead', 'tfoot', 'tbody', 'tr'] }
    },
    track: {
        className: 'HTMLTrackElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.noDescendants
    },
    u: {
        className: 'HTMLElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.anyDescendants
    },
    ul: {
        className: 'HTMLUListElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.anyDescendants
    },
    var: {
        className: 'HTMLElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.anyDescendants
    },
    video: {
        className: 'HTMLVideoElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.anyDescendants
    },
    wbr: {
        className: 'HTMLElement',
        contentModel: HTMLElementConfigContentModelEnum_js_1.default.noDescendants
    }
};
//# sourceMappingURL=HTMLElementConfig.cjs.map