import * as PropertySymbol from '../../PropertySymbol.cjs';
import SVGAnimatedNumber from '../../svg/SVGAnimatedNumber.cjs';
import SVGElement from '../svg-element/SVGElement.cjs';
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