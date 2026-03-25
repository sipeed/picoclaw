import SVGElement from '../svg-element/SVGElement.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import SVGAnimatedString from '../../svg/SVGAnimatedString.js';
import SVGAnimatedLength from '../../svg/SVGAnimatedLength.js';
import SVGAnimatedNumber from '../../svg/SVGAnimatedNumber.js';
import SVGAnimatedEnumeration from '../../svg/SVGAnimatedEnumeration.js';

/**
 * SVGFEMorphologyElement.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGFEMorphologyElement
 */
export default class SVGFEMorphologyElement extends SVGElement {
	// Static properties
	public static readonly SVG_MORPHOLOGY_OPERATOR_UNKNOWN = 0;
	public static readonly SVG_MORPHOLOGY_OPERATOR_ERODE = 1;
	public static readonly SVG_MORPHOLOGY_OPERATOR_DILATE = 2;

	// Internal properties
	public [PropertySymbol.height]: SVGAnimatedLength | null = null;
	public [PropertySymbol.in1]: SVGAnimatedString | null = null;
	public [PropertySymbol.operator]: SVGAnimatedEnumeration | null = null;
	public [PropertySymbol.radiusX]: SVGAnimatedNumber | null = null;
	public [PropertySymbol.radiusY]: SVGAnimatedNumber | null = null;
	public [PropertySymbol.result]: SVGAnimatedString | null = null;
	public [PropertySymbol.width]: SVGAnimatedLength | null = null;
	public [PropertySymbol.x]: SVGAnimatedLength | null = null;
	public [PropertySymbol.y]: SVGAnimatedLength | null = null;

	/**
	 * Returns height.
	 *
	 * @returns Height.
	 */
	public get height(): SVGAnimatedLength {
		if (!this[PropertySymbol.height]) {
			this[PropertySymbol.height] = new SVGAnimatedLength(
				PropertySymbol.illegalConstructor,
				this[PropertySymbol.window],
				{
					getAttribute: () => this.getAttribute('height'),
					setAttribute: (value) => this.setAttribute('height', value)
				}
			);
		}
		return this[PropertySymbol.height];
	}

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

	/**
	 * Returns operator.
	 *
	 * @returns Operator.
	 */
	public get operator(): SVGAnimatedEnumeration {
		if (!this[PropertySymbol.operator]) {
			this[PropertySymbol.operator] = new SVGAnimatedEnumeration(
				PropertySymbol.illegalConstructor,
				this[PropertySymbol.window],
				{
					getAttribute: () => this.getAttribute('operator'),
					setAttribute: (value) => this.setAttribute('operator', value),
					values: ['erode', 'dilate'],
					defaultValue: 'erode'
				}
			);
		}
		return this[PropertySymbol.operator];
	}

	/**
	 * Returns radiusX.
	 *
	 * @returns RadiusX.
	 */
	public get radiusX(): SVGAnimatedNumber {
		if (!this[PropertySymbol.radiusX]) {
			this[PropertySymbol.radiusX] = new SVGAnimatedNumber(
				PropertySymbol.illegalConstructor,
				this[PropertySymbol.window],
				{
					getAttribute: () => this.getAttribute('radiusX'),
					setAttribute: (value) => this.setAttribute('radiusX', value)
				}
			);
		}
		return this[PropertySymbol.radiusX];
	}

	/**
	 * Returns radiusY.
	 *
	 * @returns RadiusY.
	 */
	public get radiusY(): SVGAnimatedNumber {
		if (!this[PropertySymbol.radiusY]) {
			this[PropertySymbol.radiusY] = new SVGAnimatedNumber(
				PropertySymbol.illegalConstructor,
				this[PropertySymbol.window],
				{
					getAttribute: () => this.getAttribute('radiusY'),
					setAttribute: (value) => this.setAttribute('radiusY', value)
				}
			);
		}
		return this[PropertySymbol.radiusY];
	}

	/**
	 * Returns result.
	 *
	 * @returns Result.
	 */
	public get result(): SVGAnimatedString {
		if (!this[PropertySymbol.result]) {
			this[PropertySymbol.result] = new SVGAnimatedString(
				PropertySymbol.illegalConstructor,
				this[PropertySymbol.window],
				{
					getAttribute: () => this.getAttribute('result'),
					setAttribute: (value) => this.setAttribute('result', value)
				}
			);
		}
		return this[PropertySymbol.result];
	}

	/**
	 * Returns width.
	 *
	 * @returns Width.
	 */
	public get width(): SVGAnimatedLength {
		if (!this[PropertySymbol.width]) {
			this[PropertySymbol.width] = new SVGAnimatedLength(
				PropertySymbol.illegalConstructor,
				this[PropertySymbol.window],
				{
					getAttribute: () => this.getAttribute('width'),
					setAttribute: (value) => this.setAttribute('width', value)
				}
			);
		}
		return this[PropertySymbol.width];
	}

	/**
	 * Returns x.
	 *
	 * @returns X.
	 */
	public get x(): SVGAnimatedLength {
		if (!this[PropertySymbol.x]) {
			this[PropertySymbol.x] = new SVGAnimatedLength(
				PropertySymbol.illegalConstructor,
				this[PropertySymbol.window],
				{
					getAttribute: () => this.getAttribute('x'),
					setAttribute: (value) => this.setAttribute('x', value)
				}
			);
		}
		return this[PropertySymbol.x];
	}

	/**
	 * Returns y.
	 *
	 * @returns Y.
	 */
	public get y(): SVGAnimatedLength {
		if (!this[PropertySymbol.y]) {
			this[PropertySymbol.y] = new SVGAnimatedLength(
				PropertySymbol.illegalConstructor,
				this[PropertySymbol.window],
				{
					getAttribute: () => this.getAttribute('y'),
					setAttribute: (value) => this.setAttribute('y', value)
				}
			);
		}
		return this[PropertySymbol.y];
	}
}
