import * as PropertySymbol from '../../PropertySymbol.js';
import SVGTextContentElement from '../svg-text-content-element/SVGTextContentElement.js';
import SVGAnimatedLengthList from '../../svg/SVGAnimatedLengthList.js';
import SVGAnimatedNumberList from '../../svg/SVGAnimatedNumberList.js';

/**
 * SVG Text Positioning Element.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGTextPositioningElement
 */
export default class SVGTextPositioningElement extends SVGTextContentElement {
	// Internal properties
	public [PropertySymbol.x]: SVGAnimatedLengthList | null = null;
	public [PropertySymbol.y]: SVGAnimatedLengthList | null = null;
	public [PropertySymbol.dx]: SVGAnimatedLengthList | null = null;
	public [PropertySymbol.dy]: SVGAnimatedLengthList | null = null;
	public [PropertySymbol.rotate]: SVGAnimatedNumberList | null = null;

	/**
	 * Returns x.
	 *
	 * @returns X.
	 */
	public get x(): SVGAnimatedLengthList {
		if (!this[PropertySymbol.x]) {
			this[PropertySymbol.x] = new SVGAnimatedLengthList(
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
	public get y(): SVGAnimatedLengthList {
		if (!this[PropertySymbol.y]) {
			this[PropertySymbol.y] = new SVGAnimatedLengthList(
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
	 * Returns dx.
	 *
	 * @returns DX.
	 */
	public get dx(): SVGAnimatedLengthList {
		if (!this[PropertySymbol.dx]) {
			this[PropertySymbol.dx] = new SVGAnimatedLengthList(
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
	 * @returns DY.
	 */
	public get dy(): SVGAnimatedLengthList {
		if (!this[PropertySymbol.dy]) {
			this[PropertySymbol.dy] = new SVGAnimatedLengthList(
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
	 * Returns rotate.
	 *
	 * @returns Rotate.
	 */
	public get rotate(): SVGAnimatedNumberList {
		if (!this[PropertySymbol.rotate]) {
			this[PropertySymbol.rotate] = new SVGAnimatedNumberList(
				PropertySymbol.illegalConstructor,
				this[PropertySymbol.window],
				{
					getAttribute: () => this.getAttribute('rotate'),
					setAttribute: (value) => this.setAttribute('rotate', value)
				}
			);
		}
		return this[PropertySymbol.rotate];
	}
}
