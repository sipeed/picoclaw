import SVGElement from '../svg-element/SVGElement.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import SVGAnimatedLength from '../../svg/SVGAnimatedLength.js';
import SVGAnimatedEnumeration from '../../svg/SVGAnimatedEnumeration.js';
import SVGAnimatedString from '../../svg/SVGAnimatedString.js';
import SVGAnimatedNumberList from '../../svg/SVGAnimatedNumberList.js';

/**
 * SVG FE Color Matrix Element.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGFEColorMatrixElement
 */
export default class SVGFEColorMatrixElement extends SVGElement {
	// Static properties
	public static SVG_FEBLEND_TYPE_UNKNOWN = 0;
	public static SVG_FEBLEND_TYPE_MATRIX = 1;
	public static SVG_FEBLEND_TYPE_SATURATE = 2;
	public static SVG_FEBLEND_TYPE_HUEROTATE = 3;
	public static SVG_FEBLEND_TYPE_LUMINANCETOALPHA = 4;

	// Internal properties
	public [PropertySymbol.height]: SVGAnimatedLength | null = null;
	public [PropertySymbol.in1]: SVGAnimatedString | null = null;
	public [PropertySymbol.in2]: SVGAnimatedString | null = null;
	public [PropertySymbol.result]: SVGAnimatedString | null = null;
	public [PropertySymbol.type]: SVGAnimatedEnumeration | null = null;
	public [PropertySymbol.values]: SVGAnimatedNumberList | null = null;
	public [PropertySymbol.x]: SVGAnimatedLength | null = null;
	public [PropertySymbol.y]: SVGAnimatedLength | null = null;
	public [PropertySymbol.width]: SVGAnimatedLength | null = null;

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
	 * Returns in2.
	 *
	 * @returns In2.
	 */
	public get in2(): SVGAnimatedString {
		if (!this[PropertySymbol.in2]) {
			this[PropertySymbol.in2] = new SVGAnimatedString(
				PropertySymbol.illegalConstructor,
				this[PropertySymbol.window],
				{
					getAttribute: () => this.getAttribute('in2'),
					setAttribute: (value) => this.setAttribute('in2', value)
				}
			);
		}
		return this[PropertySymbol.in2];
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
					values: ['matrix', 'saturate', 'huerotate', 'luminancetoalpha'],
					defaultValue: 'matrix'
				}
			);
		}
		return this[PropertySymbol.type];
	}

	/**
	 * Returns values.
	 *
	 * @returns Values.
	 */
	public get values(): SVGAnimatedNumberList {
		if (!this[PropertySymbol.values]) {
			this[PropertySymbol.values] = new SVGAnimatedNumberList(
				PropertySymbol.illegalConstructor,
				this[PropertySymbol.window],
				{
					getAttribute: () => this.getAttribute('values'),
					setAttribute: (value) => this.setAttribute('values', value)
				}
			);
		}
		return this[PropertySymbol.values];
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
	 * Returns x position.
	 *
	 * @returns X position.
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
	 * Returns y position.
	 *
	 * @returns Y position.
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
