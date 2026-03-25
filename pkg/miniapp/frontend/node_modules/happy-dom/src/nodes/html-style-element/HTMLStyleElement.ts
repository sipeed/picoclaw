import CSSStyleSheet from '../../css/CSSStyleSheet.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import HTMLElement from '../html-element/HTMLElement.js';

/**
 * HTML Style Element.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/HTMLStyleElement.
 */
export default class HTMLStyleElement extends HTMLElement {
	private [PropertySymbol.sheet]: CSSStyleSheet | null = null;
	public [PropertySymbol.styleNode] = this;
	public [PropertySymbol.disabled] = false;

	/**
	 * Returns media.
	 *
	 * @returns Media.
	 */
	public get media(): string {
		return this.getAttribute('media') || '';
	}

	/**
	 * Sets media.
	 *
	 * @param media Media.
	 */
	public set media(media: string) {
		this.setAttribute('media', media);
	}

	/**
	 * Returns type.
	 *
	 * @deprecated
	 * @returns Type.
	 */
	public get type(): string {
		return this.getAttribute('type') || '';
	}

	/**
	 * Sets type.
	 *
	 * @deprecated
	 * @param type Type.
	 */
	public set type(type: string) {
		this.setAttribute('type', type);
	}

	/**
	 * Returns disabled.
	 *
	 * @returns Disabled.
	 */
	public get disabled(): boolean {
		return this[PropertySymbol.disabled];
	}

	/**
	 * Sets disabled.
	 *
	 * @param disabled Disabled.
	 */
	public set disabled(disabled: boolean) {
		this[PropertySymbol.disabled] = Boolean(disabled);
	}

	/**
	 * Returns CSS style sheet.
	 *
	 * @returns CSS style sheet.
	 */
	public get sheet(): CSSStyleSheet {
		if (!this[PropertySymbol.isConnected]) {
			return null;
		}
		if (!this[PropertySymbol.sheet]) {
			this[PropertySymbol.sheet] = new CSSStyleSheet();
			this[PropertySymbol.sheet].replaceSync(this.textContent);
		}
		return this[PropertySymbol.sheet];
	}

	/**
	 * @override
	 */
	public override [PropertySymbol.disconnectedFromDocument](): void {
		super[PropertySymbol.disconnectedFromDocument]();
		this[PropertySymbol.sheet] = null;
	}

	/**
	 * Updates the CSSStyleSheet with the text content.
	 */
	public [PropertySymbol.updateSheet](): void {
		if (this[PropertySymbol.sheet]) {
			this[PropertySymbol.sheet].replaceSync(this.textContent);
		}
	}
}
