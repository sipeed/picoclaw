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
const PropertySymbol = __importStar(require("../../PropertySymbol.cjs"));
const SVGAnimatedLength_js_1 = __importDefault(require("../../svg/SVGAnimatedLength.cjs"));
const SVGGradientElement_js_1 = __importDefault(require("../svg-gradient-element/SVGGradientElement.cjs"));
/**
 * SVG LinearGradient Element.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGLinearGradientElement
 */
class SVGLinearGradientElement extends SVGGradientElement_js_1.default {
    // Internal properties
    [PropertySymbol.x1] = null;
    [PropertySymbol.y1] = null;
    [PropertySymbol.x2] = null;
    [PropertySymbol.y2] = null;
    /**
     * Returns x1 position.
     *
     * @returns X1 position.
     */
    get x1() {
        if (!this[PropertySymbol.x1]) {
            this[PropertySymbol.x1] = new SVGAnimatedLength_js_1.default(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('x1'),
                setAttribute: (value) => this.setAttribute('x1', value)
            });
        }
        return this[PropertySymbol.x1];
    }
    /**
     * Returns y1 position.
     *
     * @returns Y1 position.
     */
    get y1() {
        if (!this[PropertySymbol.y1]) {
            this[PropertySymbol.y1] = new SVGAnimatedLength_js_1.default(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('y1'),
                setAttribute: (value) => this.setAttribute('y1', value)
            });
        }
        return this[PropertySymbol.y1];
    }
    /**
     * Returns x2 position.
     *
     * @returns X2 position.
     */
    get x2() {
        if (!this[PropertySymbol.x2]) {
            this[PropertySymbol.x2] = new SVGAnimatedLength_js_1.default(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('x2'),
                setAttribute: (value) => this.setAttribute('x2', value)
            });
        }
        return this[PropertySymbol.x2];
    }
    /**
     * Returns y2 position.
     *
     * @returns Y2 position.
     */
    get y2() {
        if (!this[PropertySymbol.y2]) {
            this[PropertySymbol.y2] = new SVGAnimatedLength_js_1.default(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('y2'),
                setAttribute: (value) => this.setAttribute('y2', value)
            });
        }
        return this[PropertySymbol.y2];
    }
}
exports.default = SVGLinearGradientElement;
//# sourceMappingURL=SVGLinearGradientElement.cjs.map