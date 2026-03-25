import SVGAnimatedString from '../../svg/SVGAnimatedString.js';
import SVGElement from '../svg-element/SVGElement.js';
import SVGAnimatedEnumeration from '../../svg/SVGAnimatedEnumeration.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import SVGAnimatedTransformList from '../../svg/SVGAnimatedTransformList.js';
import SVGAnimatedLength from '../../svg/SVGAnimatedLength.js';

/**
 * SVG Pattern Element.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGPatternElement
 */
export default class SVGPatternElement extends SVGElement {
	// Internal properties
	public [PropertySymbol.href]: SVGAnimatedString | null = null;
	public [PropertySymbol.patternUnits]: SVGAnimatedEnumeration | null = null;
	public [PropertySymbol.patternContentUnits]: SVGAnimatedEnumeration | null = null;
	public [PropertySymbol.patternTransform]: SVGAnimatedTransformList | null = null;
	public [PropertySymbol.x]: SVGAnimatedLength | null = null;
	public [PropertySymbol.y]: SVGAnimatedLength | null = null;
	public [PropertySymbol.width]: SVGAnimatedLength | null = null;
	public [PropertySymbol.height]: SVGAnimatedLength | null = null;

	/**
	 * Returns href.
	 *
	 * @returns Href.
	 */
	public get href(): SVGAnimatedString {
		if (!this[PropertySymbol.href]) {
			this[PropertySymbol.href] = new SVGAnimatedString(
				PropertySymbol.illegalConstructor,
				this[PropertySymbol.window],
				{
					getAttribute: () => this.getAttribute('href'),
					setAttribute: (value) => this.setAttribute('href', value)
				}
			);
		}
		return this[PropertySymbol.href];
	}

	/**
	 * Returns pattern units.
	 *
	 * @returns Pattern units.
	 */
	public get patternUnits(): SVGAnimatedEnumeration {
		if (!this[PropertySymbol.patternUnits]) {
			this[PropertySymbol.patternUnits] = new SVGAnimatedEnumeration(
				PropertySymbol.illegalConstructor,
				this[PropertySymbol.window],
				{
					getAttribute: () => this.getAttribute('patternUnits'),
					setAttribute: (value) => this.setAttribute('patternUnits', value),
					values: ['userSpaceOnUse', 'objectBoundingBox'],
					defaultValue: 'objectBoundingBox'
				}
			);
		}
		return this[PropertySymbol.patternUnits];
	}

	/**
	 * Returns pattern content units.
	 *
	 * @returns Pattern content units.
	 */
	public get patternContentUnits(): SVGAnimatedEnumeration {
		if (!this[PropertySymbol.patternContentUnits]) {
			this[PropertySymbol.patternContentUnits] = new SVGAnimatedEnumeration(
				PropertySymbol.illegalConstructor,
				this[PropertySymbol.window],
				{
					getAttribute: () => this.getAttribute('patternContentUnits'),
					setAttribute: (value) => this.setAttribute('patternContentUnits', value),
					values: ['userSpaceOnUse', 'objectBoundingBox'],
					defaultValue: 'userSpaceOnUse'
				}
			);
		}
		return this[PropertySymbol.patternContentUnits];
	}

	/**
	 * Returns pattern transform.
	 *
	 * @returns Pattern transform.
	 */
	public get patternTransform(): SVGAnimatedTransformList {
		if (!this[PropertySymbol.patternTransform]) {
			this[PropertySymbol.patternTransform] = new SVGAnimatedTransformList(
				PropertySymbol.illegalConstructor,
				this[PropertySymbol.window],
				{
					getAttribute: () => this.getAttribute('patternTransform'),
					setAttribute: (value) => this.setAttribute('patternTransform', value)
				}
			);
		}
		return this[PropertySymbol.patternTransform];
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
