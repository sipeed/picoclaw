import SVGElement from '../svg-element/SVGElement.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import SVGAnimatedNumber from '../../svg/SVGAnimatedNumber.js';

/**
 * SVGFEDistantLightElement.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGFEDistantLightElement
 */
export default class SVGFEDistantLightElement extends SVGElement {
	// Internal properties
	public [PropertySymbol.azimuth]: SVGAnimatedNumber | null = null;
	public [PropertySymbol.elevation]: SVGAnimatedNumber | null = null;

	/**
	 * Returns azimuth.
	 *
	 * @returns Azimuth.
	 */
	public get azimuth(): SVGAnimatedNumber {
		if (!this[PropertySymbol.azimuth]) {
			this[PropertySymbol.azimuth] = new SVGAnimatedNumber(
				PropertySymbol.illegalConstructor,
				this[PropertySymbol.window],
				{
					getAttribute: () => this.getAttribute('azimuth'),
					setAttribute: (value) => this.setAttribute('azimuth', value)
				}
			);
		}
		return this[PropertySymbol.azimuth];
	}

	/**
	 * Returns elevation.
	 *
	 * @returns Elevation.
	 */
	public get elevation(): SVGAnimatedNumber {
		if (!this[PropertySymbol.elevation]) {
			this[PropertySymbol.elevation] = new SVGAnimatedNumber(
				PropertySymbol.illegalConstructor,
				this[PropertySymbol.window],
				{
					getAttribute: () => this.getAttribute('elevation'),
					setAttribute: (value) => this.setAttribute('elevation', value)
				}
			);
		}
		return this[PropertySymbol.elevation];
	}
}
