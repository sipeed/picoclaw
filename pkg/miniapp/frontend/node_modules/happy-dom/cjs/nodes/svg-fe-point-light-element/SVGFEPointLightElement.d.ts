import SVGElement from '../svg-element/SVGElement.cjs';
import * as PropertySymbol from '../../PropertySymbol.cjs';
import SVGAnimatedNumber from '../../svg/SVGAnimatedNumber.cjs';
/**
 * SVGFEPointLightElement.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGFEPointLightElement
 */
export default class SVGFEPointLightElement extends SVGElement {
    [PropertySymbol.x]: SVGAnimatedNumber | null;
    [PropertySymbol.y]: SVGAnimatedNumber | null;
    [PropertySymbol.z]: SVGAnimatedNumber | null;
    /**
     * Returns x.
     *
     * @returns X.
     */
    get x(): SVGAnimatedNumber;
    /**
     * Returns y.
     *
     * @returns Y.
     */
    get y(): SVGAnimatedNumber;
    /**
     * Returns z.
     *
     * @returns Z.
     */
    get z(): SVGAnimatedNumber;
}
//# sourceMappingURL=SVGFEPointLightElement.d.ts.map