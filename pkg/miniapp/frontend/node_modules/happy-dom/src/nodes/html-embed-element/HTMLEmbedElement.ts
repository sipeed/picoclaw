import HTMLElement from '../html-element/HTMLElement.js';
import * as PropertySymbol from '../../PropertySymbol.js';

/**
 * HTMLEmbedElement
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/HTMLEmbedElement
 */
export default class HTMLEmbedElement extends HTMLElement {
	/**
	 * Returns height.
	 *
	 * @returns Height.
	 */
	public get height(): string {
		return this.getAttribute('height') || '';
	}

	/**
	 * Sets height.
	 *
	 * @param height Height.
	 */
	public set height(height: string) {
		this.setAttribute('height', height);
	}

	/**
	 * Returns width.
	 *
	 * @returns Width.
	 */
	public get width(): string {
		return this.getAttribute('width') || '';
	}

	/**
	 * Sets width.
	 *
	 * @param width Width.
	 */
	public set width(width: string) {
		this.setAttribute('width', width);
	}

	/**
	 * Returns source.
	 *
	 * @returns Source.
	 */
	public get src(): string {
		if (!this.hasAttribute('src')) {
			return '';
		}

		try {
			return new URL(this.getAttribute('src'), this[PropertySymbol.ownerDocument].location.href)
				.href;
		} catch (e) {
			return this.getAttribute('src');
		}
	}

	/**
	 * Sets source.
	 *
	 * @param src Source.
	 */
	public set src(src: string) {
		this.setAttribute('src', src);
	}

	/**
	 * Returns type.
	 *
	 * @returns Type.
	 */
	public get type(): string {
		return this.getAttribute('type') || '';
	}

	/**
	 * Sets type.
	 *
	 * @param type Type.
	 */
	public set type(type: string) {
		this.setAttribute('type', type);
	}
}
