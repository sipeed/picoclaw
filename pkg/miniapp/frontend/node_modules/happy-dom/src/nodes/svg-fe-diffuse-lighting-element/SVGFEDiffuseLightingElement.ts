import SVGElement from '../svg-element/SVGElement.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import SVGAnimatedLength from '../../svg/SVGAnimatedLength.js';
import SVGAnimatedString from '../../svg/SVGAnimatedString.js';
import SVGAnimatedNumber from '../../svg/SVGAnimatedNumber.js';

/**
 * SVGFEDiffuseLightingElement.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGFEDiffuseLightingElement
 */
export default class SVGFEDiffuseLightingElement extends SVGElement {
	// Internal properties
	public [PropertySymbol.diffuseConstant]: SVGAnimatedNumber | null = null;
	public [PropertySymbol.height]: SVGAnimatedLength | null = null;
	public [PropertySymbol.in1]: SVGAnimatedString | null = null;
	public [PropertySymbol.kernelUnitLengthX]: SVGAnimatedNumber | null = null;
	public [PropertySymbol.kernelUnitLengthY]: SVGAnimatedNumber | null = null;
	public [PropertySymbol.result]: SVGAnimatedString | null = null;
	public [PropertySymbol.surfaceScale]: SVGAnimatedNumber | null = null;
	public [PropertySymbol.width]: SVGAnimatedLength | null = null;
	public [PropertySymbol.x]: SVGAnimatedLength | null = null;
	public [PropertySymbol.y]: SVGAnimatedLength | null = null;

	/**
	 * Returns diffuse constant.
	 *
	 * @returns Diffuse constant.
	 */
	public get diffuseConstant(): SVGAnimatedNumber {
		if (!this[PropertySymbol.diffuseConstant]) {
			this[PropertySymbol.diffuseConstant] = new SVGAnimatedNumber(
				PropertySymbol.illegalConstructor,
				this[PropertySymbol.window],
				{
					getAttribute: () => this.getAttribute('diffuseConstant'),
					setAttribute: (value) => this.setAttribute('diffuseConstant', value)
				}
			);
		}
		return this[PropertySymbol.diffuseConstant];
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
	 * Returns kernel unit length x.
	 *
	 * @returns Kernel unit length x.
	 */
	public get kernelUnitLengthX(): SVGAnimatedNumber {
		if (!this[PropertySymbol.kernelUnitLengthX]) {
			this[PropertySymbol.kernelUnitLengthX] = new SVGAnimatedNumber(
				PropertySymbol.illegalConstructor,
				this[PropertySymbol.window],
				{
					getAttribute: () => this.getAttribute('kernelUnitLengthX'),
					setAttribute: (value) => this.setAttribute('kernelUnitLengthX', value)
				}
			);
		}
		return this[PropertySymbol.kernelUnitLengthX];
	}

	/**
	 * Returns kernel unit length y.
	 *
	 * @returns Kernel unit length y.
	 */
	public get kernelUnitLengthY(): SVGAnimatedNumber {
		if (!this[PropertySymbol.kernelUnitLengthY]) {
			this[PropertySymbol.kernelUnitLengthY] = new SVGAnimatedNumber(
				PropertySymbol.illegalConstructor,
				this[PropertySymbol.window],
				{
					getAttribute: () => this.getAttribute('kernelUnitLengthY'),
					setAttribute: (value) => this.setAttribute('kernelUnitLengthY', value)
				}
			);
		}
		return this[PropertySymbol.kernelUnitLengthY];
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
	 * Returns surface scale.
	 *
	 * @returns Surface scale.
	 */
	public get surfaceScale(): SVGAnimatedNumber {
		if (!this[PropertySymbol.surfaceScale]) {
			this[PropertySymbol.surfaceScale] = new SVGAnimatedNumber(
				PropertySymbol.illegalConstructor,
				this[PropertySymbol.window],
				{
					getAttribute: () => this.getAttribute('surfaceScale'),
					setAttribute: (value) => this.setAttribute('surfaceScale', value)
				}
			);
		}
		return this[PropertySymbol.surfaceScale];
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
