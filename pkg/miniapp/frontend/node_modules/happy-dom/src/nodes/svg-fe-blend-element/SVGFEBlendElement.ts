import SVGElement from '../svg-element/SVGElement.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import SVGAnimatedLength from '../../svg/SVGAnimatedLength.js';
import SVGAnimatedEnumeration from '../../svg/SVGAnimatedEnumeration.js';
import SVGAnimatedString from '../../svg/SVGAnimatedString.js';

/**
 * SVG FE Blend Element.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGFEBlendElement
 */
export default class SVGFEBlendElement extends SVGElement {
	// Static properties
	public static SVG_FEBLEND_MODE_UNKNOWN = 0;
	public static SVG_FEBLEND_MODE_NORMAL = 1;
	public static SVG_FEBLEND_MODE_MULTIPLY = 2;
	public static SVG_FEBLEND_MODE_SCREEN = 3;
	public static SVG_FEBLEND_MODE_DARKEN = 4;
	public static SVG_FEBLEND_MODE_LIGHTEN = 5;
	public static SVG_FEBLEND_MODE_OVERLAY = 6;
	public static SVG_FEBLEND_MODE_COLOR_DODGE = 7;
	public static SVG_FEBLEND_MODE_COLOR_BURN = 8;
	public static SVG_FEBLEND_MODE_HARD_LIGHT = 9;
	public static SVG_FEBLEND_MODE_SOFT_LIGHT = 10;
	public static SVG_FEBLEND_MODE_DIFFERENCE = 11;
	public static SVG_FEBLEND_MODE_EXCLUSION = 12;
	public static SVG_FEBLEND_MODE_HUE = 13;
	public static SVG_FEBLEND_MODE_SATURATION = 14;
	public static SVG_FEBLEND_MODE_COLOR = 15;
	public static SVG_FEBLEND_MODE_LUMINOSITY = 16;

	// Internal properties
	public [PropertySymbol.height]: SVGAnimatedLength | null = null;
	public [PropertySymbol.in1]: SVGAnimatedString | null = null;
	public [PropertySymbol.in2]: SVGAnimatedString | null = null;
	public [PropertySymbol.mode]: SVGAnimatedEnumeration | null = null;
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
	 * Returns mode.
	 *
	 * @returns Mode.
	 */
	public get mode(): SVGAnimatedEnumeration {
		if (!this[PropertySymbol.mode]) {
			this[PropertySymbol.mode] = new SVGAnimatedEnumeration(
				PropertySymbol.illegalConstructor,
				this[PropertySymbol.window],
				{
					getAttribute: () => this.getAttribute('mode'),
					setAttribute: (value) => this.setAttribute('mode', value),
					values: [
						'normal',
						'multiply',
						'screen',
						'darken',
						'lighten',
						'overlay',
						'color-dodge',
						'color-burn',
						'hard-light',
						'soft-light',
						'difference',
						'exclusion',
						'hue',
						'saturation',
						'color',
						'luminosity'
					],
					defaultValue: 'normal'
				}
			);
		}
		return this[PropertySymbol.mode];
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
