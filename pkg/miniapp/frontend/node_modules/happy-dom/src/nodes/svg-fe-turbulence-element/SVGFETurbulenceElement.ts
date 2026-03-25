import SVGElement from '../svg-element/SVGElement.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import SVGAnimatedLength from '../../svg/SVGAnimatedLength.js';
import SVGAnimatedString from '../../svg/SVGAnimatedString.js';
import SVGAnimatedNumber from '../../svg/SVGAnimatedNumber.js';
import SVGAnimatedInteger from '../../svg/SVGAnimatedInteger.js';
import SVGAnimatedEnumeration from '../../svg/SVGAnimatedEnumeration.js';

/**
 * SVGFETurbulenceElement.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGFETurbulenceElement
 */
export default class SVGFETurbulenceElement extends SVGElement {
	// Static properties
	public static readonly SVG_TURBULENCE_TYPE_UNKNOWN = 0;
	public static readonly SVG_TURBULENCE_TYPE_FRACTALNOISE = 1;
	public static readonly SVG_TURBULENCE_TYPE_TURBULENCE = 2;

	public static readonly SVG_STITCHTYPE_UNKNOWN = 0;
	public static readonly SVG_STITCHTYPE_STITCH = 1;
	public static readonly SVG_STITCHTYPE_NOSTITCH = 2;

	// Internal properties
	public [PropertySymbol.baseFrequencyX]: SVGAnimatedNumber | null = null;
	public [PropertySymbol.baseFrequencyY]: SVGAnimatedNumber | null = null;
	public [PropertySymbol.height]: SVGAnimatedLength | null = null;
	public [PropertySymbol.numOctaves]: SVGAnimatedInteger | null = null;
	public [PropertySymbol.result]: SVGAnimatedString | null = null;
	public [PropertySymbol.seed]: SVGAnimatedNumber | null = null;
	public [PropertySymbol.stitchTiles]: SVGAnimatedEnumeration | null = null;
	public [PropertySymbol.type]: SVGAnimatedEnumeration | null = null;
	public [PropertySymbol.width]: SVGAnimatedLength | null = null;
	public [PropertySymbol.x]: SVGAnimatedLength | null = null;
	public [PropertySymbol.y]: SVGAnimatedLength | null = null;

	/**
	 * Returns baseFrequencyX.
	 *
	 * @returns Base frequency x.
	 */
	public get baseFrequencyX(): SVGAnimatedNumber {
		if (!this[PropertySymbol.baseFrequencyX]) {
			this[PropertySymbol.baseFrequencyX] = new SVGAnimatedNumber(
				PropertySymbol.illegalConstructor,
				this[PropertySymbol.window],
				{
					getAttribute: () => this.getAttribute('baseFrequencyX'),
					setAttribute: (value) => this.setAttribute('baseFrequencyX', value)
				}
			);
		}
		return this[PropertySymbol.baseFrequencyX];
	}

	/**
	 * Returns baseFrequencyY.
	 *
	 * @returns Base frequency y.
	 */
	public get baseFrequencyY(): SVGAnimatedNumber {
		if (!this[PropertySymbol.baseFrequencyY]) {
			this[PropertySymbol.baseFrequencyY] = new SVGAnimatedNumber(
				PropertySymbol.illegalConstructor,
				this[PropertySymbol.window],
				{
					getAttribute: () => this.getAttribute('baseFrequencyY'),
					setAttribute: (value) => this.setAttribute('baseFrequencyY', value)
				}
			);
		}
		return this[PropertySymbol.baseFrequencyY];
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
	 * Returns numOctaves.
	 *
	 * @returns Num octaves.
	 */
	public get numOctaves(): SVGAnimatedInteger {
		if (!this[PropertySymbol.numOctaves]) {
			this[PropertySymbol.numOctaves] = new SVGAnimatedInteger(
				PropertySymbol.illegalConstructor,
				this[PropertySymbol.window],
				{
					getAttribute: () => this.getAttribute('numOctaves'),
					setAttribute: (value) => this.setAttribute('numOctaves', value)
				}
			);
		}
		return this[PropertySymbol.numOctaves];
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
	 * Returns seed.
	 *
	 * @returns Seed.
	 */
	public get seed(): SVGAnimatedNumber {
		if (!this[PropertySymbol.seed]) {
			this[PropertySymbol.seed] = new SVGAnimatedNumber(
				PropertySymbol.illegalConstructor,
				this[PropertySymbol.window],
				{
					getAttribute: () => this.getAttribute('seed'),
					setAttribute: (value) => this.setAttribute('seed', value)
				}
			);
		}
		return this[PropertySymbol.seed];
	}

	/**
	 * Returns stitchTiles.
	 *
	 * @returns Stitch tiles.
	 */
	public get stitchTiles(): SVGAnimatedEnumeration {
		if (!this[PropertySymbol.stitchTiles]) {
			this[PropertySymbol.stitchTiles] = new SVGAnimatedEnumeration(
				PropertySymbol.illegalConstructor,
				this[PropertySymbol.window],
				{
					getAttribute: () => this.getAttribute('stitchTiles'),
					setAttribute: (value) => this.setAttribute('stitchTiles', value),
					values: ['stitch', 'noStitch'],
					defaultValue: 'stitch'
				}
			);
		}
		return this[PropertySymbol.stitchTiles];
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
					values: ['fractalNoise', 'turbulence'],
					defaultValue: 'turbulence'
				}
			);
		}
		return this[PropertySymbol.type];
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
