import SVGElement from '../svg-element/SVGElement.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import SVGAnimatedLength from '../../svg/SVGAnimatedLength.js';
import SVGAnimatedString from '../../svg/SVGAnimatedString.js';
import SVGAnimatedNumber from '../../svg/SVGAnimatedNumber.js';
/**
 * SVGFEDiffuseLightingElement.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGFEDiffuseLightingElement
 */
export default class SVGFEDiffuseLightingElement extends SVGElement {
    [PropertySymbol.diffuseConstant]: SVGAnimatedNumber | null;
    [PropertySymbol.height]: SVGAnimatedLength | null;
    [PropertySymbol.in1]: SVGAnimatedString | null;
    [PropertySymbol.kernelUnitLengthX]: SVGAnimatedNumber | null;
    [PropertySymbol.kernelUnitLengthY]: SVGAnimatedNumber | null;
    [PropertySymbol.result]: SVGAnimatedString | null;
    [PropertySymbol.surfaceScale]: SVGAnimatedNumber | null;
    [PropertySymbol.width]: SVGAnimatedLength | null;
    [PropertySymbol.x]: SVGAnimatedLength | null;
    [PropertySymbol.y]: SVGAnimatedLength | null;
    /**
     * Returns diffuse constant.
     *
     * @returns Diffuse constant.
     */
    get diffuseConstant(): SVGAnimatedNumber;
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
     * Returns result.
     *
     * @returns Result.
     */
    get result(): SVGAnimatedString;
    /**
     * Returns surface scale.
     *
     * @returns Surface scale.
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
//# sourceMappingURL=SVGFEDiffuseLightingElement.d.ts.map