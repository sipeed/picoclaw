import SVGElement from '../svg-element/SVGElement.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import SVGAnimatedLength from '../../svg/SVGAnimatedLength.js';
import SVGAnimatedString from '../../svg/SVGAnimatedString.js';
import SVGAnimatedEnumeration from '../../svg/SVGAnimatedEnumeration.js';

/**
 * SVG Filter Element.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGFilterElement
 */
export default class SVGFilterElement extends SVGElement {
	// Internal properties
	public [PropertySymbol.href]: SVGAnimatedString | null = null;
	public [PropertySymbol.filterUnits]: SVGAnimatedEnumeration | null = null;
	public [PropertySymbol.primitiveUnits]: SVGAnimatedEnumeration | null = null;
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
	 * Returns filter units.
	 *
	 * @returns Filter units.
	 */
	public get filterUnits(): SVGAnimatedEnumeration {
		if (!this[PropertySymbol.filterUnits]) {
			this[PropertySymbol.filterUnits] = new SVGAnimatedEnumeration(
				PropertySymbol.illegalConstructor,
				this[PropertySymbol.window],
				{
					getAttribute: () => this.getAttribute('filterUnits'),
					setAttribute: (value) => this.setAttribute('filterUnits', value),
					values: ['userSpaceOnUse', 'objectBoundingBox'],
					defaultValue: 'userSpaceOnUse'
				}
			);
		}
		return this[PropertySymbol.filterUnits];
	}

	/**
	 * Returns primitive units.
	 *
	 * @returns Primitive units.
	 */
	public get primitiveUnits(): SVGAnimatedEnumeration {
		if (!this[PropertySymbol.primitiveUnits]) {
			this[PropertySymbol.primitiveUnits] = new SVGAnimatedEnumeration(
				PropertySymbol.illegalConstructor,
				this[PropertySymbol.window],
				{
					getAttribute: () => this.getAttribute('primitiveUnits'),
					setAttribute: (value) => this.setAttribute('primitiveUnits', value),
					values: ['userSpaceOnUse', 'objectBoundingBox'],
					defaultValue: 'userSpaceOnUse'
				}
			);
		}
		return this[PropertySymbol.primitiveUnits];
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
