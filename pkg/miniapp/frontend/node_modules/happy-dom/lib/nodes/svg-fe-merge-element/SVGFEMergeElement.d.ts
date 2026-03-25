import SVGElement from '../svg-element/SVGElement.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import SVGAnimatedString from '../../svg/SVGAnimatedString.js';
import SVGAnimatedLength from '../../svg/SVGAnimatedLength.js';
/**
 * SVGFEMergeElement.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGFEMergeElement
 */
export default class SVGFEMergeElement extends SVGElement {
    [PropertySymbol.height]: SVGAnimatedLength | null;
    [PropertySymbol.result]: SVGAnimatedString | null;
    [PropertySymbol.width]: SVGAnimatedLength | null;
    [PropertySymbol.x]: SVGAnimatedLength | null;
    [PropertySymbol.y]: SVGAnimatedLength | null;
    /**
     * Returns height.
     *
     * @returns Height.
     */
    get height(): SVGAnimatedLength;
    /**
     * Returns result.
     *
     * @returns Result.
     */
    get result(): SVGAnimatedString;
    /**
     * Returns width.
     *
     * @returns Width.
     */
    get width(): SVGAnimatedLength;
    /**
     * Returns x.
     *
     * @returns X.
     */
    get x(): SVGAnimatedLength;
    /**
     * Returns y.
     *
     * @returns Y.
     */
    get y(): SVGAnimatedLength;
}
//# sourceMappingURL=SVGFEMergeElement.d.ts.map