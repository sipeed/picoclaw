import SVGElement from '../svg-element/SVGElement.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import SVGAnimatedNumber from '../../svg/SVGAnimatedNumber.js';
import SVGAnimatedEnumeration from '../../svg/SVGAnimatedEnumeration.js';
import SVGAnimatedNumberList from '../../svg/SVGAnimatedNumberList.js';

/**
 * SVGComponentTransferFunctionElement.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGComponentTransferFunctionElement
 */
export default class SVGComponentTransferFunctionElement extends SVGElement {
	// Static properties
	public static SVG_FECOMPONENTTRANSFER_TYPE_UNKNOWN = 0;
	public static SVG_FECOMPONENTTRANSFER_TYPE_IDENTITY = 1;
	public static SVG_FECOMPONENTTRANSFER_TYPE_TABLE = 2;
	public static SVG_FECOMPONENTTRANSFER_TYPE_DISCRETE = 3;
	public static SVG_FECOMPONENTTRANSFER_TYPE_LINEAR = 4;
	public static SVG_FECOMPONENTTRANSFER_TYPE_GAMMA = 5;

	// Internal properties
	public [PropertySymbol.type]: SVGAnimatedEnumeration | null = null;
	public [PropertySymbol.tableValues]: SVGAnimatedNumberList | null = null;
	public [PropertySymbol.slope]: SVGAnimatedNumber | null = null;
	public [PropertySymbol.intercept]: SVGAnimatedNumber | null = null;
	public [PropertySymbol.amplitude]: SVGAnimatedNumber | null = null;
	public [PropertySymbol.exponent]: SVGAnimatedNumber | null = null;
	public [PropertySymbol.offset]: SVGAnimatedNumber | null = null;

	/**
	 * Returns type.
	 *
	 * @returns Type.
	 */
	public get type(): SVGAnimatedEnumeration {
		if (!this[PropertySymbol.type]) {
			this[PropertySymbol.type] = new SVGAnimatedEnumeration(
				PropertySymbol.illegalConstructor,
				this[PropertySymbol.window],
				{
					getAttribute: () => this.getAttribute('type'),
					setAttribute: (value) => this.setAttribute('type', value),
					values: ['identity', 'table', 'discrete', 'linear', 'gamma'],
					defaultValue: 'identity'
				}
			);
		}
		return this[PropertySymbol.type];
	}

	/**
	 * Returns table values.
	 *
	 * @returns Table values.
	 */
	public get tableValues(): SVGAnimatedNumberList {
		if (!this[PropertySymbol.tableValues]) {
			this[PropertySymbol.tableValues] = new SVGAnimatedNumberList(
				PropertySymbol.illegalConstructor,
				this[PropertySymbol.window],
				{
					getAttribute: () => this.getAttribute('tableValues'),
					setAttribute: (value) => this.setAttribute('tableValues', value)
				}
			);
		}
		return this[PropertySymbol.tableValues];
	}

	/**
	 * Returns slope.
	 *
	 * @returns Slope.
	 */
	public get slope(): SVGAnimatedNumber {
		if (!this[PropertySymbol.slope]) {
			this[PropertySymbol.slope] = new SVGAnimatedNumber(
				PropertySymbol.illegalConstructor,
				this[PropertySymbol.window],
				{
					getAttribute: () => this.getAttribute('slope'),
					setAttribute: (value) => this.setAttribute('slope', value),
					defaultValue: 1
				}
			);
		}
		return this[PropertySymbol.slope];
	}

	/**
	 * Returns intercept.
	 *
	 * @returns Intercept.
	 */
	public get intercept(): SVGAnimatedNumber {
		if (!this[PropertySymbol.intercept]) {
			this[PropertySymbol.intercept] = new SVGAnimatedNumber(
				PropertySymbol.illegalConstructor,
				this[PropertySymbol.window],
				{
					getAttribute: () => this.getAttribute('intercept'),
					setAttribute: (value) => this.setAttribute('intercept', value)
				}
			);
		}
		return this[PropertySymbol.intercept];
	}

	/**
	 * Returns amplitude.
	 *
	 * @returns Amplitude.
	 */
	public get amplitude(): SVGAnimatedNumber {
		if (!this[PropertySymbol.amplitude]) {
			this[PropertySymbol.amplitude] = new SVGAnimatedNumber(
				PropertySymbol.illegalConstructor,
				this[PropertySymbol.window],
				{
					getAttribute: () => this.getAttribute('amplitude'),
					setAttribute: (value) => this.setAttribute('amplitude', value),
					defaultValue: 1
				}
			);
		}
		return this[PropertySymbol.amplitude];
	}

	/**
	 * Returns exponent.
	 *
	 * @returns Exponent.
	 */
	public get exponent(): SVGAnimatedNumber {
		if (!this[PropertySymbol.exponent]) {
			this[PropertySymbol.exponent] = new SVGAnimatedNumber(
				PropertySymbol.illegalConstructor,
				this[PropertySymbol.window],
				{
					getAttribute: () => this.getAttribute('exponent'),
					setAttribute: (value) => this.setAttribute('exponent', value),
					defaultValue: 1
				}
			);
		}
		return this[PropertySymbol.exponent];
	}

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
