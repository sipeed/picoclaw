import SVGElement from '../svg-element/SVGElement.cjs';
import * as PropertySymbol from '../../PropertySymbol.cjs';
import SVGAnimatedNumber from '../../svg/SVGAnimatedNumber.cjs';
import SVGAnimatedEnumeration from '../../svg/SVGAnimatedEnumeration.cjs';
import SVGAnimatedNumberList from '../../svg/SVGAnimatedNumberList.cjs';
/**
 * SVGComponentTransferFunctionElement.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGComponentTransferFunctionElement
 */
export default class SVGComponentTransferFunctionElement extends SVGElement {
    static SVG_FECOMPONENTTRANSFER_TYPE_UNKNOWN: number;
    static SVG_FECOMPONENTTRANSFER_TYPE_IDENTITY: number;
    static SVG_FECOMPONENTTRANSFER_TYPE_TABLE: number;
    static SVG_FECOMPONENTTRANSFER_TYPE_DISCRETE: number;
    static SVG_FECOMPONENTTRANSFER_TYPE_LINEAR: number;
    static SVG_FECOMPONENTTRANSFER_TYPE_GAMMA: number;
    [PropertySymbol.type]: SVGAnimatedEnumeration | null;
    [PropertySymbol.tableValues]: SVGAnimatedNumberList | null;
    [PropertySymbol.slope]: SVGAnimatedNumber | null;
    [PropertySymbol.intercept]: SVGAnimatedNumber | null;
    [PropertySymbol.amplitude]: SVGAnimatedNumber | null;
    [PropertySymbol.exponent]: SVGAnimatedNumber | null;
    [PropertySymbol.offset]: SVGAnimatedNumber | null;
    /**
     * Returns type.
     *
     * @returns Type.
     */
    get type(): SVGAnimatedEnumeration;
    /**
     * Returns table values.
     *
     * @returns Table values.
     */
    get tableValues(): SVGAnimatedNumberList;
    /**
     * Returns slope.
     *
     * @returns Slope.
     */
    get slope(): SVGAnimatedNumber;
    /**
     * Returns intercept.
     *
     * @returns Intercept.
     */
    get intercept(): SVGAnimatedNumber;
    /**
     * Returns amplitude.
     *
     * @returns Amplitude.
     */
    get amplitude(): SVGAnimatedNumber;
    /**
     * Returns exponent.
     *
     * @returns Exponent.
     */
    get exponent(): SVGAnimatedNumber;
    /**
     * Returns offset.
     *
     * @returns Offset.
     */
    get offset(): SVGAnimatedNumber;
}
//# sourceMappingURL=SVGComponentTransferFunctionElement.d.ts.map