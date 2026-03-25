import SVGElement from '../svg-element/SVGElement.js';
import SVGStringList from '../../svg/SVGStringList.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import Event from '../../event/Event.js';

/**
 * SVG Animation Element.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGAnimationElement
 */
export default class SVGAnimationElement extends SVGElement {
	// Internal properties
	public [PropertySymbol.requiredExtensions]: SVGStringList | null = null;
	public [PropertySymbol.systemLanguage]: SVGStringList | null = null;

	// Events
	public onbegin: ((event: Event) => void) | null = null;
	public onend: ((event: Event) => void) | null = null;
	public onrepeat: ((event: Event) => void) | null = null;

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
	 * Returns target element.
	 *
	 * @returns Target element.
	 */
	public get targetElement(): SVGElement | null {
		// TODO: Implement targetElement
		return null;
	}
}
