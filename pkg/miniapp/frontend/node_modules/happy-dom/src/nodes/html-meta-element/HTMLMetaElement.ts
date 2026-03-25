import HTMLElement from '../html-element/HTMLElement.js';

/**
 * HTML Meta Element.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/HTMLMetaElement.
 */
export default class HTMLMetaElement extends HTMLElement {
	/**
	 * Returns content.
	 *
	 * @returns Content.
	 */
	public get content(): string {
		return this.getAttribute('content') || '';
	}

	/**
	 * Sets content.
	 *
	 * @param content Content.
	 */
	public set content(content: string) {
		this.setAttribute('content', content);
	}

	/**
	 * Returns httpEquiv.
	 *
	 * @returns HttpEquiv.
	 */
	public get httpEquiv(): string {
		return this.getAttribute('http-equiv') || '';
	}

	/**
	 * Sets httpEquiv.
	 *
	 * @param httpEquiv HttpEquiv.
	 */
	public set httpEquiv(httpEquiv: string) {
		this.setAttribute('http-equiv', httpEquiv);
	}

	/**
	 * Returns name.
	 *
	 * @returns Name.
	 */
	public get name(): string {
		return this.getAttribute('name') || '';
	}

	/**
	 * Sets name.
	 *
	 * @param name Name.
	 */
	public set name(name: string) {
		this.setAttribute('name', name);
	}
	/**
	 * Returns scheme.
	 *
	 * @returns Name.
	 */
	public get scheme(): string {
		return this.getAttribute('scheme') || '';
	}

	/**
	 * Sets scheme.
	 *
	 * @param scheme Scheme.
	 */
	public set scheme(scheme: string) {
		this.setAttribute('scheme', scheme);
	}
}
