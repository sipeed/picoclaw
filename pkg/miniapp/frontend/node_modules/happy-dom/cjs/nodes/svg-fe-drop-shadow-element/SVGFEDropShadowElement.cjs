"use strict";
var __createBinding = (this && this.__createBinding) || (Object.create ? (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    var desc = Object.getOwnPropertyDescriptor(m, k);
    if (!desc || ("get" in desc ? !m.__esModule : desc.writable || desc.configurable)) {
      desc = { enumerable: true, get: function() { return m[k]; } };
    }
    Object.defineProperty(o, k2, desc);
}) : (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    o[k2] = m[k];
}));
var __setModuleDefault = (this && this.__setModuleDefault) || (Object.create ? (function(o, v) {
    Object.defineProperty(o, "default", { enumerable: true, value: v });
}) : function(o, v) {
    o["default"] = v;
});
var __importStar = (this && this.__importStar) || function (mod) {
    if (mod && mod.__esModule) return mod;
    var result = {};
    if (mod != null) for (var k in mod) if (k !== "default" && Object.prototype.hasOwnProperty.call(mod, k)) __createBinding(result, mod, k);
    __setModuleDefault(result, mod);
    return result;
};
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const SVGElement_js_1 = __importDefault(require("../svg-element/SVGElement.cjs"));
const PropertySymbol = __importStar(require("../../PropertySymbol.cjs"));
const SVGAnimatedNumber_js_1 = __importDefault(require("../../svg/SVGAnimatedNumber.cjs"));
const SVGAnimatedString_js_1 = __importDefault(require("../../svg/SVGAnimatedString.cjs"));
const SVGAnimatedLength_js_1 = __importDefault(require("../../svg/SVGAnimatedLength.cjs"));
/**
 * SVGFEDropShadowElement.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGFEDropShadowElement
 */
class SVGFEDropShadowElement extends SVGElement_js_1.default {
    // Internal properties
    [PropertySymbol.dx] = null;
    [PropertySymbol.dy] = null;
    [PropertySymbol.height] = null;
    [PropertySymbol.in1] = null;
    [PropertySymbol.result] = null;
    [PropertySymbol.stdDeviationX] = null;
    [PropertySymbol.stdDeviationY] = null;
    [PropertySymbol.width] = null;
    [PropertySymbol.x] = null;
    [PropertySymbol.y] = null;
    /**
     * Returns dx.
     *
     * @returns Dx.
     */
    get dx() {
        if (!this[PropertySymbol.dx]) {
            this[PropertySymbol.dx] = new SVGAnimatedNumber_js_1.default(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('dx'),
                setAttribute: (value) => this.setAttribute('dx', value)
            });
        }
        return this[PropertySymbol.dx];
    }
    /**
     * Returns dy.
     *
     * @returns Dy.
     */
    get dy() {
        if (!this[PropertySymbol.dy]) {
            this[PropertySymbol.dy] = new SVGAnimatedNumber_js_1.default(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('dy'),
                setAttribute: (value) => this.setAttribute('dy', value)
            });
        }
        return this[PropertySymbol.dy];
    }
    /**
     * Returns height.
     *
     * @returns Height.
     */
    get height() {
        if (!this[PropertySymbol.height]) {
            this[PropertySymbol.height] = new SVGAnimatedLength_js_1.default(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
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
            this[PropertySymbol.in1] = new SVGAnimatedString_js_1.default(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
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
            this[PropertySymbol.result] = new SVGAnimatedString_js_1.default(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
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
            this[PropertySymbol.stdDeviationX] = new SVGAnimatedNumber_js_1.default(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('stdDeviationX'),
                setAttribute: (value) => this.setAttribute('stdDeviationX', value),
                defaultValue: 2
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
            this[PropertySymbol.stdDeviationY] = new SVGAnimatedNumber_js_1.default(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('stdDeviationY'),
                setAttribute: (value) => this.setAttribute('stdDeviationY', value),
                defaultValue: 2
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
            this[PropertySymbol.width] = new SVGAnimatedLength_js_1.default(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
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
            this[PropertySymbol.x] = new SVGAnimatedLength_js_1.default(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
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
            this[PropertySymbol.y] = new SVGAnimatedLength_js_1.default(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
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
exports.default = SVGFEDropShadowElement;
//# sourceMappingURL=SVGFEDropShadowElement.cjs.map