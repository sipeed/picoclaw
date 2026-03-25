import SVGTextContentElement from '../svg-text-content-element/SVGTextContentElement.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import SVGAnimatedString from '../../svg/SVGAnimatedString.js';
import SVGAnimatedLength from '../../svg/SVGAnimatedLength.js';
import SVGAnimatedEnumeration from '../../svg/SVGAnimatedEnumeration.js';

/**
 * SVG Text Path Element.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGTextPathElement
 */
export default class SVGTextPathElement extends SVGTextContentElement {
	// Public static properties
	public static readonly TEXTPATH_METHODTYPE_UNKNOWN = 0;
	public static readonly TEXTPATH_METHODTYPE_ALIGN = 1;
	public static readonly TEXTPATH_METHODTYPE_STRETCH = 2;
	public static readonly TEXTPATH_SPACINGTYPE_UNKNOWN = 0;
	public static readonly TEXTPATH_SPACINGTYPE_AUTO = 1;
	public static readonly TEXTPATH_SPACINGTYPE_EXACT = 2;

	// Internal properties
	public [PropertySymbol.href]: SVGAnimatedString | null = null;
	public [PropertySymbol.startOffset]: SVGAnimatedLength | null = null;
	public [PropertySymbol.method]: SVGAnimatedEnumeration | null = null;
	public [PropertySymbol.spacing]: SVGAnimatedEnumeration | null = null;

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
	 * Returns start offset.
	 *
	 * @returns Start offset.
	 */
	public get startOffset(): SVGAnimatedLength {
		if (!this[PropertySymbol.startOffset]) {
			this[PropertySymbol.startOffset] = new SVGAnimatedLength(
				PropertySymbol.illegalConstructor,
				this[PropertySymbol.window],
				{
					getAttribute: () => this.getAttribute('startOffset'),
					setAttribute: (value) => this.setAttribute('startOffset', value)
				}
			);
		}
		return this[PropertySymbol.startOffset];
	}

	/**
	 * Returns method.
	 *
	 * @returns ClipPathUnits.
	 */
	public get method(): SVGAnimatedEnumeration {
		if (!this[PropertySymbol.method]) {
			this[PropertySymbol.method] = new SVGAnimatedEnumeration(
				PropertySymbol.illegalConstructor,
				this[PropertySymbol.window],
				{
					getAttribute: () => this.getAttribute('method'),
					setAttribute: (value) => this.setAttribute('method', value),
					values: ['align', 'stretch'],
					defaultValue: 'align'
				}
			);
		}
		return this[PropertySymbol.method];
	}

	/**
	 * Returns spacing.
	 *
	 * @returns Spacing.
	 */
	public get spacing(): SVGAnimatedEnumeration {
		if (!this[PropertySymbol.spacing]) {
			this[PropertySymbol.spacing] = new SVGAnimatedEnumeration(
				PropertySymbol.illegalConstructor,
				this[PropertySymbol.window],
				{
					getAttribute: () => this.getAttribute('spacing'),
					setAttribute: (value) => this.setAttribute('spacing', value),
					values: ['auto', 'exact'],
					defaultValue: 'exact'
				}
			);
		}
		return this[PropertySymbol.spacing];
	}
}
