import SVGElement from '../svg-element/SVGElement.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import SVGAnimatedLength from '../../svg/SVGAnimatedLength.js';
import SVGAnimatedString from '../../svg/SVGAnimatedString.js';
import SVGAnimatedNumber from '../../svg/SVGAnimatedNumber.js';
import SVGAnimatedEnumeration from '../../svg/SVGAnimatedEnumeration.js';
/**
 * SVGFEDisplacementMapElement.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGFEDisplacementMapElement
 */
export default class SVGFEDisplacementMapElement extends SVGElement {
    // Static properties
    static SVG_CHANNEL_UNKNOWN = 0;
    static SVG_CHANNEL_R = 1;
    static SVG_CHANNEL_G = 2;
    static SVG_CHANNEL_B = 3;
    static SVG_CHANNEL_A = 4;
    // Internal properties
    [PropertySymbol.height] = null;
    [PropertySymbol.in1] = null;
    [PropertySymbol.in2] = null;
    [PropertySymbol.result] = null;
    [PropertySymbol.scale] = null;
    [PropertySymbol.width] = null;
    [PropertySymbol.x] = null;
    [PropertySymbol.xChannelSelector] = null;
    [PropertySymbol.y] = null;
    [PropertySymbol.yChannelSelector] = null;
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
     * Returns in2.
     *
     * @returns In2.
     */
    get in2() {
        if (!this[PropertySymbol.in2]) {
            this[PropertySymbol.in2] = new SVGAnimatedString(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('in2'),
                setAttribute: (value) => this.setAttribute('in2', value)
            });
        }
        return this[PropertySymbol.in2];
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
     * Returns scale.
     *
     * @returns Scale.
     */
    get scale() {
        if (!this[PropertySymbol.scale]) {
            this[PropertySymbol.scale] = new SVGAnimatedNumber(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('scale'),
                setAttribute: (value) => this.setAttribute('scale', value)
            });
        }
        return this[PropertySymbol.scale];
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
     * Returns x channel selector.
     *
     * @returns X channel selector.
     */
    get xChannelSelector() {
        if (!this[PropertySymbol.xChannelSelector]) {
            this[PropertySymbol.xChannelSelector] = new SVGAnimatedEnumeration(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('xChannelSelector'),
                setAttribute: (value) => this.setAttribute('xChannelSelector', value),
                values: ['r', 'g', 'b', 'a'],
                defaultValue: 'r'
            });
        }
        return this[PropertySymbol.xChannelSelector];
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
     * Returns y channel selector.
     *
     * @returns Y channel selector.
     */
    get yChannelSelector() {
        if (!this[PropertySymbol.yChannelSelector]) {
            this[PropertySymbol.yChannelSelector] = new SVGAnimatedEnumeration(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('yChannelSelector'),
                setAttribute: (value) => this.setAttribute('yChannelSelector', value),
                values: ['r', 'g', 'b', 'a'],
                defaultValue: 'r'
            });
        }
        return this[PropertySymbol.yChannelSelector];
    }
}
//# sourceMappingURL=SVGFEDisplacementMapElement.js.map