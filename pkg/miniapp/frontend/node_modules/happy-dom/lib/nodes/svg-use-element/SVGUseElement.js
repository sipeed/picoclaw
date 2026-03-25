import * as PropertySymbol from '../../PropertySymbol.js';
import SVGAnimatedLength from '../../svg/SVGAnimatedLength.js';
import SVGAnimatedString from '../../svg/SVGAnimatedString.js';
import SVGGraphicsElement from '../svg-graphics-element/SVGGraphicsElement.js';
/**
 * SVG Use Element.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGUseElement
 */
export default class SVGUseElement extends SVGGraphicsElement {
    // Internal properties
    [PropertySymbol.href] = null;
    [PropertySymbol.x] = null;
    [PropertySymbol.y] = null;
    [PropertySymbol.width] = null;
    [PropertySymbol.height] = null;
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
}
//# sourceMappingURL=SVGUseElement.js.map