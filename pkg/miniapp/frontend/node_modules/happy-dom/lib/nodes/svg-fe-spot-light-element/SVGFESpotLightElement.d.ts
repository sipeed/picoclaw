import SVGElement from '../svg-element/SVGElement.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import SVGAnimatedNumber from '../../svg/SVGAnimatedNumber.js';
/**
 * SVGFESpotLightElement.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGFESpotLightElement
 */
export default class SVGFESpotLightElement extends SVGElement {
    [PropertySymbol.x]: SVGAnimatedNumber | null;
    [PropertySymbol.y]: SVGAnimatedNumber | null;
    [PropertySymbol.z]: SVGAnimatedNumber | null;
    [PropertySymbol.pointsAtX]: SVGAnimatedNumber | null;
    [PropertySymbol.pointsAtY]: SVGAnimatedNumber | null;
    [PropertySymbol.pointsAtZ]: SVGAnimatedNumber | null;
    [PropertySymbol.specularExponent]: SVGAnimatedNumber | null;
    [PropertySymbol.limitingConeAngle]: SVGAnimatedNumber | null;
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
    /**
     * Returns pointsAtX.
     *
     * @returns PointsAtX.
     */
    get pointsAtX(): SVGAnimatedNumber;
    /**
     * Returns pointsAtY.
     *
     * @returns PointsAtY.
     */
    get pointsAtY(): SVGAnimatedNumber;
    /**
     * Returns pointsAtZ.
     *
     * @returns PointsAtZ.
     */
    get pointsAtZ(): SVGAnimatedNumber;
    /**
     * Returns specularExponent.
     *
     * @returns SpecularExponent.
     */
    get specularExponent(): SVGAnimatedNumber;
    /**
     * Returns limitingConeAngle.
     *
     * @returns LimitingConeAngle.
     */
    get limitingConeAngle(): SVGAnimatedNumber;
}
//# sourceMappingURL=SVGFESpotLightElement.d.ts.map