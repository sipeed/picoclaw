import SVGElement from '../svg-element/SVGElement.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import SVGAnimatedNumber from '../../svg/SVGAnimatedNumber.js';
import SVGAnimatedString from '../../svg/SVGAnimatedString.js';
import SVGAnimatedLength from '../../svg/SVGAnimatedLength.js';
import SVGAnimatedEnumeration from '../../svg/SVGAnimatedEnumeration.js';
/**
 * SVGFEGaussianBlurElement.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGFEGaussianBlurElement
 */
export default class SVGFEGaussianBlurElement extends SVGElement {
    // Static properties
    static SVG_EDGEMODE_UNKNOWN = 0;
    static SVG_EDGEMODE_DUPLICATE = 1;
    static SVG_EDGEMODE_WRAP = 2;
    static SVG_EDGEMODE_NONE = 3;
    // Internal properties
    [PropertySymbol.edgeMode] = null;
    [PropertySymbol.height] = null;
    [PropertySymbol.in1] = null;
    [PropertySymbol.result] = null;
    [PropertySymbol.stdDeviationX] = null;
    [PropertySymbol.stdDeviationY] = null;
    [PropertySymbol.width] = null;
    [PropertySymbol.x] = null;
    [PropertySymbol.y] = null;
    /**
     * Returns edge mode.
     *
     * @returns Edge mode.
     */
    get edgeMode() {
        if (!this[PropertySymbol.edgeMode]) {
            this[PropertySymbol.edgeMode] = new SVGAnimatedEnumeration(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('edgeMode'),
                setAttribute: (value) => this.setAttribute('edgeMode', value),
                values: ['duplicate', 'wrap', 'none'],
                defaultValue: 'duplicate'
            });
        }
        return this[PropertySymbol.edgeMode];
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
     * Returns in1.
     *
     * @returns In1.
     */
    get in1() {
        if (!this[PropertySymbol.in1]) {
            this[PropertySymbol.in1] = new SVGAnimatedString(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('in'),
                setAttribute: (value) => this.setAttribute('in', value)
            });
        }
        return this[PropertySymbol.in1];
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
     * Returns stdDeviationX.
     *
     * @returns StdDeviationX.
     */
    get stdDeviationX() {
        if (!this[PropertySymbol.stdDeviationX]) {
            this[PropertySymbol.stdDeviationX] = new SVGAnimatedNumber(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('stdDeviationX') || '2',
                setAttribute: (value) => this.setAttribute('stdDeviationX', value)
            });
        }
        return this[PropertySymbol.stdDeviationX];
    }
    /**
     * Returns stdDeviationY.
     *
     * @returns StdDeviationY.
     */
    get stdDeviationY() {
        if (!this[PropertySymbol.stdDeviationY]) {
            this[PropertySymbol.stdDeviationY] = new SVGAnimatedNumber(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('stdDeviationY') || '2',
                setAttribute: (value) => this.setAttribute('stdDeviationY', value)
            });
        }
        return this[PropertySymbol.stdDeviationY];
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
     * Sets stdDeviation.
     *
     * @param x X.
     * @param y Y.
     */
    setStdDeviation(x, y) {
        this.stdDeviationX.baseVal = x;
        this.stdDeviationY.baseVal = y;
    }
}
//# sourceMappingURL=SVGFEGaussianBlurElement.js.map