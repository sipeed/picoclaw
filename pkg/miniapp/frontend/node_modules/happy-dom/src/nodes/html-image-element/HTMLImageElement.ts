import HTMLElement from '../html-element/HTMLElement.js';
import * as PropertySymbol from '../../PropertySymbol.js';

/**
 * HTML Image Element.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/HTMLImageElement.
 */
export default class HTMLImageElement extends HTMLElement {
	public [PropertySymbol.tagName] = 'IMG';
	public [PropertySymbol.complete] = false;
	public [PropertySymbol.naturalHeight] = 0;
	public [PropertySymbol.naturalWidth] = 0;
	public [PropertySymbol.loading] = 'auto';
	public [PropertySymbol.x] = 0;
	public [PropertySymbol.y] = 0;
	public declare cloneNode: (deep?: boolean) => HTMLImageElement;

	/**
	 * Returns complete.
	 *
	 * @returns Complete.
	 */
	public get complete(): boolean {
		return this[PropertySymbol.complete];
	}

	/**
	 * Returns natural height.
	 *
	 * @returns Natural height.
	 */
	public get naturalHeight(): number {
		return this[PropertySymbol.naturalHeight];
	}

	/**
	 * Returns natural width.
	 *
	 * @returns Natural width.
	 */
	public get naturalWidth(): number {
		return this[PropertySymbol.naturalWidth];
	}

	/**
	 * Returns loading.
	 *
	 * @returns Loading.
	 */
	public get loading(): string {
		const loading = this.getAttribute('loading');
		return loading === 'eager' || loading === 'lazy' ? loading : 'auto';
	}

	/**
	 * Sets loading.
	 *
	 * @param loading Loading.
	 */
	public set loading(loading: string) {
		this.setAttribute('loading', loading);
	}

	/**
	 * Returns x.
	 */
	public get x(): number {
		return this[PropertySymbol.x];
	}

	/**
	 * Returns y.
	 */
	public get y(): number {
		return this[PropertySymbol.y];
	}

	/**
	 * Returns decoding.
	 *
	 * @returns Decoding.
	 */
	public get decoding(): string {
		return this.getAttribute('decoding') || 'auto';
	}

	/**
	 * Sets decoding.
	 *
	 * @param decoding Decoding.
	 */
	public set decoding(decoding: string) {
		this.setAttribute('decoding', decoding);
	}

	/**
	 * Returns cross origin.
	 *
	 * @returns Cross origin.
	 */
	public get crossOrigin(): string | null {
		return this.getAttribute('crossOrigin');
	}

	/**
	 * Sets cross origin.
	 *
	 * @param crossOrigin Cross origin.
	 */
	public set crossOrigin(crossOrigin: string | null) {
		if (crossOrigin === 'anonymous' || crossOrigin === 'use-credentials') {
			this.setAttribute('crossOrigin', crossOrigin);
		}
	}

	/**
	 * Returns alt.
	 *
	 * @returns Alt.
	 */
	public get alt(): string {
		return this.getAttribute('alt') || '';
	}

	/**
	 * Sets alt.
	 *
	 * @param alt Alt.
	 */
	public set alt(alt: string) {
		this.setAttribute('alt', alt);
	}

	/**
	 * Returns current src.
	 *
	 * @returns Current src.
	 */
	public get currentSrc(): string {
		return this.src;
	}

	/**
	 * Returns width.
	 *
	 * @returns Width.
	 */
	public get width(): number {
		const width = this.getAttribute('width');
		return width !== null ? Number(width) : 0;
	}

	/**
	 * Sets width.
	 *
	 * @param width Width.
	 */
	public set width(width: number) {
		this.setAttribute('width', String(width));
	}

	/**
	 * Returns height.
	 *
	 * @returns Height.
	 */
	public get height(): number {
		const height = this.getAttribute('height');
		return height !== null ? Number(height) : 0;
	}

	/**
	 * Sets height.
	 *
	 * @param height Height.
	 */
	public set height(height: number) {
		this.setAttribute('height', String(height));
	}

	/**
	 * Returns is map.
	 *
	 * @returns Is map.
	 */
	public get isMap(): boolean {
		return this.getAttribute('ismap') !== null;
	}

	/**
	 * Sets is map.
	 *
	 * @param ismap Is map.
	 */
	public set isMap(isMap: boolean) {
		if (!isMap) {
			this.removeAttribute('ismap');
		} else {
			this.setAttribute('ismap', '');
		}
	}

	/**
	 * Returns referrer policy.
	 *
	 * @returns Referrer policy.
	 */
	public get referrerPolicy(): string {
		return this.getAttribute('referrerpolicy') || '';
	}

	/**
	 * Sets referrer policy.
	 *
	 * @param referrerPolicy Referrer policy.
	 */
	public set referrerPolicy(referrerPolicy: string) {
		this.setAttribute('referrerpolicy', referrerPolicy);
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
	 * @param sizes Sizes.
	 */
	public set sizes(sizes: string) {
		this.setAttribute('sizes', sizes);
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
	 * Returns srcset.
	 *
	 * @returns Source.
	 */
	public get srcset(): string {
		return this.getAttribute('srcset') || '';
	}

	/**
	 * Sets src set.
	 *
	 * @param srcset Src set.
	 */
	public set srcset(srcset: string) {
		this.setAttribute('srcset', srcset);
	}

	/**
	 * Returns use map.
	 *
	 * @returns Use map.
	 */
	public get useMap(): string {
		return this.getAttribute('usemap') || '';
	}

	/**
	 * Sets is map.
	 *
	 * @param useMap Is map.
	 */
	public set useMap(useMap: string) {
		this.setAttribute('usemap', useMap);
	}

	/**
	 * The decode() method of the HTMLImageElement interface returns a Promise that resolves when the image is decoded and it is safe to append the image to the DOM.
	 *
	 * @returns Promise.
	 */
	public decode(): Promise<void> {
		return Promise.resolve();
	}

	/**
	 * @override
	 */
	public override [PropertySymbol.cloneNode](deep = false): HTMLImageElement {
		return <HTMLImageElement>super[PropertySymbol.cloneNode](deep);
	}
}
