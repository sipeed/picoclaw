import SVGElement from '../svg-element/SVGElement.cjs';
import * as PropertySymbol from '../../PropertySymbol.cjs';
import SVGAnimatedEnumeration from '../../svg/SVGAnimatedEnumeration.cjs';
/**
 * SVG ClipPath Element.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGClipPathElement
 */
export default class SVGClipPathElement extends SVGElement {
    [PropertySymbol.clipPathUnits]: SVGAnimatedEnumeration | null;
    /**
     * Returns clipPathUnits.
     *
     * @returns ClipPathUnits.
     */
    get clipPathUnits(): SVGAnimatedEnumeration;
}
//# sourceMappingURL=SVGClipPathElement.d.ts.map