import Element from '../element/Element.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import BrowserWindow from '../../window/BrowserWindow.js';
import Node from '../node/Node.js';
import NodeIterator from '../../tree-walker/NodeIterator.js';
import TreeWalker from '../../tree-walker/TreeWalker.js';
import DocumentFragment from '../document-fragment/DocumentFragment.js';
import Event from '../../event/Event.js';
import DOMImplementation from '../../dom-implementation/DOMImplementation.js';
import INodeFilter from '../../tree-walker/INodeFilter.js';
import NamespaceURI from '../../config/NamespaceURI.js';
import DocumentType from '../document-type/DocumentType.js';
import ParentNodeUtility from '../parent-node/ParentNodeUtility.js';
import QuerySelector from '../../query-selector/QuerySelector.js';
import CSSStyleSheet from '../../css/CSSStyleSheet.js';
import HTMLScriptElement from '../html-script-element/HTMLScriptElement.js';
import HTMLElement from '../html-element/HTMLElement.js';
import Comment from '../comment/Comment.js';
import Text from '../text/Text.js';
import NodeList from '../node/NodeList.js';
import HTMLCollection from '../element/HTMLCollection.js';
import HTMLLinkElement from '../html-link-element/HTMLLinkElement.js';
import HTMLStyleElement from '../html-style-element/HTMLStyleElement.js';
import DocumentReadyStateEnum from './DocumentReadyStateEnum.js';
import Location from '../../location/Location.js';
import Selection from '../../selection/Selection.js';
import ShadowRoot from '../shadow-root/ShadowRoot.js';
import Range from '../../range/Range.js';
import Attr from '../attr/Attr.js';
import ProcessingInstruction from '../processing-instruction/ProcessingInstruction.js';
import VisibilityStateEnum from './VisibilityStateEnum.js';
import NodeTypeEnum from '../node/NodeTypeEnum.js';
import CookieStringUtility from '../../cookie/urilities/CookieStringUtility.js';
import { URL } from 'url';
import IHTMLElementTagNameMap from '../../config/IHTMLElementTagNameMap.js';
import ISVGElementTagNameMap from '../../config/ISVGElementTagNameMap.js';
import SVGElement from '../svg-element/SVGElement.js';
import HTMLFormElement from '../html-form-element/HTMLFormElement.js';
import HTMLAnchorElement from '../html-anchor-element/HTMLAnchorElement.js';
import HTMLElementConfig from '../../config/HTMLElementConfig.js';
import HTMLHtmlElement from '../html-html-element/HTMLHtmlElement.js';
import HTMLBodyElement from '../html-body-element/HTMLBodyElement.js';
import HTMLHeadElement from '../html-head-element/HTMLHeadElement.js';
import HTMLBaseElement from '../html-base-element/HTMLBaseElement.js';
import ICachedResult from '../node/ICachedResult.js';
import HTMLTitleElement from '../html-title-element/HTMLTitleElement.js';
import WindowBrowserContext from '../../window/WindowBrowserContext.js';
import NodeFactory from '../NodeFactory.js';
import SVGElementConfig from '../../config/SVGElementConfig.js';
import StringUtility from '../../utilities/StringUtility.js';
import HTMLParser from '../../html-parser/HTMLParser.js';

const PROCESSING_INSTRUCTION_TARGET_REGEXP = /^[a-z][a-z0-9-]+$/;

/**
 * Document.
 */
export default class Document extends Node {
	// Internal properties
	public [PropertySymbol.children]: HTMLCollection<Element> | null = null;
	public [PropertySymbol.activeElement]: HTMLElement | SVGElement = null;
	public [PropertySymbol.nextActiveElement]: HTMLElement | SVGElement = null;
	public [PropertySymbol.currentScript]: HTMLScriptElement = null;
	public [PropertySymbol.rootNode] = this;
	public [PropertySymbol.isFirstWrite] = true;
	public [PropertySymbol.isFirstWriteAfterOpen] = false;
	public [PropertySymbol.nodeType] = NodeTypeEnum.documentNode;
	public [PropertySymbol.isConnected] = true;
	public [PropertySymbol.adoptedStyleSheets]: CSSStyleSheet[] = [];
	public [PropertySymbol.implementation] = new DOMImplementation(this);
	public [PropertySymbol.readyState] = DocumentReadyStateEnum.interactive;
	public [PropertySymbol.referrer] = '';
	public [PropertySymbol.defaultView]: BrowserWindow | null = null;
	public [PropertySymbol.forms]: HTMLCollection<HTMLFormElement> | null = null;
	public [PropertySymbol.affectsComputedStyleCache]: ICachedResult[] = [];
	public [PropertySymbol.ownerDocument]: Document = <Document>(<unknown>null);
	public [PropertySymbol.elementIdMap]: Map<
		string,
		{ htmlCollection: HTMLCollection<Element> | null; elements: Element[] }
	> = new Map();
	public [PropertySymbol.contentType]: string = 'text/html';
	public [PropertySymbol.xmlProcessingInstruction]: ProcessingInstruction | null = null;
	public declare cloneNode: (deep?: boolean) => Document;

	// Private properties
	#selection: Selection = null;

	// Events
	public onreadystatechange: (event: Event) => void = null;
	public onpointerlockchange: (event: Event) => void = null;
	public onpointerlockerror: (event: Event) => void = null;
	public onbeforecopy: (event: Event) => void = null;
	public onbeforecut: (event: Event) => void = null;
	public onbeforepaste: (event: Event) => void = null;
	public onfreeze: (event: Event) => void = null;
	public onresume: (event: Event) => void = null;
	public onsearch: (event: Event) => void = null;
	public onvisibilitychange: (event: Event) => void = null;
	public onfullscreenchange: (event: Event) => void = null;
	public onfullscreenerror: (event: Event) => void = null;
	public onwebkitfullscreenchange: (event: Event) => void = null;
	public onwebkitfullscreenerror: (event: Event) => void = null;
	public onbeforexrselect: (event: Event) => void = null;
	public onabort: (event: Event) => void = null;
	public onbeforeinput: (event: Event) => void = null;
	public onblur: (event: Event) => void = null;
	public oncancel: (event: Event) => void = null;
	public oncanplay: (event: Event) => void = null;
	public oncanplaythrough: (event: Event) => void = null;
	public onchange: (event: Event) => void = null;
	public onclick: (event: Event) => void = null;
	public onclose: (event: Event) => void = null;
	public oncontextlost: (event: Event) => void = null;
	public oncontextmenu: (event: Event) => void = null;
	public oncontextrestored: (event: Event) => void = null;
	public oncuechange: (event: Event) => void = null;
	public ondblclick: (event: Event) => void = null;
	public ondrag: (event: Event) => void = null;
	public ondragend: (event: Event) => void = null;
	public ondragenter: (event: Event) => void = null;
	public ondragleave: (event: Event) => void = null;
	public ondragover: (event: Event) => void = null;
	public ondragstart: (event: Event) => void = null;
	public ondrop: (event: Event) => void = null;
	public ondurationchange: (event: Event) => void = null;
	public onemptied: (event: Event) => void = null;
	public onended: (event: Event) => void = null;
	public onerror: (event: Event) => void = null;
	public onfocus: (event: Event) => void = null;
	public onformdata: (event: Event) => void = null;
	public oninput: (event: Event) => void = null;
	public oninvalid: (event: Event) => void = null;
	public onkeydown: (event: Event) => void = null;
	public onkeypress: (event: Event) => void = null;
	public onkeyup: (event: Event) => void = null;
	public onload: (event: Event) => void = null;
	public onloadeddata: (event: Event) => void = null;
	public onloadedmetadata: (event: Event) => void = null;
	public onloadstart: (event: Event) => void = null;
	public onmousedown: (event: Event) => void = null;
	public onmouseenter: (event: Event) => void = null;
	public onmouseleave: (event: Event) => void = null;
	public onmousemove: (event: Event) => void = null;
	public onmouseout: (event: Event) => void = null;
	public onmouseover: (event: Event) => void = null;
	public onmouseup: (event: Event) => void = null;
	public onmousewheel: (event: Event) => void = null;
	public onpause: (event: Event) => void = null;
	public onplay: (event: Event) => void = null;
	public onplaying: (event: Event) => void = null;
	public onprogress: (event: Event) => void = null;
	public onratechange: (event: Event) => void = null;
	public onreset: (event: Event) => void = null;
	public onresize: (event: Event) => void = null;
	public onscroll: (event: Event) => void = null;
	public onsecuritypolicyviolation: (event: Event) => void = null;
	public onseeked: (event: Event) => void = null;
	public onseeking: (event: Event) => void = null;
	public onselect: (event: Event) => void = null;
	public onslotchange: (event: Event) => void = null;
	public onstalled: (event: Event) => void = null;
	public onsubmit: (event: Event) => void = null;
	public onsuspend: (event: Event) => void = null;
	public ontimeupdate: (event: Event) => void = null;
	public ontoggle: (event: Event) => void = null;
	public onvolumechange: (event: Event) => void = null;
	public onwaiting: (event: Event) => void = null;
	public onwebkitanimationend: (event: Event) => void = null;
	public onwebkitanimationiteration: (event: Event) => void = null;
	public onwebkitanimationstart: (event: Event) => void = null;
	public onwebkittransitionend: (event: Event) => void = null;
	public onwheel: (event: Event) => void = null;
	public onauxclick: (event: Event) => void = null;
	public ongotpointercapture: (event: Event) => void = null;
	public onlostpointercapture: (event: Event) => void = null;
	public onpointerdown: (event: Event) => void = null;
	public onpointermove: (event: Event) => void = null;
	public onpointerrawupdate: (event: Event) => void = null;
	public onpointerup: (event: Event) => void = null;
	public onpointercancel: (event: Event) => void = null;
	public onpointerover: (event: Event) => void = null;
	public onpointerout: (event: Event) => void = null;
	public onpointerenter: (event: Event) => void = null;
	public onpointerleave: (event: Event) => void = null;
	public onselectstart: (event: Event) => void = null;
	public onselectionchange: (event: Event) => void = null;
	public onanimationend: (event: Event) => void = null;
	public onanimationiteration: (event: Event) => void = null;
	public onanimationstart: (event: Event) => void = null;
	public ontransitionrun: (event: Event) => void = null;
	public ontransitionstart: (event: Event) => void = null;
	public ontransitionend: (event: Event) => void = null;
	public ontransitioncancel: (event: Event) => void = null;
	public oncopy: (event: Event) => void = null;
	public oncut: (event: Event) => void = null;
	public onpaste: (event: Event) => void = null;
	public onbeforematch: (event: Event) => void = null;

	/**
	 * Returns adopted style sheets.
	 *
	 * @returns Adopted style sheets.
	 */
	public get adoptedStyleSheets(): CSSStyleSheet[] {
		return this[PropertySymbol.adoptedStyleSheets];
	}

	/**
	 * Sets adopted style sheets.
	 *
	 * @param value Adopted style sheets.
	 */
	public set adoptedStyleSheets(value: CSSStyleSheet[]) {
		this[PropertySymbol.adoptedStyleSheets] = value;
	}

	/**
	 * Returns DOM implementation.
	 *
	 * @returns DOM implementation.
	 */
	public get implementation(): DOMImplementation {
		return this[PropertySymbol.implementation];
	}

	/**
	 * Returns document ready state.
	 *
	 * @returns Document ready state.
	 */
	public get readyState(): DocumentReadyStateEnum {
		return this[PropertySymbol.readyState];
	}

	/**
	 * Returns referrer.
	 *
	 * @returns Referrer.
	 */
	public get referrer(): string {
		return this[PropertySymbol.referrer];
	}

	/**
	 * Returns default view.
	 *
	 * @returns Default view.
	 */
	public get defaultView(): BrowserWindow | null {
		return this[PropertySymbol.defaultView];
	}

	/**
	 * Returns document children.
	 */
	public get children(): HTMLCollection<Element> {
		if (!this[PropertySymbol.children]) {
			const elements = this[PropertySymbol.elementArray];
			this[PropertySymbol.children] = new HTMLCollection<Element>(
				PropertySymbol.illegalConstructor,
				() => elements
			);
		}
		return this[PropertySymbol.children];
	}

	/**
	 * Returns character set.
	 *
	 * @deprecated
	 * @returns Character set.
	 */
	public get charset(): string {
		return this.characterSet;
	}

	/**
	 * Returns character set.
	 *
	 * @returns Character set.
	 */
	public get characterSet(): string {
		const charset = QuerySelector.querySelector(this, 'meta[charset]')?.getAttributeNS(
			null,
			'charset'
		);
		return charset ? charset : 'UTF-8';
	}

	/**
	 * Returns title.
	 *
	 * @returns Title.
	 */
	public get title(): string {
		const element = <HTMLTitleElement | null>ParentNodeUtility.getElementByTagName(this, 'title');
		if (element) {
			return element.text.trim();
		}
		return '';
	}

	/**
	 * Returns set title.
	 *
	 */
	public set title(title: string) {
		const element = ParentNodeUtility.getElementByTagName(this, 'title');
		if (element) {
			element.textContent = title;
		} else {
			const newElement = this.createElement('title');
			newElement.textContent = title;
			this.head.appendChild(newElement);
		}
	}

	/**
	 * Returns a collection of all area elements and a elements in a document with a value for the href attribute.
	 */
	public get links(): NodeList<HTMLAnchorElement | HTMLElement> {
		return <NodeList<HTMLElement>>QuerySelector.querySelectorAll(this, 'a[href],area[href]');
	}

	/**
	 * Returns a collection of all form elements in a document.
	 */
	public get forms(): HTMLCollection<HTMLFormElement> {
		if (!this[PropertySymbol.forms]) {
			this[PropertySymbol.forms] = <HTMLCollection<HTMLFormElement>>(
				ParentNodeUtility.getElementsByTagName(this, 'form')
			);
		}
		return this[PropertySymbol.forms];
	}

	/**
	 * Last element child.
	 *
	 * @returns Element.
	 */
	public get childElementCount(): number {
		return this[PropertySymbol.elementArray].length;
	}

	/**
	 * First element child.
	 *
	 * @returns Element.
	 */
	public get firstElementChild(): Element {
		return this[PropertySymbol.elementArray][0] ?? null;
	}

	/**
	 * Last element child.
	 *
	 * @returns Element.
	 */
	public get lastElementChild(): Element {
		const children = this[PropertySymbol.elementArray];
		return children[children.length - 1] ?? null;
	}

	/**
	 * Returns cookie string.
	 *
	 * @returns Cookie.
	 */
	public get cookie(): string {
		const browserFrame = new WindowBrowserContext(this[PropertySymbol.window]).getBrowserFrame();
		if (!browserFrame) {
			return '';
		}
		return CookieStringUtility.cookiesToString(
			browserFrame.page.context.cookieContainer.getCookies(
				new URL(this[PropertySymbol.window].location.href),
				true
			)
		);
	}

	/**
	 * Sets a cookie string.
	 *
	 * @param cookie Cookie string.
	 */
	public set cookie(cookie: string) {
		const browserFrame = new WindowBrowserContext(this[PropertySymbol.window]).getBrowserFrame();
		if (!browserFrame) {
			return;
		}
		browserFrame.page.context.cookieContainer.addCookies([
			CookieStringUtility.stringToCookie(new URL(this[PropertySymbol.window].location.href), cookie)
		]);
	}

	/**
	 * Node name.
	 *
	 * @returns Node name.
	 */
	public get nodeName(): string {
		return '#document';
	}

	/**
	 * Returns <html> element.
	 *
	 * @returns Element.
	 */
	public get documentElement(): HTMLHtmlElement {
		return <HTMLHtmlElement>this[PropertySymbol.elementArray][0] ?? null;
	}

	/**
	 * Returns document type element.
	 *
	 * @returns Document type.
	 */
	public get doctype(): DocumentType {
		for (const node of this[PropertySymbol.nodeArray]) {
			if (node instanceof DocumentType) {
				return node;
			}
		}
		return null;
	}

	/**
	 * Returns <body> element.
	 *
	 * @returns Element.
	 */
	public get body(): HTMLBodyElement {
		const documentElement = this.documentElement;
		return documentElement
			? <HTMLBodyElement>ParentNodeUtility.getElementByTagName(documentElement, 'body')
			: null;
	}

	/**
	 * Returns <head> element.
	 *
	 * @returns Element.
	 */
	public get head(): HTMLHeadElement {
		const documentElement = this.documentElement;
		return documentElement
			? <HTMLHeadElement>ParentNodeUtility.getElementByTagName(documentElement, 'head')
			: null;
	}

	/**
	 * Returns CSS style sheets.
	 *
	 * @returns CSS style sheets.
	 */
	public get styleSheets(): CSSStyleSheet[] {
		const styles = <NodeList<HTMLLinkElement | HTMLStyleElement>>(
			QuerySelector.querySelectorAll(this, 'link[rel="stylesheet"][href],style')
		);
		const styleSheets = [];
		for (const style of styles) {
			const sheet = style.sheet;
			if (sheet) {
				styleSheets.push(sheet);
			}
		}
		return styleSheets;
	}

	/**
	 * Returns active element.
	 *
	 * @returns Active element.
	 */
	public get activeElement(): HTMLElement | SVGElement {
		if (
			this[PropertySymbol.activeElement] &&
			!this[PropertySymbol.activeElement][PropertySymbol.isConnected]
		) {
			this[PropertySymbol.activeElement] = null;
		}

		if (
			this[PropertySymbol.activeElement] &&
			this[PropertySymbol.activeElement] instanceof Element
		) {
			let rootNode: ShadowRoot | Document = <ShadowRoot | Document>(
				this[PropertySymbol.activeElement].getRootNode()
			);
			let activeElement = this[PropertySymbol.activeElement];
			while (rootNode !== this) {
				activeElement = <HTMLElement | SVGElement>(<ShadowRoot>rootNode).host;
				rootNode = activeElement ? <ShadowRoot | Document>activeElement.getRootNode() : this;
			}
			return activeElement;
		}
		return this[PropertySymbol.activeElement] || this.body || this.documentElement || null;
	}

	/**
	 * Returns scrolling element.
	 *
	 * @returns Scrolling element.
	 */
	public get scrollingElement(): HTMLElement {
		return this.documentElement;
	}

	/**
	 * Returns location.
	 *
	 * @returns Location.
	 */
	public get location(): Location {
		return this[PropertySymbol.window].location;
	}

	/**
	 * Returns scripts.
	 *
	 * @returns Scripts.
	 */
	public get scripts(): HTMLCollection<HTMLScriptElement> {
		return this.getElementsByTagName('script');
	}

	/**
	 * Returns base URI.
	 *
	 * @override
	 * @returns Base URI.
	 */
	public get baseURI(): string {
		const element = <HTMLBaseElement>ParentNodeUtility.getElementByTagName(this, 'base');
		if (element) {
			return element.href;
		}
		return this[PropertySymbol.window].location.href;
	}

	/**
	 * Returns URL.
	 *
	 * @returns URL of the current document.
	 * */
	public get URL(): string {
		return this[PropertySymbol.window].location.href;
	}

	/**
	 * Returns document URI.
	 *
	 * @returns URL of the current document.
	 * */
	public get documentURI(): string {
		return this.URL;
	}

	/**
	 * Returns domain.
	 *
	 * @returns Domain.
	 * */
	public get domain(): string {
		return this[PropertySymbol.window].location.hostname;
	}

	/**
	 * Returns document visibility state.
	 *
	 * @returns the visibility state of the current document.
	 * */
	public get visibilityState(): VisibilityStateEnum {
		if (this.defaultView) {
			return VisibilityStateEnum.visible;
		}

		return VisibilityStateEnum.hidden;
	}

	/**
	 * Returns document hidden state.
	 *
	 * @returns the hidden state of the current document.
	 * */
	public get hidden(): boolean {
		if (this.defaultView) {
			return false;
		}

		return true;
	}

	/**
	 * Gets the currently executing script element.
	 *
	 * @returns the currently executing script element.
	 */
	public get currentScript(): HTMLScriptElement {
		return this[PropertySymbol.currentScript];
	}

	/**
	 * Returns content type.
	 *
	 * @returns Content type.
	 */
	public get contentType(): string {
		return this[PropertySymbol.contentType];
	}

	/**
	 * Inserts a set of Node objects or DOMString objects after the last child of the ParentNode. DOMString objects are inserted as equivalent Text nodes.
	 *
	 * @param nodes List of Node or DOMString.
	 */
	public append(...nodes: (Node | string)[]): void {
		ParentNodeUtility.append(this, ...nodes);
	}

	/**
	 * Inserts a set of Node objects or DOMString objects before the first child of the ParentNode. DOMString objects are inserted as equivalent Text nodes.
	 *
	 * @param nodes List of Node or DOMString.
	 */
	public prepend(...nodes: (Node | string)[]): void {
		ParentNodeUtility.prepend(this, ...nodes);
	}

	/**
	 * Replaces the existing children of a node with a specified new set of children.
	 *
	 * @param nodes List of Node or DOMString.
	 */
	public replaceChildren(...nodes: (Node | string)[]): void {
		ParentNodeUtility.replaceChildren(this, ...nodes);
	}

	/**
	 * Query CSS selector to find matching nodes.
	 *
	 * @param selector CSS selector.
	 * @returns Matching elements.
	 */
	public querySelectorAll<K extends keyof IHTMLElementTagNameMap>(
		selector: K
	): NodeList<IHTMLElementTagNameMap[K]>;

	/**
	 * Query CSS selector to find matching elments.
	 *
	 * @param selector CSS selector.
	 * @returns Matching elements.
	 */
	public querySelectorAll<K extends keyof ISVGElementTagNameMap>(
		selector: K
	): NodeList<ISVGElementTagNameMap[K]>;

	/**
	 * Query CSS selector to find matching elments.
	 *
	 * @param selector CSS selector.
	 * @returns Matching elements.
	 */
	public querySelectorAll(selector: string): NodeList<Element>;

	/**
	 * Query CSS selector to find matching elments.
	 *
	 * @param selector CSS selector.
	 * @returns Matching elements.
	 */
	public querySelectorAll(selector: string): NodeList<Element> {
		return QuerySelector.querySelectorAll(this, selector);
	}

	/**
	 * Query CSS Selector to find matching node.
	 *
	 * @param selector CSS selector.
	 * @returns Matching element.
	 */
	public querySelector<K extends keyof IHTMLElementTagNameMap>(
		selector: K
	): IHTMLElementTagNameMap[K] | null;

	/**
	 * Query CSS Selector to find matching node.
	 *
	 * @param selector CSS selector.
	 * @returns Matching element.
	 */
	public querySelector<K extends keyof ISVGElementTagNameMap>(
		selector: K
	): ISVGElementTagNameMap[K] | null;

	/**
	 * Query CSS Selector to find matching node.
	 *
	 * @param selector CSS selector.
	 * @returns Matching element.
	 */
	public querySelector(selector: string): Element | null;

	/**
	 * Query CSS Selector to find matching node.
	 *
	 * @param selector CSS selector.
	 * @returns Matching element.
	 */
	public querySelector(selector: string): Element | null {
		return QuerySelector.querySelector(this, selector);
	}
	/**
	 * Returns true if the command is supported.
	 * @deprecated
	 * @param _ Command.
	 * @returns True if the command is supported, false otherwise.
	 */
	public queryCommandSupported(_: string): boolean {
		if (!arguments.length) {
			throw new this[PropertySymbol.window].TypeError(
				"Failed to execute 'queryCommandSupported' on 'Document': 1 argument required, but only 0 present."
			);
		}
		return true;
	}

	/**
	 * Returns an elements by class name.
	 *
	 * @param className Tag name.
	 * @returns Matching element.
	 */
	public getElementsByClassName(className: string): HTMLCollection<Element> {
		return ParentNodeUtility.getElementsByClassName(this, className);
	}

	/**
	 * Returns an elements by tag name.
	 *
	 * @param tagName Tag name.
	 * @returns Matching element.
	 */
	public getElementsByTagName<K extends keyof IHTMLElementTagNameMap>(
		tagName: K
	): HTMLCollection<IHTMLElementTagNameMap[K]>;

	/**
	 * Returns an elements by tag name.
	 *
	 * @param tagName Tag name.
	 * @returns Matching element.
	 */
	public getElementsByTagName<K extends keyof ISVGElementTagNameMap>(
		tagName: K
	): HTMLCollection<ISVGElementTagNameMap[K]>;

	/**
	 * Returns an elements by tag name.
	 *
	 * @param tagName Tag name.
	 * @returns Matching element.
	 */
	public getElementsByTagName(tagName: string): HTMLCollection<Element>;

	/**
	 * Returns an elements by tag name.
	 *
	 * @param tagName Tag name.
	 * @returns Matching element.
	 */
	public getElementsByTagName(tagName: string): HTMLCollection<Element> {
		return ParentNodeUtility.getElementsByTagName(this, tagName);
	}

	/**
	 * Returns an elements by tag name and namespace.
	 *
	 * @param namespaceURI Namespace URI.
	 * @param tagName Tag name.
	 * @returns Matching element.
	 */
	public getElementsByTagNameNS<K extends keyof IHTMLElementTagNameMap>(
		namespaceURI: 'http://www.w3.org/1999/xhtml',
		tagName: K
	): HTMLCollection<IHTMLElementTagNameMap[K]>;

	/**
	 * Returns an elements by tag name and namespace.
	 *
	 * @param namespaceURI Namespace URI.
	 * @param tagName Tag name.
	 * @returns Matching element.
	 */
	public getElementsByTagNameNS<K extends keyof ISVGElementTagNameMap>(
		namespaceURI: 'http://www.w3.org/2000/svg',
		tagName: K
	): HTMLCollection<ISVGElementTagNameMap[K]>;

	/**
	 * Returns an elements by tag name and namespace.
	 *
	 * @param namespaceURI Namespace URI.
	 * @param tagName Tag name.
	 * @returns Matching element.
	 */
	public getElementsByTagNameNS(namespaceURI: string, tagName: string): HTMLCollection<Element>;

	/**
	 * Returns an elements by tag name and namespace.
	 *
	 * @param namespaceURI Namespace URI.
	 * @param tagName Tag name.
	 * @returns Matching element.
	 */
	public getElementsByTagNameNS(namespaceURI: string, tagName: string): HTMLCollection<Element> {
		return ParentNodeUtility.getElementsByTagNameNS(this, namespaceURI, tagName);
	}

	/**
	 * Returns an element by ID.
	 *
	 * @param id ID.
	 * @returns Matching element.
	 */
	public getElementById(id: string): Element | null {
		return <Element>ParentNodeUtility.getElementById(this, id);
	}

	/**
	 * Returns an element by Name.
	 *
	 * @returns Matching element.
	 * @param name
	 */
	public getElementsByName(name: string): NodeList<Element> {
		return QuerySelector.querySelectorAll(this, `[name="${name}"]`);
	}

	/**
	 * Replaces the document HTML with new HTML.
	 *
	 * @param html HTML.
	 */
	public write(html: string): void {
		if (this[PropertySymbol.isFirstWrite] || this[PropertySymbol.isFirstWriteAfterOpen]) {
			if (this[PropertySymbol.isFirstWrite]) {
				if (!this[PropertySymbol.isFirstWriteAfterOpen]) {
					this.open();
				}

				this[PropertySymbol.isFirstWrite] = false;
			}

			const { documentElement, head, body } = this;

			if (!documentElement || !head || !body) {
				this.open();
			}

			this[PropertySymbol.isFirstWrite] = false;
			this[PropertySymbol.isFirstWriteAfterOpen] = false;

			new HTMLParser(this[PropertySymbol.window], {
				evaluateScripts: true
			}).parse(html, this);
		} else {
			new HTMLParser(this[PropertySymbol.window], {
				evaluateScripts: true
			}).parse(html, this.body);
		}
	}

	/**
	 * Opens the document.
	 *
	 * @returns Document.
	 */
	public open(): Document {
		this[PropertySymbol.isFirstWriteAfterOpen] = true;

		for (const eventType of this[PropertySymbol.listeners].bubbling.keys()) {
			const listeners = this[PropertySymbol.listeners].bubbling.get(eventType);
			if (listeners) {
				for (const listener of listeners) {
					this.removeEventListener(eventType, listener);
				}
			}
		}

		for (const eventType of this[PropertySymbol.listeners].capturing.keys()) {
			const listeners = this[PropertySymbol.listeners].capturing.get(eventType);
			if (listeners) {
				for (const listener of listeners) {
					this.removeEventListener(eventType, listener);
				}
			}
		}

		const childNodes = this[PropertySymbol.nodeArray];
		while (childNodes.length) {
			this.removeChild(childNodes[0]);
		}

		// Default document elements
		const doctype = this[PropertySymbol.implementation].createDocumentType('html', '', '');
		const documentElement = this.createElement('html');
		const bodyElement = this.createElement('body');
		const headElement = this.createElement('head');

		this.appendChild(doctype);
		this.appendChild(documentElement);

		documentElement.appendChild(headElement);
		documentElement.appendChild(bodyElement);

		return this;
	}

	/**
	 * Closes the document.
	 */
	public close(): void {}

	/* eslint-disable jsdoc/valid-types */

	/**
	 * Creates an element.
	 *
	 * @param qualifiedName Tag name.
	 * @param [options] Options.
	 * @param [options.is] Tag name of a custom element previously defined via customElements.define().
	 * @returns Element.
	 */
	public createElement<K extends keyof IHTMLElementTagNameMap>(
		qualifiedName: K,
		options?: { is?: string }
	): IHTMLElementTagNameMap[K];

	/**
	 * Creates an element.
	 *
	 * @param qualifiedName Tag name.
	 * @param [options] Options.
	 * @param [options.is] Tag name of a custom element previously defined via customElements.define().
	 * @returns Element.
	 */
	public createElement<K extends keyof ISVGElementTagNameMap>(
		qualifiedName: K,
		options?: { is?: string }
	): ISVGElementTagNameMap[K];

	/**
	 * Creates an element.
	 *
	 * @param qualifiedName Tag name.
	 * @param [options] Options.
	 * @param [options.is] Tag name of a custom element previously defined via customElements.define().
	 * @returns Element.
	 */
	public createElement(qualifiedName: string, options?: { is?: string }): HTMLElement;

	/**
	 * Creates an element.
	 *
	 * @param qualifiedName Tag name.
	 * @param [options] Options.
	 * @param [options.is] Tag name of a custom element previously defined via customElements.define().
	 * @returns Element.
	 */
	public createElement(qualifiedName: string, options?: { is?: string }): HTMLElement {
		return <HTMLElement>(
			this.createElementNS(
				NamespaceURI.html,
				StringUtility.asciiLowerCase(String(qualifiedName)),
				options
			)
		);
	}

	/**
	 * Creates an element with the specified namespace URI and qualified name.
	 *
	 * @param namespaceURI Namespace URI.
	 * @param qualifiedName Tag name.
	 * @param [options] Options.
	 * @param [options.is] Tag name of a custom element previously defined via customElements.define().
	 * @returns Element.
	 */
	public createElementNS<K extends keyof IHTMLElementTagNameMap>(
		namespaceURI: 'http://www.w3.org/1999/xhtml',
		qualifiedName: K,
		options?: { is?: string }
	): IHTMLElementTagNameMap[K];

	/**
	 * Creates an element with the specified namespace URI and qualified name.
	 *
	 * @param namespaceURI Namespace URI.
	 * @param qualifiedName Tag name.
	 * @param [options] Options.
	 * @param [options.is] Tag name of a custom element previously defined via customElements.define().
	 * @returns Element.
	 */
	public createElementNS<K extends keyof ISVGElementTagNameMap>(
		namespaceURI: 'http://www.w3.org/2000/svg',
		qualifiedName: K,
		options?: { is?: string }
	): ISVGElementTagNameMap[K];

	/**
	 * Creates an element with the specified namespace URI and qualified name.
	 *
	 * @param namespaceURI Namespace URI.
	 * @param qualifiedName Tag name.
	 * @param [options] Options.
	 * @param [options.is] Tag name of a custom element previously defined via customElements.define().
	 * @returns Element.
	 */
	public createElementNS(
		namespaceURI: string,
		qualifiedName: string,
		options?: { is?: string }
	): Element;

	/**
	 * Creates an element with the specified namespace URI and qualified name.
	 *
	 * @param namespaceURI Namespace URI.
	 * @param qualifiedName Tag name.
	 * @param [options] Options.
	 * @param [options.is] Tag name of a custom element previously defined via customElements.define().
	 * @returns Element.
	 */
	public createElementNS(
		namespaceURI: string,
		qualifiedName: string,
		options?: { is?: string }
	): Element {
		const window = this[PropertySymbol.window];

		qualifiedName = String(qualifiedName);

		if (!qualifiedName) {
			throw new window.DOMException(
				"Failed to execute 'createElementNS' on 'Document': The qualified name provided is empty."
			);
		}

		const parts = qualifiedName.split(':');
		const localName = parts[1] ?? parts[0];
		const prefix = parts[1] ? parts[0] : null;

		switch (namespaceURI) {
			case NamespaceURI.svg:
				const config = SVGElementConfig[qualifiedName.toLowerCase()];
				const svgElementClass =
					config && config.localName === qualifiedName
						? window[config.className]
						: window.SVGElement;

				const svgElement = NodeFactory.createNode<SVGElement>(this, svgElementClass);

				svgElement[PropertySymbol.tagName] = qualifiedName;
				svgElement[PropertySymbol.localName] = localName;
				svgElement[PropertySymbol.prefix] = prefix;
				svgElement[PropertySymbol.namespaceURI] = namespaceURI;
				svgElement[PropertySymbol.isValue] = options && options.is ? String(options.is) : null;

				return svgElement;
			case NamespaceURI.html:
				// Custom HTML element
				// If a polyfill is used, [PropertySymbol.registry] may be undefined
				const customElementDefinition = window.customElements[PropertySymbol.registry]?.get(
					options && options.is ? String(options.is) : qualifiedName
				);

				if (customElementDefinition) {
					const element = new customElementDefinition.elementClass();
					element[PropertySymbol.tagName] = StringUtility.asciiUpperCase(qualifiedName);
					element[PropertySymbol.localName] = localName;
					element[PropertySymbol.prefix] = prefix;
					element[PropertySymbol.namespaceURI] = namespaceURI;
					element[PropertySymbol.isValue] = options && options.is ? String(options.is) : null;
					return element;
				}

				const elementClass = HTMLElementConfig[qualifiedName]
					? window[HTMLElementConfig[qualifiedName].className]
					: null;

				// Known HTML element
				if (elementClass) {
					const element = NodeFactory.createNode<Element>(this, elementClass);

					element[PropertySymbol.tagName] = StringUtility.asciiUpperCase(qualifiedName);
					element[PropertySymbol.localName] = localName;
					element[PropertySymbol.prefix] = prefix;
					element[PropertySymbol.namespaceURI] = namespaceURI;
					element[PropertySymbol.isValue] = options && options.is ? String(options.is) : null;

					return element;
				}

				// Unknown HTML element (if it has an hyphen in the name, it is a custom element that hasn't been defined yet)
				const unknownElementClass = qualifiedName.includes('-')
					? window.HTMLElement
					: window.HTMLUnknownElement;

				const unknownElement = NodeFactory.createNode<Element>(this, unknownElementClass);

				unknownElement[PropertySymbol.tagName] = StringUtility.asciiUpperCase(qualifiedName);
				unknownElement[PropertySymbol.localName] = localName;
				unknownElement[PropertySymbol.prefix] = prefix;
				unknownElement[PropertySymbol.namespaceURI] = namespaceURI;
				unknownElement[PropertySymbol.isValue] = options && options.is ? String(options.is) : null;

				return unknownElement;
			default:
				const element = NodeFactory.createNode<Element>(this, Element);

				element[PropertySymbol.tagName] = qualifiedName;
				element[PropertySymbol.localName] = localName;
				element[PropertySymbol.prefix] = prefix;
				element[PropertySymbol.namespaceURI] = namespaceURI;
				element[PropertySymbol.isValue] = options && options.is ? String(options.is) : null;

				return element;
		}
	}

	/* eslint-enable jsdoc/valid-types */

	/**
	 * Creates a text node.
	 *
	 * @param [data] Text data.
	 * @returns Text node.
	 */
	public createTextNode(data: string): Text {
		if (arguments.length < 1) {
			throw new this[PropertySymbol.window].TypeError(
				`Failed to execute 'createTextNode' on 'Document': 1 argument required, but only ${arguments.length} present.`
			);
		}
		// We should use the NodeFactory and not the class constructor, so that owner document will be this document
		return NodeFactory.createNode(this, this[PropertySymbol.window].Text, String(data));
	}

	/**
	 * Creates a comment node.
	 *
	 * @param [data] Text data.
	 * @returns Text node.
	 */
	public createComment(data?: string): Comment {
		if (arguments.length < 1) {
			throw new this[PropertySymbol.window].TypeError(
				`Failed to execute 'createComment' on 'Document': 1 argument required, but only ${arguments.length} present.`
			);
		}
		// We should use the NodeFactory and not the class constructor, so that owner document will be this document
		return NodeFactory.createNode(this, this[PropertySymbol.window].Comment, String(data));
	}

	/**
	 * Creates a document fragment.
	 *
	 * @returns Document fragment.
	 */
	public createDocumentFragment(): DocumentFragment {
		// We should use the NodeFactory and not the class constructor, so that owner document will be this document
		return NodeFactory.createNode(this, this[PropertySymbol.window].DocumentFragment);
	}

	/**
	 * Creates a node iterator.
	 *
	 * @param root Root.
	 * @param [whatToShow] What to show.
	 * @param [filter] Filter.
	 */
	public createNodeIterator(root: Node, whatToShow = -1, filter: INodeFilter = null): NodeIterator {
		return new NodeIterator(root, whatToShow, filter);
	}

	/**
	 * Creates a Tree Walker.
	 *
	 * @param root Root.
	 * @param [whatToShow] What to show.
	 * @param [filter] Filter.
	 */
	public createTreeWalker(root: Node, whatToShow = -1, filter: INodeFilter = null): TreeWalker {
		return new TreeWalker(root, whatToShow, filter);
	}

	/**
	 * Creates an event.
	 *
	 * @deprecated
	 * @param type Type.
	 * @returns Event.
	 */
	public createEvent(type: string): Event {
		if (typeof this[PropertySymbol.window][type] === 'function') {
			return new this[PropertySymbol.window][type]('init');
		}
		return new Event('init');
	}

	/**
	 * Creates an Attr node.
	 *
	 * @param qualifiedName Name.
	 * @returns Attribute.
	 */
	public createAttribute(qualifiedName: string): Attr {
		return this.createAttributeNS(null, StringUtility.asciiLowerCase(qualifiedName));
	}

	/**
	 * Creates a namespaced Attr node.
	 *
	 * @param namespaceURI Namespace URI.
	 * @param qualifiedName Qualified name.
	 * @returns Element.
	 */
	public createAttributeNS(namespaceURI: string, qualifiedName: string): Attr {
		// We should use the NodeFactory and not the class constructor, so that owner document will be this document
		const attribute = NodeFactory.createNode(this, this[PropertySymbol.window].Attr);

		const parts = qualifiedName.split(':');
		attribute[PropertySymbol.namespaceURI] = namespaceURI;
		attribute[PropertySymbol.name] = qualifiedName;
		attribute[PropertySymbol.localName] = parts[1] ?? qualifiedName;
		attribute[PropertySymbol.prefix] = parts[1] ? parts[0] : null;

		return attribute;
	}

	/**
	 * Imports a node.
	 *
	 * @param node Node to import.
	 * @param [deep=false] Set to "true" if the clone should be deep.
	 */
	public importNode(node: Node, deep = false): Node {
		if (!(node instanceof Node)) {
			throw new this[PropertySymbol.window].DOMException('Parameter 1 was not of type Node.');
		}
		const clone = node.cloneNode(deep);
		this.#importNode(clone);
		return clone;
	}

	/**
	 * Creates a range.
	 *
	 * @returns Range.
	 */
	public createRange(): Range {
		return new this[PropertySymbol.window].Range();
	}

	/**
	 * Adopts a node.
	 *
	 * @param node Node to adopt.
	 * @returns Adopted node.
	 */
	public adoptNode(node: Node): Node {
		if (!(node instanceof Node)) {
			throw new this[PropertySymbol.window].DOMException('Parameter 1 was not of type Node.');
		}

		const adopted = node[PropertySymbol.parentNode]
			? node[PropertySymbol.parentNode].removeChild(node)
			: node;
		const document = this;
		Object.defineProperty(adopted, 'ownerDocument', { value: document });
		return adopted;
	}

	/**
	 * Returns selection.
	 *
	 * @returns Selection.
	 */
	public getSelection(): Selection {
		if (!this.#selection) {
			this.#selection = new Selection(this);
		}
		return this.#selection;
	}

	/**
	 * Returns a boolean value indicating whether the document or any element inside the document has focus.
	 *
	 * @returns "true" if the document has focus.
	 */
	public hasFocus(): boolean {
		return !!this.activeElement;
	}

	/**
	 * Creates a Processing Instruction node.
	 *
	 * @param target Target.
	 * @param data Data.
	 * @returns ProcessingInstruction.
	 */
	public createProcessingInstruction(target: string, data: string): ProcessingInstruction {
		if (arguments.length < 2) {
			throw new this[PropertySymbol.window].TypeError(
				`Failed to execute 'createProcessingInstruction' on 'Document': 2 arguments required, but only ${arguments.length} present.`
			);
		}

		target = String(target);
		data = String(data);

		if (!target || !PROCESSING_INSTRUCTION_TARGET_REGEXP.test(target)) {
			throw new this[PropertySymbol.window].DOMException(
				`Failed to execute 'createProcessingInstruction' on 'Document': The target provided ('${target}') is not a valid name.`
			);
		}
		if (data.includes('?>')) {
			throw new this[PropertySymbol.window].DOMException(
				`Failed to execute 'createProcessingInstruction' on 'Document': The data provided ('?>') contains '?>'`
			);
		}
		// We should use the NodeFactory and not the class constructor, so that owner document will be this document
		const element = NodeFactory.createNode(this, this[PropertySymbol.window].ProcessingInstruction);

		element[PropertySymbol.data] = data;
		element[PropertySymbol.target] = target;

		return element;
	}

	/**
	 * Get element at a given point.
	 *
	 * @param _x horizontal coordinate
	 * @param _y vertical coordinate
	 * @returns Always returns null since Happy DOM does not render elements.
	 */
	public elementFromPoint(_x: number, _y: number): Element | null {
		return null;
	}

	/**
	 * Imports a node.
	 *
	 * @param node Node.
	 */
	#importNode(node: Node): void {
		node[PropertySymbol.ownerDocument] = this;

		for (const child of node[PropertySymbol.nodeArray]) {
			this.#importNode(child);
		}
	}
}
