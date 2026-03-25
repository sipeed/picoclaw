import SVGElement from '../svg-element/SVGElement.cjs';
import * as PropertySymbol from '../../PropertySymbol.cjs';
import SVGAnimatedLength from '../../svg/SVGAnimatedLength.cjs';
import SVGAnimatedString from '../../svg/SVGAnimatedString.cjs';
/**
 * SVGFETileElement.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGFETileElement
 */
export default class SVGFETileElement extends SVGElement {
    [PropertySymbol.height]: SVGAnimatedLength | null;
    [PropertySymbol.in1]: SVGAnimatedString | null;
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
     * Returns in1.
     *
     * @returns In1.
     */
    get in1(): SVGAnimatedString;
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
//# sourceMappingURL=SVGFETileElement.d.ts.map