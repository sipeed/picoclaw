import * as PropertySymbol from '../../PropertySymbol.js';
import SVGAnimatedNumber from '../../svg/SVGAnimatedNumber.js';
import SVGElement from '../svg-element/SVGElement.js';

/**
 * SVG Stop Element.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGStopElement
 */
export default class SVGStopElement extends SVGElement {
	// Internal properties
	public [PropertySymbol.offset]: SVGAnimatedNumber | null = null;

	/**
	 * Returns offset.
	 *
	 * @returns Offset.
	 */
	public get offset(): SVGAnimatedNumber {
		if (!this[PropertySymbol.offset]) {
			this[PropertySymbol.offset] = new SVGAnimatedNumber(
				PropertySymbol.illegalConstructor,
				this[PropertySymbol.window],
				{
					getAttribute: () => this.getAttribute('offset'),
					setAttribute: (value) => this.setAttribute('offset', value)
				}
			);
		}
		return this[PropertySymbol.offset];
	}
}
