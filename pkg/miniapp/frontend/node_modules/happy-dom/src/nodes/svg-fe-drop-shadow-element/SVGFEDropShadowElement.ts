import SVGElement from '../svg-element/SVGElement.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import SVGAnimatedNumber from '../../svg/SVGAnimatedNumber.js';
import SVGAnimatedString from '../../svg/SVGAnimatedString.js';
import SVGAnimatedLength from '../../svg/SVGAnimatedLength.js';

/**
 * SVGFEDropShadowElement.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGFEDropShadowElement
 */
export default class SVGFEDropShadowElement extends SVGElement {
	// Internal properties
	public [PropertySymbol.dx]: SVGAnimatedNumber | null = null;
	public [PropertySymbol.dy]: SVGAnimatedNumber | null = null;
	public [PropertySymbol.height]: SVGAnimatedLength | null = null;
	public [PropertySymbol.in1]: SVGAnimatedString | null = null;
	public [PropertySymbol.result]: SVGAnimatedString | null = null;
	public [PropertySymbol.stdDeviationX]: SVGAnimatedNumber | null = null;
	public [PropertySymbol.stdDeviationY]: SVGAnimatedNumber | null = null;
	public [PropertySymbol.width]: SVGAnimatedLength | null = null;
	public [PropertySymbol.x]: SVGAnimatedLength | null = null;
	public [PropertySymbol.y]: SVGAnimatedLength | null = null;

	/**
	 * Returns dx.
	 *
	 * @returns Dx.
	 */
	public get dx(): SVGAnimatedNumber {
		if (!this[PropertySymbol.dx]) {
			this[PropertySymbol.dx] = new SVGAnimatedNumber(
				PropertySymbol.illegalConstructor,
				this[PropertySymbol.window],
				{
					getAttribute: () => this.getAttribute('dx'),
					setAttribute: (value) => this.setAttribute('dx', value)
				}
			);
		}
		return this[PropertySymbol.dx];
	}

	/**
	 * Returns dy.
	 *
	 * @returns Dy.
	 */
	public get dy(): SVGAnimatedNumber {
		if (!this[PropertySymbol.dy]) {
			this[PropertySymbol.dy] = new SVGAnimatedNumber(
				PropertySymbol.illegalConstructor,
				this[PropertySymbol.window],
				{
					getAttribute: () => this.getAttribute('dy'),
					setAttribute: (value) => this.setAttribute('dy', value)
				}
			);
		}
		return this[PropertySymbol.dy];
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
					getAttribute: () => this.getAttribute('stdDeviationX'),
					setAttribute: (value) => this.setAttribute('stdDeviationX', value),
					defaultValue: 2
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
					getAttribute: () => this.getAttribute('stdDeviationY'),
					setAttribute: (value) => this.setAttribute('stdDeviationY', value),
					defaultValue: 2
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
