import SVGElement from '../svg-element/SVGElement.cjs';
import * as PropertySymbol from '../../PropertySymbol.cjs';
import SVGAnimatedNumber from '../../svg/SVGAnimatedNumber.cjs';
import SVGAnimatedString from '../../svg/SVGAnimatedString.cjs';
import SVGAnimatedLength from '../../svg/SVGAnimatedLength.cjs';
import SVGAnimatedEnumeration from '../../svg/SVGAnimatedEnumeration.cjs';
/**
 * SVGFEGaussianBlurElement.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGFEGaussianBlurElement
 */
export default class SVGFEGaussianBlurElement extends SVGElement {
    static SVG_EDGEMODE_UNKNOWN: number;
    static SVG_EDGEMODE_DUPLICATE: number;
    static SVG_EDGEMODE_WRAP: number;
    static SVG_EDGEMODE_NONE: number;
    [PropertySymbol.edgeMode]: SVGAnimatedEnumeration | null;
    [PropertySymbol.height]: SVGAnimatedLength | null;
    [PropertySymbol.in1]: SVGAnimatedString | null;
    [PropertySymbol.result]: SVGAnimatedString | null;
    [PropertySymbol.stdDeviationX]: SVGAnimatedNumber | null;
    [PropertySymbol.stdDeviationY]: SVGAnimatedNumber | null;
    [PropertySymbol.width]: SVGAnimatedLength | null;
    [PropertySymbol.x]: SVGAnimatedLength | null;
    [PropertySymbol.y]: SVGAnimatedLength | null;
    /**
     * Returns edge mode.
     *
     * @returns Edge mode.
     */
    get edgeMode(): SVGAnimatedEnumeration;
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
//# sourceMappingURL=SVGFEGaussianBlurElement.d.ts.map