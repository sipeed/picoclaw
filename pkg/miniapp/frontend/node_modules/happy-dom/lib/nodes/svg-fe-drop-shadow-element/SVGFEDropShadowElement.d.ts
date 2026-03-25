import SVGElement from '../svg-element/SVGElement.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import SVGAnimatedNumber from '../../svg/SVGAnimatedNumber.js';
import SVGAnimatedString from '../../svg/SVGAnimatedString.js';
import SVGAnimatedLength from '../../svg/SVGAnimatedLength.js';
/**
 * SVGFEDropShadowElement.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGFEDropShadowElement
 */
export default class SVGFEDropShadowElement extends SVGElement {
    [PropertySymbol.dx]: SVGAnimatedNumber | null;
    [PropertySymbol.dy]: SVGAnimatedNumber | null;
    [PropertySymbol.height]: SVGAnimatedLength | null;
    [PropertySymbol.in1]: SVGAnimatedString | null;
    [PropertySymbol.result]: SVGAnimatedString | null;
    [PropertySymbol.stdDeviationX]: SVGAnimatedNumber | null;
    [PropertySymbol.stdDeviationY]: SVGAnimatedNumber | null;
    [PropertySymbol.width]: SVGAnimatedLength | null;
    [PropertySymbol.x]: SVGAnimatedLength | null;
    [PropertySymbol.y]: SVGAnimatedLength | null;
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
     * Returns stdDeviationX.
     *
     * @returns StdDeviationX.
     */
    get stdDeviationX(): SVGAnimatedNumber;
    /**
     * Returns stdDeviationY.
     *
     * @returns StdDeviationY.
     */
    get stdDeviationY(): SVGAnimatedNumber;
    /**
     * Returns width.
     *
     * @returns Width.
     */
    get width(): SVGAnimatedLength;
    /**
     * Returns x position.
     *
     * @returns X position.
     */
    get x(): SVGAnimatedLength;
    /**
     * Returns y position.
     *
     * @returns Y position.
     */
    get y(): SVGAnimatedLength;
    /**
     * Sets stdDeviation.
     *
     * @param x X.
     * @param y Y.
     */
    setStdDeviation(x: number, y: number): void;
}
//# sourceMappingURL=SVGFEDropShadowElement.d.ts.map