import HTMLElementConfigContentModelEnum from './HTMLElementConfigContentModelEnum.js';
/**
 * @see https://html.spec.whatwg.org/multipage/indices.html
 */
export default {
    a: {
        className: 'HTMLAnchorElement',
        contentModel: HTMLElementConfigContentModelEnum.noSelfDescendants
    },
    abbr: {
        className: 'HTMLElement',
        contentModel: HTMLElementConfigContentModelEnum.anyDescendants
    },
    address: {
        className: 'HTMLElement',
        contentModel: HTMLElementConfigContentModelEnum.anyDescendants
    },
    area: {
        className: 'HTMLAreaElement',
        contentModel: HTMLElementConfigContentModelEnum.noDescendants
    },
    article: {
        className: 'HTMLElement',
        contentModel: HTMLElementConfigContentModelEnum.anyDescendants
    },
    aside: {
        className: 'HTMLElement',
        contentModel: HTMLElementConfigContentModelEnum.anyDescendants
    },
    audio: {
        className: 'HTMLAudioElement',
        contentModel: HTMLElementConfigContentModelEnum.anyDescendants
    },
    b: {
        className: 'HTMLElement',
        contentModel: HTMLElementConfigContentModelEnum.anyDescendants
    },
    base: {
        className: 'HTMLBaseElement',
        contentModel: HTMLElementConfigContentModelEnum.noDescendants
    },
    bdi: {
        className: 'HTMLElement',
        contentModel: HTMLElementConfigContentModelEnum.anyDescendants
    },
    bdo: {
        className: 'HTMLElement',
        contentModel: HTMLElementConfigContentModelEnum.anyDescendants
    },
    body: {
        className: 'HTMLBodyElement',
        contentModel: HTMLElementConfigContentModelEnum.anyDescendants
    },
    template: {
        className: 'HTMLTemplateElement',
        contentModel: HTMLElementConfigContentModelEnum.anyDescendants
    },
    form: {
        className: 'HTMLFormElement',
        contentModel: HTMLElementConfigContentModelEnum.anyDescendants
    },
    input: {
        className: 'HTMLInputElement',
        contentModel: HTMLElementConfigContentModelEnum.noDescendants
    },
    textarea: {
        className: 'HTMLTextAreaElement',
        contentModel: HTMLElementConfigContentModelEnum.anyDescendants
    },
    script: {
        className: 'HTMLScriptElement',
        contentModel: HTMLElementConfigContentModelEnum.rawText
    },
    img: {
        className: 'HTMLImageElement',
        contentModel: HTMLElementConfigContentModelEnum.noDescendants
    },
    link: {
        className: 'HTMLLinkElement',
        contentModel: HTMLElementConfigContentModelEnum.noDescendants
    },
    style: {
        className: 'HTMLStyleElement',
        contentModel: HTMLElementConfigContentModelEnum.rawText
    },
    label: {
        className: 'HTMLLabelElement',
        contentModel: HTMLElementConfigContentModelEnum.anyDescendants
    },
    slot: {
        className: 'HTMLSlotElement',
        contentModel: HTMLElementConfigContentModelEnum.anyDescendants
    },
    meta: {
        className: 'HTMLMetaElement',
        contentModel: HTMLElementConfigContentModelEnum.noDescendants
    },
    blockquote: {
        className: 'HTMLQuoteElement',
        contentModel: HTMLElementConfigContentModelEnum.anyDescendants
    },
    br: {
        className: 'HTMLBRElement',
        contentModel: HTMLElementConfigContentModelEnum.noDescendants
    },
    button: {
        className: 'HTMLButtonElement',
        contentModel: HTMLElementConfigContentModelEnum.anyDescendants
    },
    canvas: {
        className: 'HTMLCanvasElement',
        contentModel: HTMLElementConfigContentModelEnum.anyDescendants
    },
    caption: {
        className: 'HTMLTableCaptionElement',
        contentModel: HTMLElementConfigContentModelEnum.textOrComments
    },
    cite: {
        className: 'HTMLElement',
        contentModel: HTMLElementConfigContentModelEnum.anyDescendants
    },
    code: {
        className: 'HTMLElement',
        contentModel: HTMLElementConfigContentModelEnum.anyDescendants
    },
    col: {
        className: 'HTMLTableColElement',
        contentModel: HTMLElementConfigContentModelEnum.noDescendants,
        permittedParents: ['colgroup']
    },
    colgroup: {
        className: 'HTMLTableColElement',
        contentModel: HTMLElementConfigContentModelEnum.permittedDescendants,
        permittedDescendants: ['col']
    },
    data: {
        className: 'HTMLDataElement',
        contentModel: HTMLElementConfigContentModelEnum.anyDescendants
    },
    datalist: {
        className: 'HTMLDataListElement',
        contentModel: HTMLElementConfigContentModelEnum.anyDescendants
    },
    dd: {
        className: 'HTMLElement',
        contentModel: HTMLElementConfigContentModelEnum.noForbiddenFirstLevelDescendants,
        forbiddenDescendants: ['dt', 'dd']
    },
    del: {
        className: 'HTMLModElement',
        contentModel: HTMLElementConfigContentModelEnum.anyDescendants
    },
    details: {
        className: 'HTMLDetailsElement',
        contentModel: HTMLElementConfigContentModelEnum.anyDescendants
    },
    dfn: {
        className: 'HTMLElement',
        contentModel: HTMLElementConfigContentModelEnum.anyDescendants
    },
    dialog: {
        className: 'HTMLDialogElement',
        contentModel: HTMLElementConfigContentModelEnum.anyDescendants
    },
    div: {
        className: 'HTMLDivElement',
        contentModel: HTMLElementConfigContentModelEnum.anyDescendants
    },
    dl: {
        className: 'HTMLDListElement',
        contentModel: HTMLElementConfigContentModelEnum.anyDescendants
    },
    dt: {
        className: 'HTMLElement',
        contentModel: HTMLElementConfigContentModelEnum.noForbiddenFirstLevelDescendants,
        forbiddenDescendants: ['dt', 'dd']
    },
    em: {
        className: 'HTMLElement',
        contentModel: HTMLElementConfigContentModelEnum.anyDescendants
    },
    embed: {
        className: 'HTMLEmbedElement',
        contentModel: HTMLElementConfigContentModelEnum.noDescendants
    },
    fieldset: {
        className: 'HTMLFieldSetElement',
        contentModel: HTMLElementConfigContentModelEnum.anyDescendants
    },
    figcaption: {
        className: 'HTMLElement',
        contentModel: HTMLElementConfigContentModelEnum.anyDescendants
    },
    figure: {
        className: 'HTMLElement',
        contentModel: HTMLElementConfigContentModelEnum.anyDescendants
    },
    footer: {
        className: 'HTMLElement',
        contentModel: HTMLElementConfigContentModelEnum.anyDescendants
    },
    h1: {
        className: 'HTMLHeadingElement',
        contentModel: HTMLElementConfigContentModelEnum.noFirstLevelSelfDescendants
    },
    h2: {
        className: 'HTMLHeadingElement',
        contentModel: HTMLElementConfigContentModelEnum.noFirstLevelSelfDescendants
    },
    h3: {
        className: 'HTMLHeadingElement',
        contentModel: HTMLElementConfigContentModelEnum.noFirstLevelSelfDescendants
    },
    h4: {
        className: 'HTMLHeadingElement',
        contentModel: HTMLElementConfigContentModelEnum.noFirstLevelSelfDescendants
    },
    h5: {
        className: 'HTMLHeadingElement',
        contentModel: HTMLElementConfigContentModelEnum.noFirstLevelSelfDescendants
    },
    h6: {
        className: 'HTMLHeadingElement',
        contentModel: HTMLElementConfigContentModelEnum.noFirstLevelSelfDescendants
    },
    head: {
        className: 'HTMLHeadElement',
        contentModel: HTMLElementConfigContentModelEnum.anyDescendants
    },
    header: {
        className: 'HTMLElement',
        contentModel: HTMLElementConfigContentModelEnum.anyDescendants
    },
    hgroup: {
        className: 'HTMLElement',
        contentModel: HTMLElementConfigContentModelEnum.anyDescendants
    },
    hr: {
        className: 'HTMLHRElement',
        contentModel: HTMLElementConfigContentModelEnum.noDescendants
    },
    html: {
        className: 'HTMLHtmlElement',
        contentModel: HTMLElementConfigContentModelEnum.anyDescendants
    },
    i: {
        className: 'HTMLElement',
        contentModel: HTMLElementConfigContentModelEnum.anyDescendants
    },
    iframe: {
        className: 'HTMLIFrameElement',
        contentModel: HTMLElementConfigContentModelEnum.anyDescendants
    },
    ins: {
        className: 'HTMLModElement',
        contentModel: HTMLElementConfigContentModelEnum.anyDescendants
    },
    kbd: {
        className: 'HTMLElement',
        contentModel: HTMLElementConfigContentModelEnum.anyDescendants
    },
    legend: {
        className: 'HTMLLegendElement',
        contentModel: HTMLElementConfigContentModelEnum.anyDescendants
    },
    li: {
        className: 'HTMLLIElement',
        contentModel: HTMLElementConfigContentModelEnum.noFirstLevelSelfDescendants
    },
    main: {
        className: 'HTMLElement',
        contentModel: HTMLElementConfigContentModelEnum.anyDescendants
    },
    map: {
        className: 'HTMLMapElement',
        contentModel: HTMLElementConfigContentModelEnum.anyDescendants
    },
    mark: {
        className: 'HTMLElement',
        contentModel: HTMLElementConfigContentModelEnum.anyDescendants
    },
    menu: {
        className: 'HTMLMenuElement',
        contentModel: HTMLElementConfigContentModelEnum.anyDescendants
    },
    meter: {
        className: 'HTMLMeterElement',
        contentModel: HTMLElementConfigContentModelEnum.anyDescendants
    },
    nav: {
        className: 'HTMLElement',
        contentModel: HTMLElementConfigContentModelEnum.anyDescendants
    },
    noscript: {
        className: 'HTMLElement',
        contentModel: HTMLElementConfigContentModelEnum.anyDescendants
    },
    object: {
        className: 'HTMLObjectElement',
        contentModel: HTMLElementConfigContentModelEnum.anyDescendants
    },
    ol: {
        className: 'HTMLOListElement',
        contentModel: HTMLElementConfigContentModelEnum.anyDescendants
    },
    optgroup: {
        className: 'HTMLOptGroupElement',
        contentModel: HTMLElementConfigContentModelEnum.noFirstLevelSelfDescendants
    },
    option: {
        className: 'HTMLOptionElement',
        contentModel: HTMLElementConfigContentModelEnum.noForbiddenFirstLevelDescendants,
        forbiddenDescendants: ['option', 'optgroup']
    },
    output: {
        className: 'HTMLOutputElement',
        contentModel: HTMLElementConfigContentModelEnum.anyDescendants
    },
    p: {
        className: 'HTMLParagraphElement',
        contentModel: HTMLElementConfigContentModelEnum.anyDescendants
    },
    param: {
        className: 'HTMLParamElement',
        contentModel: HTMLElementConfigContentModelEnum.noDescendants
    },
    picture: {
        className: 'HTMLPictureElement',
        contentModel: HTMLElementConfigContentModelEnum.anyDescendants
    },
    pre: {
        className: 'HTMLPreElement',
        contentModel: HTMLElementConfigContentModelEnum.anyDescendants
    },
    progress: {
        className: 'HTMLProgressElement',
        contentModel: HTMLElementConfigContentModelEnum.anyDescendants
    },
    q: {
        className: 'HTMLQuoteElement',
        contentModel: HTMLElementConfigContentModelEnum.anyDescendants
    },
    rb: {
        className: 'HTMLElement',
        contentModel: HTMLElementConfigContentModelEnum.anyDescendants
    },
    rp: {
        className: 'HTMLElement',
        contentModel: HTMLElementConfigContentModelEnum.noForbiddenFirstLevelDescendants,
        forbiddenDescendants: ['rp', 'rt']
    },
    rt: {
        className: 'HTMLElement',
        contentModel: HTMLElementConfigContentModelEnum.noForbiddenFirstLevelDescendants,
        forbiddenDescendants: ['rp', 'rt']
    },
    rtc: {
        className: 'HTMLElement',
        contentModel: HTMLElementConfigContentModelEnum.anyDescendants
    },
    ruby: {
        className: 'HTMLElement',
        contentModel: HTMLElementConfigContentModelEnum.anyDescendants
    },
    s: {
        className: 'HTMLElement',
        contentModel: HTMLElementConfigContentModelEnum.anyDescendants
    },
    samp: {
        className: 'HTMLElement',
        contentModel: HTMLElementConfigContentModelEnum.anyDescendants
    },
    section: {
        className: 'HTMLElement',
        contentModel: HTMLElementConfigContentModelEnum.anyDescendants
    },
    select: {
        className: 'HTMLSelectElement',
        contentModel: HTMLElementConfigContentModelEnum.anyDescendants
    },
    small: {
        className: 'HTMLElement',
        contentModel: HTMLElementConfigContentModelEnum.anyDescendants
    },
    source: {
        className: 'HTMLSourceElement',
        contentModel: HTMLElementConfigContentModelEnum.noDescendants
    },
    span: {
        className: 'HTMLSpanElement',
        contentModel: HTMLElementConfigContentModelEnum.anyDescendants
    },
    strong: {
        className: 'HTMLElement',
        contentModel: HTMLElementConfigContentModelEnum.anyDescendants
    },
    sub: {
        className: 'HTMLElement',
        contentModel: HTMLElementConfigContentModelEnum.anyDescendants
    },
    summary: {
        className: 'HTMLElement',
        contentModel: HTMLElementConfigContentModelEnum.anyDescendants
    },
    sup: {
        className: 'HTMLElement',
        contentModel: HTMLElementConfigContentModelEnum.anyDescendants
    },
    table: {
        className: 'HTMLTableElement',
        contentModel: HTMLElementConfigContentModelEnum.permittedDescendants,
        permittedDescendants: ['caption', 'colgroup', 'thead', 'tfoot', 'tbody'],
        moveForbiddenDescendant: { exclude: [] }
    },
    tbody: {
        className: 'HTMLTableSectionElement',
        contentModel: HTMLElementConfigContentModelEnum.permittedDescendants,
        permittedDescendants: ['tr'],
        permittedParents: ['table'],
        moveForbiddenDescendant: { exclude: ['caption', 'colgroup', 'thead', 'tfoot', 'tbody'] }
    },
    td: {
        className: 'HTMLTableCellElement',
        contentModel: HTMLElementConfigContentModelEnum.noForbiddenFirstLevelDescendants,
        forbiddenDescendants: ['td', 'th', 'tr', 'tbody', 'tfoot', 'thead'],
        permittedParents: ['tr']
    },
    tfoot: {
        className: 'HTMLTableSectionElement',
        contentModel: HTMLElementConfigContentModelEnum.permittedDescendants,
        permittedDescendants: ['tr'],
        permittedParents: ['table'],
        moveForbiddenDescendant: { exclude: ['caption', 'colgroup', 'thead', 'tfoot', 'tbody'] }
    },
    th: {
        className: 'HTMLTableCellElement',
        contentModel: HTMLElementConfigContentModelEnum.noForbiddenFirstLevelDescendants,
        forbiddenDescendants: ['td', 'th', 'tr', 'tbody', 'tfoot', 'thead'],
        permittedParents: ['tr']
    },
    thead: {
        className: 'HTMLTableSectionElement',
        contentModel: HTMLElementConfigContentModelEnum.permittedDescendants,
        permittedDescendants: ['tr'],
        permittedParents: ['table'],
        moveForbiddenDescendant: { exclude: ['caption', 'colgroup', 'thead', 'tfoot', 'tbody'] }
    },
    time: {
        className: 'HTMLTimeElement',
        contentModel: HTMLElementConfigContentModelEnum.anyDescendants
    },
    title: {
        className: 'HTMLTitleElement',
        contentModel: HTMLElementConfigContentModelEnum.anyDescendants
    },
    tr: {
        className: 'HTMLTableRowElement',
        contentModel: HTMLElementConfigContentModelEnum.permittedDescendants,
        permittedDescendants: ['td', 'th'],
        permittedParents: ['tbody', 'tfoot', 'thead'],
        addPermittedParent: 'tbody',
        moveForbiddenDescendant: { exclude: ['caption', 'colgroup', 'thead', 'tfoot', 'tbody', 'tr'] }
    },
    track: {
        className: 'HTMLTrackElement',
        contentModel: HTMLElementConfigContentModelEnum.noDescendants
    },
    u: {
        className: 'HTMLElement',
        contentModel: HTMLElementConfigContentModelEnum.anyDescendants
    },
    ul: {
        className: 'HTMLUListElement',
        contentModel: HTMLElementConfigContentModelEnum.anyDescendants
    },
    var: {
        className: 'HTMLElement',
        contentModel: HTMLElementConfigContentModelEnum.anyDescendants
    },
    video: {
        className: 'HTMLVideoElement',
        contentModel: HTMLElementConfigContentModelEnum.anyDescendants
    },
    wbr: {
        className: 'HTMLElement',
        contentModel: HTMLElementConfigContentModelEnum.noDescendants
    }
};
//# sourceMappingURL=HTMLElementConfig.js.map