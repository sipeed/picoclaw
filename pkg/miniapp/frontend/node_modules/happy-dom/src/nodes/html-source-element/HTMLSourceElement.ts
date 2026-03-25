import HTMLElement from '../html-element/HTMLElement.js';
import * as PropertySymbol from '../../PropertySymbol.js';

/**
 * HTMLSourceElement
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/HTMLSourceElement
 */
export default class HTMLSourceElement extends HTMLElement {
	/**
	 * Returns height.
	 *
	 * @returns Height.
	 */
	public get height(): number {
		const value = Number(this.getAttribute('height'));
		return isNaN(value) || value < 0 ? 0 : value;
	}

	/**
	 * Sets height.
	 *
	 * @param value Height.
	 */
	public set height(value: number) {
		const parsedValue = Number(value);
		this.setAttribute('height', isNaN(parsedValue) || parsedValue < 0 ? '0' : String(parsedValue));
	}

	/**
	 * Returns width.
	 *
	 * @returns Width.
	 */
	public get width(): number {
		const value = Number(this.getAttribute('width'));
		return isNaN(value) || value < 0 ? 0 : value;
	}

	/**
	 * Sets width.
	 *
	 * @param value Width.
	 */
	public set width(value: number) {
		const parsedValue = Number(value);
		this.setAttribute('width', isNaN(parsedValue) || parsedValue < 0 ? '0' : String(parsedValue));
	}

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
	 * @param value Media.
	 */
	public set media(value: string) {
		this.setAttribute('media', value);
	}

	/**
	 * Returns sizes.
	 *
	 * @returns Sizes.
	 */
	public get sizes(): string {
		return this.getAttribute('sizes') || '';
	}

	/**
	 * Sets sizes.
	 *
	 * @param value Sizes.
	 */
	public set sizes(value: string) {
		this.setAttribute('sizes', value);
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
	 * Returns source set.
	 *
	 * @returns Source set.
	 */
	public get srcset(): string {
		return this.getAttribute('srcset') || '';
	}

	/**
	 * Sets source set.
	 *
	 * @param value Source set.
	 */
	public set srcset(value: string) {
		this.setAttribute('srcset', value);
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
