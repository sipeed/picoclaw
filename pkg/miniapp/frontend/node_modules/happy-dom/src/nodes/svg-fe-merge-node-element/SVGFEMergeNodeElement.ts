import SVGElement from '../svg-element/SVGElement.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import SVGAnimatedString from '../../svg/SVGAnimatedString.js';

/**
 * SVGFEMergeNodeElement.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGFEMergeNodeElement
 */
export default class SVGFEMergeNodeElement extends SVGElement {
	// Internal properties
	public [PropertySymbol.in1]: SVGAnimatedString | null = null;

	/**
	 * Returns in1.
	 *
	 * @returns In1.
	 */
	public get in1(): SVGAnimatedString {
		if (!this[PropertySymbol.in1]) {
			this[PropertySymbol.in1] = new SVGAnimatedString(
				PropertySymbol.illegalConstructor,
				this[PropertySymbol.window],
				{
					getAttribute: () => this.getAttribute('in'),
					setAttribute: (value) => this.setAttribute('in', value)
				}
			);
		}
		return this[PropertySymbol.in1];
	}
}
