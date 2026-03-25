import SVGElement from '../svg-element/SVGElement.js';
import DOMRect from '../../dom/DOMRect.js';
import DOMMatrix from '../../dom/dom-matrix/DOMMatrix.js';
import SVGStringList from '../../svg/SVGStringList.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import SVGAnimatedTransformList from '../../svg/SVGAnimatedTransformList.js';
import Event from '../../event/Event.js';

/**
 * SVG Graphics Element.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGGraphicsElement
 */
export default class SVGGraphicsElement extends SVGElement {
	// Internal properties
	public [PropertySymbol.requiredExtensions]: SVGStringList | null = null;
	public [PropertySymbol.systemLanguage]: SVGStringList | null = null;
	public [PropertySymbol.transform]: SVGAnimatedTransformList | null = null;

	// Events
	public oncopy: ((event: Event) => void) | null = null;
	public oncut: ((event: Event) => void) | null = null;
	public onpaste: ((event: Event) => void) | null = null;

	/**
	 * Returns required extensions.
	 *
	 * @returns Required extensions.
	 */
	public get requiredExtensions(): SVGStringList {
		if (!this[PropertySymbol.requiredExtensions]) {
			this[PropertySymbol.requiredExtensions] = new SVGStringList(
				PropertySymbol.illegalConstructor,
				this[PropertySymbol.window],
				{
					getAttribute: () => this.getAttribute('requiredExtensions'),
					setAttribute: (value) => this.setAttribute('requiredExtensions', value)
				}
			);
		}
		return this[PropertySymbol.requiredExtensions];
	}

	/**
	 * Returns system language.
	 *
	 * @returns System language.
	 */
	public get systemLanguage(): SVGStringList {
		if (!this[PropertySymbol.systemLanguage]) {
			this[PropertySymbol.systemLanguage] = new SVGStringList(
				PropertySymbol.illegalConstructor,
				this[PropertySymbol.window],
				{
					getAttribute: () => this.getAttribute('systemLanguage'),
					setAttribute: (value) => this.setAttribute('systemLanguage', value)
				}
			);
		}
		return this[PropertySymbol.systemLanguage];
	}

	/**
	 * Returns transform.
	 *
	 * @returns Transform.
	 */
	public get transform(): SVGAnimatedTransformList {
		if (!this[PropertySymbol.transform]) {
			this[PropertySymbol.transform] = new SVGAnimatedTransformList(
				PropertySymbol.illegalConstructor,
				this[PropertySymbol.window],
				{
					getAttribute: () => this.getAttribute('transform'),
					setAttribute: (value) => this.setAttribute('transform', value)
				}
			);
		}
		return this[PropertySymbol.transform];
	}

	/**
	 * Returns DOM rect.
	 *
	 * @returns DOM rect.
	 */
	public getBBox(): DOMRect {
		return new DOMRect();
	}

	/**
	 * Returns CTM.
	 *
	 * @returns CTM.
	 */
	public getCTM(): DOMMatrix {
		return new DOMMatrix();
	}

	/**
	 * Returns screen CTM.
	 *
	 * @returns Screen CTM.
	 */
	public getScreenCTM(): DOMMatrix {
		return new DOMMatrix();
	}
}
