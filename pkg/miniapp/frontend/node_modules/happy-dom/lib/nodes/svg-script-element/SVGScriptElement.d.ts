import * as PropertySymbol from '../../PropertySymbol.js';
import SVGAnimatedString from '../../svg/SVGAnimatedString.js';
import SVGGraphicsElement from '../svg-graphics-element/SVGGraphicsElement.js';
/**
 * SVG Script Element.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGScriptElement
 */
export default class SVGScriptElement extends SVGGraphicsElement {
    [PropertySymbol.href]: SVGAnimatedString | null;
    /**
     * Returns href.
     *
     * @returns Href.
     */
    get href(): SVGAnimatedString;
    /**
     * Returns type.
     *
     * @returns Type.
     */
    get type(): string;
    /**
     * Sets type.
     *
     * @param value New value.
     */
    set type(value: string);
}
//# sourceMappingURL=SVGScriptElement.d.ts.map