import SVGElement from '../svg-element/SVGElement.cjs';
import * as PropertySymbol from '../../PropertySymbol.cjs';
import SVGAnimatedString from '../../svg/SVGAnimatedString.cjs';
import SVGAnimatedLength from '../../svg/SVGAnimatedLength.cjs';
import SVGAnimatedNumber from '../../svg/SVGAnimatedNumber.cjs';
import SVGAnimatedEnumeration from '../../svg/SVGAnimatedEnumeration.cjs';
/**
 * SVGFEMorphologyElement.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGFEMorphologyElement
 */
export default class SVGFEMorphologyElement extends SVGElement {
    static readonly SVG_MORPHOLOGY_OPERATOR_UNKNOWN = 0;
    static readonly SVG_MORPHOLOGY_OPERATOR_ERODE = 1;
    static readonly SVG_MORPHOLOGY_OPERATOR_DILATE = 2;
    [PropertySymbol.height]: SVGAnimatedLength | null;
    [PropertySymbol.in1]: SVGAnimatedString | null;
    [PropertySymbol.operator]: SVGAnimatedEnumeration | null;
    [PropertySymbol.radiusX]: SVGAnimatedNumber | null;
    [PropertySymbol.radiusY]: SVGAnimatedNumber | null;
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
     * Returns operator.
     *
     * @returns Operator.
     */
    get operator(): SVGAnimatedEnumeration;
    /**
     * Returns radiusX.
     *
     * @returns RadiusX.
     */
    get radiusX(): SVGAnimatedNumber;
    /**
     * Returns radiusY.
     *
     * @returns RadiusY.
     */
    get radiusY(): SVGAnimatedNumber;
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
//# sourceMappingURL=SVGFEMorphologyElement.d.ts.map