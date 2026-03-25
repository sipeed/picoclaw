import CSSStyleDeclaration from '../../css/declaration/CSSStyleDeclaration.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import Element from '../element/Element.js';
import SVGSVGElement from '../svg-svg-element/SVGSVGElement.js';
import Event from '../../event/Event.js';
import HTMLElementUtility from '../html-element/HTMLElementUtility.js';
import DOMStringMap from '../../dom/DOMStringMap.js';

/**
 * SVG Element.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/SVGElement.
 */
export default class SVGElement extends Element {
	// Events
	public onabort: ((event: Event) => void) | null = null;
	public onerror: ((event: Event) => void) | null = null;
	public onload: ((event: Event) => void) | null = null;
	public onresize: ((event: Event) => void) | null = null;
	public onscroll: ((event: Event) => void) | null = null;
	public onunload: ((event: Event) => void) | null = null;

	// Internal properties
	public [PropertySymbol.style]: CSSStyleDeclaration | null = null;

	// Private properties
	#dataset: DOMStringMap | null = null;

	/**
	 * Returns current translate.
	 *
	 * @returns Element.
	 */
	public get ownerSVGElement(): SVGSVGElement | null {
		let parent = this[PropertySymbol.parentNode];
		while (parent) {
			if (parent[PropertySymbol.localName] === 'svg') {
				return <SVGSVGElement>parent;
			}

			parent = parent[PropertySymbol.parentNode];
		}
		return null;
	}

	/**
	 * Returns the SVGElement which established the current viewport. Often the nearest ancestor <svg> element. null if the given element is the outermost <svg> element.
	 *
	 * @returns SVG element.
	 */
	public get viewportElement(): SVGElement | null {
		return this.ownerSVGElement;
	}

	/**
	 * Returns data set.
	 *
	 * @returns Data set.
	 */
	public get dataset(): DOMStringMap {
		return (this.#dataset ??= new DOMStringMap(PropertySymbol.illegalConstructor, this));
	}

	/**
	 * Returns style.
	 *
	 * @returns Style.
	 */
	public get style(): CSSStyleDeclaration {
		if (!this[PropertySymbol.style]) {
			this[PropertySymbol.style] = new CSSStyleDeclaration(
				PropertySymbol.illegalConstructor,
				this[PropertySymbol.window],
				{ element: this }
			);
		}
		return this[PropertySymbol.style];
	}

	/**
	 * Returns tab index.
	 *
	 * @returns Tab index.
	 */
	public get tabIndex(): number {
		const tabIndex = this.getAttribute('tabindex');
		return tabIndex !== null ? Number(tabIndex) : -1;
	}

	/**
	 * Returns tab index.
	 *
	 * @param tabIndex Tab index.
	 */
	public set tabIndex(tabIndex: number) {
		if (tabIndex === -1) {
			this.removeAttribute('tabindex');
		} else {
			this.setAttribute('tabindex', String(tabIndex));
		}
	}

	/**
	 * Triggers a blur event.
	 */
	public blur(): void {
		HTMLElementUtility.blur(this);
	}

	/**
	 * Triggers a focus event.
	 */
	public focus(): void {
		HTMLElementUtility.focus(this);
	}
}
