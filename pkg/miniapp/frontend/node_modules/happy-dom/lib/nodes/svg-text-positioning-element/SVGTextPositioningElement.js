import * as PropertySymbol from '../../PropertySymbol.js';
import SVGTextContentElement from '../svg-text-content-element/SVGTextContentElement.js';
import SVGAnimatedLengthList from '../../svg/SVGAnimatedLengthList.js';
import SVGAnimatedNumberList from '../../svg/SVGAnimatedNumberList.js';
/**
 * SVG Text Positioning Element.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGTextPositioningElement
 */
export default class SVGTextPositioningElement extends SVGTextContentElement {
    // Internal properties
    [PropertySymbol.x] = null;
    [PropertySymbol.y] = null;
    [PropertySymbol.dx] = null;
    [PropertySymbol.dy] = null;
    [PropertySymbol.rotate] = null;
    /**
     * Returns x.
     *
     * @returns X.
     */
    get x() {
        if (!this[PropertySymbol.x]) {
            this[PropertySymbol.x] = new SVGAnimatedLengthList(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
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
            this[PropertySymbol.y] = new SVGAnimatedLengthList(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('y'),
                setAttribute: (value) => this.setAttribute('y', value)
            });
        }
        return this[PropertySymbol.y];
    }
    /**
     * Returns dx.
     *
     * @returns DX.
     */
    get dx() {
        if (!this[PropertySymbol.dx]) {
            this[PropertySymbol.dx] = new SVGAnimatedLengthList(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('dx'),
                setAttribute: (value) => this.setAttribute('dx', value)
            });
        }
        return this[PropertySymbol.dx];
    }
    /**
     * Returns dy.
     *
     * @returns DY.
     */
    get dy() {
        if (!this[PropertySymbol.dy]) {
            this[PropertySymbol.dy] = new SVGAnimatedLengthList(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('dy'),
                setAttribute: (value) => this.setAttribute('dy', value)
            });
        }
        return this[PropertySymbol.dy];
    }
    /**
     * Returns rotate.
     *
     * @returns Rotate.
     */
    get rotate() {
        if (!this[PropertySymbol.rotate]) {
            this[PropertySymbol.rotate] = new SVGAnimatedNumberList(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('rotate'),
                setAttribute: (value) => this.setAttribute('rotate', value)
            });
        }
        return this[PropertySymbol.rotate];
    }
}
//# sourceMappingURL=SVGTextPositioningElement.js.map