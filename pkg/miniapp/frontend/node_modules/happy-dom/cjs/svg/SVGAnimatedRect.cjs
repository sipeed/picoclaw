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
const PropertySymbol = __importStar(require("../PropertySymbol.cjs"));
const SVGRect_js_1 = __importDefault(require("./SVGRect.cjs"));
/**
 * SVG Animated Number.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGAnimatedRect
 */
class SVGAnimatedRect {
    // Internal properties
    [PropertySymbol.window];
    [PropertySymbol.getAttribute];
    [PropertySymbol.setAttribute];
    [PropertySymbol.baseVal] = null;
    [PropertySymbol.animVal] = null;
    /**
     * Constructor.
     *
     * @param illegalConstructorSymbol Illegal constructor symbol.
     * @param window Window.
     * @param options Options.
     * @param options.getAttribute Get attribute.
     * @param options.setAttribute Set attribute.
     */
    constructor(illegalConstructorSymbol, window, options) {
        if (illegalConstructorSymbol !== PropertySymbol.illegalConstructor) {
            throw new TypeError('Illegal constructor');
        }
        this[PropertySymbol.window] = window;
        this[PropertySymbol.getAttribute] = options.getAttribute;
        this[PropertySymbol.setAttribute] = options.setAttribute;
    }
    /**
     * Returns animated value.
     *
     * @returns Animated value.
     */
    get animVal() {
        if (!this[PropertySymbol.animVal]) {
            this[PropertySymbol.animVal] = new SVGRect_js_1.default(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                readOnly: true,
                getAttribute: this[PropertySymbol.getAttribute]
            });
        }
        return this[PropertySymbol.animVal];
    }
    /**
     * Returns animated value.
     *
     * @param value Animated value.
     */
    set animVal(_value) {
        // Do nothing
    }
    /**
     * Returns base value.
     *
     * @returns Base value.
     */
    get baseVal() {
        if (!this[PropertySymbol.baseVal]) {
            this[PropertySymbol.baseVal] = new SVGRect_js_1.default(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: this[PropertySymbol.getAttribute],
                setAttribute: this[PropertySymbol.setAttribute]
            });
        }
        return this[PropertySymbol.baseVal];
    }
    /**
     * Returns base value.
     *
     * @param value Base value.
     */
    set baseVal(_value) {
        // Do nothing
    }
}
exports.default = SVGAnimatedRect;
//# sourceMappingURL=SVGAnimatedRect.cjs.map