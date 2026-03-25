import SVGElement from '../svg-element/SVGElement.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import SVGAnimatedLength from '../../svg/SVGAnimatedLength.js';
import SVGAnimatedEnumeration from '../../svg/SVGAnimatedEnumeration.js';
import SVGAnimatedString from '../../svg/SVGAnimatedString.js';
import SVGAnimatedNumberList from '../../svg/SVGAnimatedNumberList.js';
import SVGAnimatedNumber from '../../svg/SVGAnimatedNumber.js';
import SVGAnimatedBoolean from '../../svg/SVGAnimatedBoolean.js';
import SVGAnimatedInteger from '../../svg/SVGAnimatedInteger.js';
/**
 * SVGFEConvolveMatrixElement.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGFEConvolveMatrixElement
 */
export default class SVGFEConvolveMatrixElement extends SVGElement {
    static SVG_EDGEMODE_UNKNOWN: number;
    static SVG_EDGEMODE_DUPLICATE: number;
    static SVG_EDGEMODE_WRAP: number;
    static SVG_EDGEMODE_NONE: number;
    [PropertySymbol.bias]: SVGAnimatedNumber | null;
    [PropertySymbol.divisor]: SVGAnimatedNumber | null;
    [PropertySymbol.edgeMode]: SVGAnimatedEnumeration | null;
    [PropertySymbol.height]: SVGAnimatedLength | null;
    [PropertySymbol.in1]: SVGAnimatedString | null;
    [PropertySymbol.kernelMatrix]: SVGAnimatedNumberList | null;
    [PropertySymbol.kernelUnitLengthX]: SVGAnimatedNumber | null;
    [PropertySymbol.kernelUnitLengthY]: SVGAnimatedNumber | null;
    [PropertySymbol.orderX]: SVGAnimatedInteger | null;
    [PropertySymbol.orderY]: SVGAnimatedInteger | null;
    [PropertySymbol.preserveAlpha]: SVGAnimatedBoolean | null;
    [PropertySymbol.result]: SVGAnimatedString | null;
    [PropertySymbol.targetX]: SVGAnimatedInteger | null;
    [PropertySymbol.targetY]: SVGAnimatedInteger | null;
    [PropertySymbol.width]: SVGAnimatedLength | null;
    [PropertySymbol.x]: SVGAnimatedLength | null;
    [PropertySymbol.y]: SVGAnimatedLength | null;
    /**
     * Returns bias.
     *
     * @returns Bias.
     */
    get bias(): SVGAnimatedNumber;
    /**
     * Returns divisor.
     *
     * @returns Divisor.
     */
    get divisor(): SVGAnimatedNumber;
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
     * Returns kernel matrix.
     *
     * @returns Kernel matrix.
     */
    get kernelMatrix(): SVGAnimatedNumberList;
    /**
     * Returns kernel unit length x.
     *
     * @returns Kernel unit length x.
     */
    get kernelUnitLengthX(): SVGAnimatedNumber;
    /**
     * Returns kernel unit length y.
     *
     * @returns Kernel unit length y.
     */
    get kernelUnitLengthY(): SVGAnimatedNumber;
    /**
     * Returns order x.
     *
     * @returns Order x.
     */
    get orderX(): SVGAnimatedInteger;
    /**
     * Returns order y.
     *
     * @returns Order y.
     */
    get orderY(): SVGAnimatedInteger;
    /**
     * Returns preserve alpha.
     *
     * @returns Preserve alpha.
     */
    get preserveAlpha(): SVGAnimatedBoolean;
    /**
     * Returns result.
     *
     * @returns Result.
     */
    get result(): SVGAnimatedString;
    /**
     * Returns target x.
     *
     * @returns Target x.
     */
    get targetX(): SVGAnimatedInteger;
    /**
     * Returns target y.
     *
     * @returns Target y.
     */
    get targetY(): SVGAnimatedInteger;
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
}
//# sourceMappingURL=SVGFEConvolveMatrixElement.d.ts.map