import SVGAnimatedString from '../../svg/SVGAnimatedString.js';
import SVGElement from '../svg-element/SVGElement.js';
import * as PropertySymbol from '../../PropertySymbol.js';
/**
 * SVG MPath Element.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGMPathElement
 */
export default class SVGMPathElement extends SVGElement {
    // Internal properties
    [PropertySymbol.href] = null;
    /**
     * Returns href.
     *
     * @returns Href.
     */
    get href() {
        if (!this[PropertySymbol.href]) {
            this[PropertySymbol.href] = new SVGAnimatedString(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('href'),
                setAttribute: (value) => this.setAttribute('href', value)
            });
        }
        return this[PropertySymbol.href];
    }
}
//# sourceMappingURL=SVGMPathElement.js.map