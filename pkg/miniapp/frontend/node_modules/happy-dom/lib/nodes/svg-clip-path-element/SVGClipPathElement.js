import SVGElement from '../svg-element/SVGElement.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import SVGAnimatedEnumeration from '../../svg/SVGAnimatedEnumeration.js';
/**
 * SVG ClipPath Element.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGClipPathElement
 */
export default class SVGClipPathElement extends SVGElement {
    // Internal properties
    [PropertySymbol.clipPathUnits] = null;
    /**
     * Returns clipPathUnits.
     *
     * @returns ClipPathUnits.
     */
    get clipPathUnits() {
        if (!this[PropertySymbol.clipPathUnits]) {
            this[PropertySymbol.clipPathUnits] = new SVGAnimatedEnumeration(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('clipPathUnits'),
                setAttribute: (value) => this.setAttribute('clipPathUnits', value),
                values: ['userSpaceOnUse', 'objectBoundingBox'],
                defaultValue: 'userSpaceOnUse'
            });
        }
        return this[PropertySymbol.clipPathUnits];
    }
}
//# sourceMappingURL=SVGClipPathElement.js.map