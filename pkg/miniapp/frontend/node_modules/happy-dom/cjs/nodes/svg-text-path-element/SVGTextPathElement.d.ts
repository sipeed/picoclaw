import SVGTextContentElement from '../svg-text-content-element/SVGTextContentElement.cjs';
import * as PropertySymbol from '../../PropertySymbol.cjs';
import SVGAnimatedString from '../../svg/SVGAnimatedString.cjs';
import SVGAnimatedLength from '../../svg/SVGAnimatedLength.cjs';
import SVGAnimatedEnumeration from '../../svg/SVGAnimatedEnumeration.cjs';
/**
 * SVG Text Path Element.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGTextPathElement
 */
export default class SVGTextPathElement extends SVGTextContentElement {
    static readonly TEXTPATH_METHODTYPE_UNKNOWN = 0;
    static readonly TEXTPATH_METHODTYPE_ALIGN = 1;
    static readonly TEXTPATH_METHODTYPE_STRETCH = 2;
    static readonly TEXTPATH_SPACINGTYPE_UNKNOWN = 0;
    static readonly TEXTPATH_SPACINGTYPE_AUTO = 1;
    static readonly TEXTPATH_SPACINGTYPE_EXACT = 2;
    [PropertySymbol.href]: SVGAnimatedString | null;
    [PropertySymbol.startOffset]: SVGAnimatedLength | null;
    [PropertySymbol.method]: SVGAnimatedEnumeration | null;
    [PropertySymbol.spacing]: SVGAnimatedEnumeration | null;
    /**
     * Returns href.
     *
     * @returns Href.
     */
    get href(): SVGAnimatedString;
    /**
     * Returns start offset.
     *
     * @returns Start offset.
     */
    get startOffset(): SVGAnimatedLength;
    /**
     * Returns method.
     *
     * @returns ClipPathUnits.
     */
    get method(): SVGAnimatedEnumeration;
    /**
     * Returns spacing.
     *
     * @returns Spacing.
     */
    get spacing(): SVGAnimatedEnumeration;
}
//# sourceMappingURL=SVGTextPathElement.d.ts.map