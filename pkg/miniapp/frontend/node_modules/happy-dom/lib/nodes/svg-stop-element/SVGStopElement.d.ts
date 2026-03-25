import * as PropertySymbol from '../../PropertySymbol.js';
import SVGAnimatedNumber from '../../svg/SVGAnimatedNumber.js';
import SVGElement from '../svg-element/SVGElement.js';
/**
 * SVG Stop Element.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGStopElement
 */
export default class SVGStopElement extends SVGElement {
    [PropertySymbol.offset]: SVGAnimatedNumber | null;
    /**
     * Returns offset.
     *
     * @returns Offset.
     */
    get offset(): SVGAnimatedNumber;
}
//# sourceMappingURL=SVGStopElement.d.ts.map