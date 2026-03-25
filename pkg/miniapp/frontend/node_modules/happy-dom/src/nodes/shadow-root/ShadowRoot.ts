import DocumentFragment from '../document-fragment/DocumentFragment.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import Element from '../element/Element.js';
import CSSStyleSheet from '../../css/CSSStyleSheet.js';
import HTMLElement from '../../nodes/html-element/HTMLElement.js';
import Event from '../../event/Event.js';
import SVGElement from '../svg-element/SVGElement.js';
import Document from '../document/Document.js';
import HTMLSerializer from '../../html-serializer/HTMLSerializer.js';
import HTMLParser from '../../html-parser/HTMLParser.js';

/**
 * ShadowRoot.
 */
export default class ShadowRoot extends DocumentFragment {
	// Public properties
	public declare cloneNode: (deep?: boolean) => ShadowRoot;

	// Events
	public onslotchange: ((event: Event) => void) | null = null;

	// Internal properties
	public [PropertySymbol.adoptedStyleSheets]: CSSStyleSheet[] = [];
	public [PropertySymbol.mode] = 'open';
	public [PropertySymbol.host]: Element | null = null;
	public [PropertySymbol.clonable]: boolean = false;
	public [PropertySymbol.delegatesFocus]: boolean = false;
	public [PropertySymbol.serializable]: boolean = false;
	public [PropertySymbol.slotAssignment]: 'named' | 'manual' = 'named';

	/**
	 * Returns mode.
	 *
	 * @returns Mode.
	 */
	public get mode(): string {
		return this[PropertySymbol.mode];
	}

	/**
	 * Returns host.
	 *
	 * @returns Host.
	 */
	public get host(): Element {
		return this[PropertySymbol.host];
	}

	/**
	 * Returns clonable.
	 *
	 * @returns Clonable.
	 */
	public get clonable(): boolean {
		return this[PropertySymbol.clonable];
	}

	/**
	 * Returns delegates focus.
	 *
	 * @returns Delegates focus.
	 */
	public get delegatesFocus(): boolean {
		// TODO: Implement delegates focus
		return this[PropertySymbol.delegatesFocus];
	}

	/**
	 * Returns serializable.
	 *
	 * @returns Serializable.
	 */
	public get serializable(): boolean {
		return this[PropertySymbol.serializable];
	}

	/**
	 * Returns slot assignment.
	 *
	 * @returns Slot assignment.
	 */
	public get slotAssignment(): 'named' | 'manual' {
		return this[PropertySymbol.slotAssignment];
	}

	/**
	 * The element that's currently in full screen mode for this shadow tree.
	 *
	 * @returns Fullscreen element.
	 */
	public get fullscreenElement(): Element | null {
		// TODO: Implement fullscreen element
		return null;
	}

	/**
	 * Returns the Element within the shadow tree that is currently being presented in picture-in-picture mode.
	 *
	 * @returns Picture-in-picture element.
	 */
	public get pictureInPictureElement(): Element | null {
		// TODO: Implement picture-in-picture element
		return null;
	}

	/**
	 * Returns the Element set as the target for mouse events while the pointer is locked. null if lock is pending, pointer is unlocked, or if the target is in another tree.
	 *
	 * @returns Pointer lock element.
	 */
	public get pointerLockElement(): Element | null {
		// TODO: Implement pointer lock element
		return null;
	}

	/**
	 * Returns inner HTML.
	 *
	 * @returns HTML.
	 */
	public get innerHTML(): string {
		const serializer = new HTMLSerializer();
		let html = '';
		for (const node of this[PropertySymbol.nodeArray]) {
			html += serializer.serializeToString(node);
		}
		return html;
	}

	/**
	 * Sets inner HTML.
	 *
	 * @param html HTML.
	 */
	public set innerHTML(html: string) {
		const childNodes = this[PropertySymbol.nodeArray];

		while (childNodes.length) {
			this.removeChild(childNodes[0]);
		}

		new HTMLParser(this[PropertySymbol.window]).parse(html, this);
	}

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
	 * Returns active element.
	 *
	 * @returns Active element.
	 */
	public get activeElement(): HTMLElement | SVGElement | null {
		let activeElement: HTMLElement | SVGElement =
			this[PropertySymbol.ownerDocument][PropertySymbol.activeElement];

		let rootNode: ShadowRoot | Document = <ShadowRoot | Document>activeElement?.getRootNode();

		if (!rootNode || rootNode === this[PropertySymbol.ownerDocument]) {
			return null;
		}

		if (rootNode === this) {
			return activeElement;
		}

		while (rootNode && rootNode !== this) {
			activeElement = <HTMLElement | SVGElement>(<ShadowRoot>rootNode).host;
			rootNode = <ShadowRoot | Document>activeElement.getRootNode();
		}

		return activeElement;
	}

	/**
	 * Returns an array of all Animation objects currently in effect, whose target elements are descendants of the shadow tree.
	 *
	 * @returns Array of animations.
	 */
	public getAnimations(): object[] {
		// TODO: Implement getAnimations()
		return [];
	}

	/**
	 * Parses a string of HTML into a document fragment, without sanitization, which then replaces the shadowroot's original subtree. The HTML string may include declarative shadow roots, which would be parsed as template elements the HTML was set using ShadowRoot.innerHTML.
	 *
	 * @param html HTML.
	 */
	public setHTMLUnsafe(html: string): void {
		// TODO: Implement support for declarative shadow roots

		const childNodes = this[PropertySymbol.nodeArray];

		while (childNodes.length) {
			this.removeChild(childNodes[0]);
		}

		new HTMLParser(this[PropertySymbol.window]).parse(html, this);
	}

	/**
	 * Converts to string.
	 *
	 * @returns String.
	 */
	public toString(): string {
		return this.innerHTML;
	}

	/**
	 * @override
	 */
	public override [PropertySymbol.cloneNode](deep = false): ShadowRoot {
		const clone = <ShadowRoot>super[PropertySymbol.cloneNode](deep);
		clone[PropertySymbol.mode] = this[PropertySymbol.mode];
		clone[PropertySymbol.clonable] = this[PropertySymbol.clonable];
		clone[PropertySymbol.delegatesFocus] = this[PropertySymbol.delegatesFocus];
		clone[PropertySymbol.serializable] = this[PropertySymbol.serializable];
		clone[PropertySymbol.slotAssignment] = this[PropertySymbol.slotAssignment];
		return clone;
	}
}
