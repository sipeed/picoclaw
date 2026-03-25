import SVGElement from '../svg-element/SVGElement.cjs';
import * as PropertySymbol from '../../PropertySymbol.cjs';
import SVGAnimatedString from '../../svg/SVGAnimatedString.cjs';
/**
 * SVGFEMergeNodeElement.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGFEMergeNodeElement
 */
export default class SVGFEMergeNodeElement extends SVGElement {
    [PropertySymbol.in1]: SVGAnimatedString | null;
    /**
     * Returns in1.
     *
     * @returns In1.
     */
    get in1(): SVGAnimatedString;
}
//# sourceMappingURL=SVGFEMergeNodeElement.d.ts.map