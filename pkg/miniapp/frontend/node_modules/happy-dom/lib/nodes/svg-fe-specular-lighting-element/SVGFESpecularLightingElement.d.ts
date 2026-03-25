import SVGElement from '../svg-element/SVGElement.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import SVGAnimatedNumber from '../../svg/SVGAnimatedNumber.js';
import SVGAnimatedLength from '../../svg/SVGAnimatedLength.js';
import SVGAnimatedString from '../../svg/SVGAnimatedString.js';
/**
 * SVGFESpecularLightingElement.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGFESpecularLightingElement
 */
export default class SVGFESpecularLightingElement extends SVGElement {
    [PropertySymbol.height]: SVGAnimatedLength | null;
    [PropertySymbol.in1]: SVGAnimatedString | null;
    [PropertySymbol.kernelUnitLengthX]: SVGAnimatedNumber | null;
    [PropertySymbol.kernelUnitLengthY]: SVGAnimatedNumber | null;
    [PropertySymbol.result]: SVGAnimatedString | null;
    [PropertySymbol.specularConstant]: SVGAnimatedNumber | null;
    [PropertySymbol.specularExponent]: SVGAnimatedNumber | null;
    [PropertySymbol.surfaceScale]: SVGAnimatedNumber | null;
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
     * Returns kernelUnitLengthX.
     *
     * @returns KernelUnitLengthX.
     */
    get kernelUnitLengthX(): SVGAnimatedNumber;
    /**
     * Returns kernelUnitLengthY.
     *
     * @returns KernelUnitLengthY.
     */
    get kernelUnitLengthY(): SVGAnimatedNumber;
    /**
     * Returns result.
     *
     * @returns Result.
     */
    get result(): SVGAnimatedString;
    /**
     * Returns specularConstant.
     *
     * @returns SpecularConstant.
     */
    get specularConstant(): SVGAnimatedNumber;
    /**
     * Returns specularExponent.
     *
     * @returns SpecularExponent.
     */
    get specularExponent(): SVGAnimatedNumber;
    /**
     * Returns surfaceScale.
     *
     * @returns SurfaceScale.
     */
    get surfaceScale(): SVGAnimatedNumber;
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
//# sourceMappingURL=SVGFESpecularLightingElement.d.ts.map