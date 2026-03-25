import SVGElement from '../svg-element/SVGElement.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import SVGAnimatedNumber from '../../svg/SVGAnimatedNumber.js';
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