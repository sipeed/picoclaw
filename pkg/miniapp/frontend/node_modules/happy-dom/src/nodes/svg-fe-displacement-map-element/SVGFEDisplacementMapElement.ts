import SVGElement from '../svg-element/SVGElement.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import SVGAnimatedLength from '../../svg/SVGAnimatedLength.js';
import SVGAnimatedString from '../../svg/SVGAnimatedString.js';
import SVGAnimatedNumber from '../../svg/SVGAnimatedNumber.js';
import SVGAnimatedEnumeration from '../../svg/SVGAnimatedEnumeration.js';

/**
 * SVGFEDisplacementMapElement.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGFEDisplacementMapElement
 */
export default class SVGFEDisplacementMapElement extends SVGElement {
	// Static properties
	public static SVG_CHANNEL_UNKNOWN = 0;
	public static SVG_CHANNEL_R = 1;
	public static SVG_CHANNEL_G = 2;
	public static SVG_CHANNEL_B = 3;
	public static SVG_CHANNEL_A = 4;

	// Internal properties
	public [PropertySymbol.height]: SVGAnimatedLength | null = null;
	public [PropertySymbol.in1]: SVGAnimatedString | null = null;
	public [PropertySymbol.in2]: SVGAnimatedString | null = null;
	public [PropertySymbol.result]: SVGAnimatedString | null = null;
	public [PropertySymbol.scale]: SVGAnimatedNumber | null = null;
	public [PropertySymbol.width]: SVGAnimatedLength | null = null;
	public [PropertySymbol.x]: SVGAnimatedLength | null = null;
	public [PropertySymbol.xChannelSelector]: SVGAnimatedEnumeration | null = null;
	public [PropertySymbol.y]: SVGAnimatedLength | null = null;
	public [PropertySymbol.yChannelSelector]: SVGAnimatedEnumeration | null = null;

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
	 * Returns scale.
	 *
	 * @returns Scale.
	 */
	public get scale(): SVGAnimatedNumber {
		if (!this[PropertySymbol.scale]) {
			this[PropertySymbol.scale] = new SVGAnimatedNumber(
				PropertySymbol.illegalConstructor,
				this[PropertySymbol.window],
				{
					getAttribute: () => this.getAttribute('scale'),
					setAttribute: (value) => this.setAttribute('scale', value)
				}
			);
		}
		return this[PropertySymbol.scale];
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
	 * Returns x channel selector.
	 *
	 * @returns X channel selector.
	 */
	public get xChannelSelector(): SVGAnimatedEnumeration {
		if (!this[PropertySymbol.xChannelSelector]) {
			this[PropertySymbol.xChannelSelector] = new SVGAnimatedEnumeration(
				PropertySymbol.illegalConstructor,
				this[PropertySymbol.window],
				{
					getAttribute: () => this.getAttribute('xChannelSelector'),
					setAttribute: (value) => this.setAttribute('xChannelSelector', value),
					values: ['r', 'g', 'b', 'a'],
					defaultValue: 'r'
				}
			);
		}
		return this[PropertySymbol.xChannelSelector];
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
	 * Returns y channel selector.
	 *
	 * @returns Y channel selector.
	 */
	public get yChannelSelector(): SVGAnimatedEnumeration {
		if (!this[PropertySymbol.yChannelSelector]) {
			this[PropertySymbol.yChannelSelector] = new SVGAnimatedEnumeration(
				PropertySymbol.illegalConstructor,
				this[PropertySymbol.window],
				{
					getAttribute: () => this.getAttribute('yChannelSelector'),
					setAttribute: (value) => this.setAttribute('yChannelSelector', value),
					values: ['r', 'g', 'b', 'a'],
					defaultValue: 'r'
				}
			);
		}
		return this[PropertySymbol.yChannelSelector];
	}
}
