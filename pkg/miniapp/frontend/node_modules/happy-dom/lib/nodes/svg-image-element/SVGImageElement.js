import * as PropertySymbol from '../../PropertySymbol.js';
import SVGAnimatedLength from '../../svg/SVGAnimatedLength.js';
import SVGAnimatedPreserveAspectRatio from '../../svg/SVGAnimatedPreserveAspectRatio.js';
import SVGAnimatedString from '../../svg/SVGAnimatedString.js';
import SVGGraphicsElement from '../svg-graphics-element/SVGGraphicsElement.js';
/**
 * SVG Image Element.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGImageElement
 */
export default class SVGImageElement extends SVGGraphicsElement {
    // Internal properties
    [PropertySymbol.href] = null;
    [PropertySymbol.preserveAspectRatio] = null;
    [PropertySymbol.x] = null;
    [PropertySymbol.y] = null;
    [PropertySymbol.width] = null;
    [PropertySymbol.height] = null;
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
     * Returns decoding.
     *
     * @returns Decoding.
     */
    get decoding() {
        const value = this.getAttribute('decoding');
        switch (value) {
            case 'sync':
            case 'async':
            case 'auto':
                return value;
            default:
                return 'auto';
        }
    }
    /**
     * Sets decoding.
     *
     * @param value Decoding.
     */
    set decoding(value) {
        this.setAttribute('decoding', value);
    }
    /**
     * Returns preserve aspect ratio.
     *
     * @returns Preserve aspect ratio.
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
     * Decodes the image.
     *
     * @returns Promise.
     */
    decode() {
        // TODO: Implement decode()
        return Promise.resolve();
    }
}
//# sourceMappingURL=SVGImageElement.js.map