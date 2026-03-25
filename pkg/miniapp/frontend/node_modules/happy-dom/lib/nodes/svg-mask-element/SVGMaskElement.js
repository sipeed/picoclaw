import SVGElement from '../svg-element/SVGElement.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import SVGAnimatedEnumeration from '../../svg/SVGAnimatedEnumeration.js';
import SVGAnimatedLength from '../../svg/SVGAnimatedLength.js';
/**
 * SVG Mask Element.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGMaskElement
 */
export default class SVGMaskElement extends SVGElement {
    // Internal properties
    [PropertySymbol.maskUnits] = null;
    [PropertySymbol.maskContentUnits] = null;
    [PropertySymbol.x] = null;
    [PropertySymbol.y] = null;
    [PropertySymbol.width] = null;
    [PropertySymbol.height] = null;
    /**
     * Returns mask units.
     *
     * @returns Mask units.
     */
    get maskUnits() {
        if (!this[PropertySymbol.maskUnits]) {
            this[PropertySymbol.maskUnits] = new SVGAnimatedEnumeration(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('maskUnits'),
                setAttribute: (value) => this.setAttribute('maskUnits', value),
                values: ['userSpaceOnUse', 'objectBoundingBox'],
                defaultValue: 'userSpaceOnUse'
            });
        }
        return this[PropertySymbol.maskUnits];
    }
    /**
     * Returns mask content units.
     *
     * @returns Mask content units.
     */
    get maskContentUnits() {
        if (!this[PropertySymbol.maskContentUnits]) {
            this[PropertySymbol.maskContentUnits] = new SVGAnimatedEnumeration(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('maskContentUnits'),
                setAttribute: (value) => this.setAttribute('maskContentUnits', value),
                values: ['userSpaceOnUse', 'objectBoundingBox'],
                defaultValue: 'userSpaceOnUse'
            });
        }
        return this[PropertySymbol.maskContentUnits];
    }
    /**
     * Returns width.
     *
     * @returns Width.
     */
    get width() {
        if (!this[PropertySymbol.width]) {
            this[PropertySymbol.width] = new SVGAnimatedLength(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('width'),
                setAttribute: (value) => this.setAttribute('width', value)
            });
        }
        return this[PropertySymbol.width];
    }
    /**
     * Returns height.
     *
     * @returns Height.
     */
    get height() {
        if (!this[PropertySymbol.height]) {
            this[PropertySymbol.height] = new SVGAnimatedLength(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('height'),
                setAttribute: (value) => this.setAttribute('height', value)
            });
        }
        return this[PropertySymbol.height];
    }
    /**
     * Returns x position.
     *
     * @returns X position.
     */
    get x() {
        if (!this[PropertySymbol.x]) {
            this[PropertySymbol.x] = new SVGAnimatedLength(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('x'),
                setAttribute: (value) => this.setAttribute('x', value)
            });
        }
        return this[PropertySymbol.x];
    }
    /**
     * Returns y position.
     *
     * @returns Y position.
     */
    get y() {
        if (!this[PropertySymbol.y]) {
            this[PropertySymbol.y] = new SVGAnimatedLength(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('y'),
                setAttribute: (value) => this.setAttribute('y', value)
            });
        }
        return this[PropertySymbol.y];
    }
}
//# sourceMappingURL=SVGMaskElement.js.map