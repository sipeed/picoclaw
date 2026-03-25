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
    static TEXTPATH_METHODTYPE_UNKNOWN = 0;
    static TEXTPATH_METHODTYPE_ALIGN = 1;
    static TEXTPATH_METHODTYPE_STRETCH = 2;
    static TEXTPATH_SPACINGTYPE_UNKNOWN = 0;
    static TEXTPATH_SPACINGTYPE_AUTO = 1;
    static TEXTPATH_SPACINGTYPE_EXACT = 2;
    // Internal properties
    [PropertySymbol.href] = null;
    [PropertySymbol.startOffset] = null;
    [PropertySymbol.method] = null;
    [PropertySymbol.spacing] = null;
    /**
     * Returns href.
     *
     * @returns Href.
     */
    get href() {
        if (!this[PropertySymbol.href]) {
            this[PropertySymbol.href] = new SVGAnimatedString(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('href'),
                setAttribute: (value) => this.setAttribute('href', value)
            });
        }
        return this[PropertySymbol.href];
    }
    /**
     * Returns start offset.
     *
     * @returns Start offset.
     */
    get startOffset() {
        if (!this[PropertySymbol.startOffset]) {
            this[PropertySymbol.startOffset] = new SVGAnimatedLength(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('startOffset'),
                setAttribute: (value) => this.setAttribute('startOffset', value)
            });
        }
        return this[PropertySymbol.startOffset];
    }
    /**
     * Returns method.
     *
     * @returns ClipPathUnits.
     */
    get method() {
        if (!this[PropertySymbol.method]) {
            this[PropertySymbol.method] = new SVGAnimatedEnumeration(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('method'),
                setAttribute: (value) => this.setAttribute('method', value),
                values: ['align', 'stretch'],
                defaultValue: 'align'
            });
        }
        return this[PropertySymbol.method];
    }
    /**
     * Returns spacing.
     *
     * @returns Spacing.
     */
    get spacing() {
        if (!this[PropertySymbol.spacing]) {
            this[PropertySymbol.spacing] = new SVGAnimatedEnumeration(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('spacing'),
                setAttribute: (value) => this.setAttribute('spacing', value),
                values: ['auto', 'exact'],
                defaultValue: 'exact'
            });
        }
        return this[PropertySymbol.spacing];
    }
}
//# sourceMappingURL=SVGTextPathElement.js.map