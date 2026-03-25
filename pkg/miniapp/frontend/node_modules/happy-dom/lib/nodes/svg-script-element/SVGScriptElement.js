import * as PropertySymbol from '../../PropertySymbol.js';
import SVGAnimatedString from '../../svg/SVGAnimatedString.js';
import SVGGraphicsElement from '../svg-graphics-element/SVGGraphicsElement.js';
/**
 * SVG Script Element.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGScriptElement
 */
export default class SVGScriptElement extends SVGGraphicsElement {
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
    /**
     * Returns type.
     *
     * @returns Type.
     */
    get type() {
        return this.getAttribute('type') || '';
    }
    /**
     * Sets type.
     *
     * @param value New value.
     */
    set type(value) {
        this.setAttribute('type', value);
    }
}
//# sourceMappingURL=SVGScriptElement.js.map