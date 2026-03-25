import SVGElement from '../svg-element/SVGElement.cjs';
import * as PropertySymbol from '../../PropertySymbol.cjs';
import SVGAnimatedString from '../../svg/SVGAnimatedString.cjs';
import SVGAnimatedLength from '../../svg/SVGAnimatedLength.cjs';
import SVGAnimatedNumber from '../../svg/SVGAnimatedNumber.cjs';
/**
 * SVGFEOffsetElement.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGFEOffsetElement
 */
export default class SVGFEOffsetElement extends SVGElement {
    [PropertySymbol.height]: SVGAnimatedLength | null;
    [PropertySymbol.in1]: SVGAnimatedString | null;
    [PropertySymbol.dx]: SVGAnimatedNumber | null;
    [PropertySymbol.dy]: SVGAnimatedNumber | null;
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
     * Returns dx.
     *
     * @returns Dx.
     */
    get dx(): SVGAnimatedNumber;
    /**
     * Returns dy.
     *
     * @returns Dy.
     */
    get dy(): SVGAnimatedNumber;
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
//# sourceMappingURL=SVGFEOffsetElement.d.ts.map