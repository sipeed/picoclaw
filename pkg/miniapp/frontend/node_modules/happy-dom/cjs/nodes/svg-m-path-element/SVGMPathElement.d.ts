import SVGAnimatedString from '../../svg/SVGAnimatedString.cjs';
import SVGElement from '../svg-element/SVGElement.cjs';
import * as PropertySymbol from '../../PropertySymbol.cjs';
/**
 * SVG MPath Element.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGMPathElement
 */
export default class SVGMPathElement extends SVGElement {
    [PropertySymbol.href]: SVGAnimatedString | null;
    /**
     * Returns href.
     *
     * @returns Href.
     */
    get href(): SVGAnimatedString;
}
//# sourceMappingURL=SVGMPathElement.d.ts.map