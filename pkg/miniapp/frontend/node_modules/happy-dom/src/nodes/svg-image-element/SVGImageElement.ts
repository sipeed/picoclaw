import * as PropertySymbol from '../../PropertySymbol.js';
import SVGAnimatedLength from '../../svg/SVGAnimatedLength.js';
import SVGAnimatedPreserveAspectRatio from '../../svg/SVGAnimatedPreserveAspectRatio.js';
import SVGAnimatedString from '../../svg/SVGAnimatedString.js';
import SVGGraphicsElement from '../svg-graphics-element/SVGGraphicsElement.js';

/**
 * SVG Image Element.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGImageElement
 */
export default class SVGImageElement extends SVGGraphicsElement {
	// Internal properties
	public [PropertySymbol.href]: SVGAnimatedString | null = null;
	public [PropertySymbol.preserveAspectRatio]: SVGAnimatedPreserveAspectRatio | null = null;
	public [PropertySymbol.x]: SVGAnimatedLength | null = null;
	public [PropertySymbol.y]: SVGAnimatedLength | null = null;
	public [PropertySymbol.width]: SVGAnimatedLength | null = null;
	public [PropertySymbol.height]: SVGAnimatedLength | null = null;

	/**
	 * Returns "crossorigin" attribute.
	 *
	 * @returns Cross origin.
	 */
	public get crossOrigin(): string {
		return this.getAttribute('crossorigin');
	}

	/**
	 * Sets "crossorigin" attribute.
	 *
	 * @param value Cross origin.
	 */
	public set crossOrigin(value: string) {
		this.setAttribute('crossorigin', value);
	}

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
	 * Returns decoding.
	 *
	 * @returns Decoding.
	 */
	public get decoding(): string {
		const value = this.getAttribute('decoding');
		switch (value) {
			case 'sync':
			case 'async':
			case 'auto':
				return value;
			default:
				return 'auto';
		}
	}

	/**
	 * Sets decoding.
	 *
	 * @param value Decoding.
	 */
	public set decoding(value: string) {
		this.setAttribute('decoding', value);
	}

	/**
	 * Returns preserve aspect ratio.
	 *
	 * @returns Preserve aspect ratio.
	 */
	public get preserveAspectRatio(): SVGAnimatedPreserveAspectRatio {
		if (!this[PropertySymbol.preserveAspectRatio]) {
			this[PropertySymbol.preserveAspectRatio] = new SVGAnimatedPreserveAspectRatio(
				PropertySymbol.illegalConstructor,
				this[PropertySymbol.window],
				{
					getAttribute: () => this.getAttribute('preserveAspectRatio'),
					setAttribute: (value) => this.setAttribute('preserveAspectRatio', value)
				}
			);
		}
		return this[PropertySymbol.preserveAspectRatio];
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

	/**
	 * Decodes the image.
	 *
	 * @returns Promise.
	 */
	public decode(): Promise<void> {
		// TODO: Implement decode()
		return Promise.resolve();
	}
}
