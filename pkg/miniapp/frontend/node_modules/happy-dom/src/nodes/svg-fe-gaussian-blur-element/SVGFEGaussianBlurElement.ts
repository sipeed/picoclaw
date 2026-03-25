import SVGElement from '../svg-element/SVGElement.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import SVGAnimatedNumber from '../../svg/SVGAnimatedNumber.js';
import SVGAnimatedString from '../../svg/SVGAnimatedString.js';
import SVGAnimatedLength from '../../svg/SVGAnimatedLength.js';
import SVGAnimatedEnumeration from '../../svg/SVGAnimatedEnumeration.js';

/**
 * SVGFEGaussianBlurElement.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGFEGaussianBlurElement
 */
export default class SVGFEGaussianBlurElement extends SVGElement {
	// Static properties
	public static SVG_EDGEMODE_UNKNOWN = 0;
	public static SVG_EDGEMODE_DUPLICATE = 1;
	public static SVG_EDGEMODE_WRAP = 2;
	public static SVG_EDGEMODE_NONE = 3;

	// Internal properties
	public [PropertySymbol.edgeMode]: SVGAnimatedEnumeration | null = null;
	public [PropertySymbol.height]: SVGAnimatedLength | null = null;
	public [PropertySymbol.in1]: SVGAnimatedString | null = null;
	public [PropertySymbol.result]: SVGAnimatedString | null = null;
	public [PropertySymbol.stdDeviationX]: SVGAnimatedNumber | null = null;
	public [PropertySymbol.stdDeviationY]: SVGAnimatedNumber | null = null;
	public [PropertySymbol.width]: SVGAnimatedLength | null = null;
	public [PropertySymbol.x]: SVGAnimatedLength | null = null;
	public [PropertySymbol.y]: SVGAnimatedLength | null = null;

	/**
	 * Returns edge mode.
	 *
	 * @returns Edge mode.
	 */
	public get edgeMode(): SVGAnimatedEnumeration {
		if (!this[PropertySymbol.edgeMode]) {
			this[PropertySymbol.edgeMode] = new SVGAnimatedEnumeration(
				PropertySymbol.illegalConstructor,
				this[PropertySymbol.window],
				{
					getAttribute: () => this.getAttribute('edgeMode'),
					setAttribute: (value) => this.setAttribute('edgeMode', value),
					values: ['duplicate', 'wrap', 'none'],
					defaultValue: 'duplicate'
				}
			);
		}
		return this[PropertySymbol.edgeMode];
	}

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
	 * Returns stdDeviationX.
	 *
	 * @returns StdDeviationX.
	 */
	public get stdDeviationX(): SVGAnimatedNumber {
		if (!this[PropertySymbol.stdDeviationX]) {
			this[PropertySymbol.stdDeviationX] = new SVGAnimatedNumber(
				PropertySymbol.illegalConstructor,
				this[PropertySymbol.window],
				{
					getAttribute: () => this.getAttribute('stdDeviationX') || '2',
					setAttribute: (value) => this.setAttribute('stdDeviationX', value)
				}
			);
		}
		return this[PropertySymbol.stdDeviationX];
	}

	/**
	 * Returns stdDeviationY.
	 *
	 * @returns StdDeviationY.
	 */
	public get stdDeviationY(): SVGAnimatedNumber {
		if (!this[PropertySymbol.stdDeviationY]) {
			this[PropertySymbol.stdDeviationY] = new SVGAnimatedNumber(
				PropertySymbol.illegalConstructor,
				this[PropertySymbol.window],
				{
					getAttribute: () => this.getAttribute('stdDeviationY') || '2',
					setAttribute: (value) => this.setAttribute('stdDeviationY', value)
				}
			);
		}
		return this[PropertySymbol.stdDeviationY];
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

	/**
	 * Sets stdDeviation.
	 *
	 * @param x X.
	 * @param y Y.
	 */
	public setStdDeviation(x: number, y: number): void {
		this.stdDeviationX.baseVal = x;
		this.stdDeviationY.baseVal = y;
	}
}
