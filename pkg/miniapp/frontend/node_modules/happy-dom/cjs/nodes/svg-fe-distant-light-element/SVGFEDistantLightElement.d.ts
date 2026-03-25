import SVGElement from '../svg-element/SVGElement.cjs';
import * as PropertySymbol from '../../PropertySymbol.cjs';
import SVGAnimatedNumber from '../../svg/SVGAnimatedNumber.cjs';
/**
 * SVGFEDistantLightElement.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGFEDistantLightElement
 */
export default class SVGFEDistantLightElement extends SVGElement {
    [PropertySymbol.azimuth]: SVGAnimatedNumber | null;
    [PropertySymbol.elevation]: SVGAnimatedNumber | null;
    /**
     * Returns azimuth.
     *
     * @returns Azimuth.
     */
    get azimuth(): SVGAnimatedNumber;
    /**
     * Returns elevation.
     *
     * @returns Elevation.
     */
    get elevation(): SVGAnimatedNumber;
}
//# sourceMappingURL=SVGFEDistantLightElement.d.ts.map