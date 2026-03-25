import SVGElement from '../svg-element/SVGElement.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import SVGAnimatedString from '../../svg/SVGAnimatedString.js';
import SVGAnimatedLength from '../../svg/SVGAnimatedLength.js';
import SVGAnimatedPreserveAspectRatio from '../../svg/SVGAnimatedPreserveAspectRatio.js';
/**
 * SVGFEImageElement.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGFEImageElement
 */
export default class SVGFEImageElement extends SVGElement {
    // Internal properties
    [PropertySymbol.crossOrigin] = null;
    [PropertySymbol.height] = null;
    [PropertySymbol.href] = null;
    [PropertySymbol.preserveAspectRatio] = null;
    [PropertySymbol.result] = null;
    [PropertySymbol.width] = null;
    [PropertySymbol.x] = null;
    [PropertySymbol.y] = null;
    /**
     * Returns "crossorigin" attribute.
     *
     * @returns Cross origin.
     */
    get crossOrigin() {
        return this.getAttribute('crossorigin');
    }
    /**
     * Sets "crossorigin" attribute.
     *
     * @param value Cross origin.
     */
    set crossOrigin(value) {
        this.setAttribute('crossorigin', value);
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
     * Returns preserveAspectRatio.
     *
     * @returns PreserveAspectRatio.
     */
    get preserveAspectRatio() {
        if (!this[PropertySymbol.preserveAspectRatio]) {
            this[PropertySymbol.preserveAspectRatio] = new SVGAnimatedPreserveAspectRatio(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('preserveAspectRatio'),
                setAttribute: (value) => this.setAttribute('preserveAspectRatio', value)
            });
        }
        return this[PropertySymbol.preserveAspectRatio];
    }
    /**
     * Returns result.
     *
     * @returns Result.
     */
    get result() {
        if (!this[PropertySymbol.result]) {
            this[PropertySymbol.result] = new SVGAnimatedString(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('result'),
                setAttribute: (value) => this.setAttribute('result', value)
            });
        }
        return this[PropertySymbol.result];
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
     * Returns x.
     *
     * @returns X.
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
     * Returns y.
     *
     * @returns Y.
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
//# sourceMappingURL=SVGFEImageElement.js.map