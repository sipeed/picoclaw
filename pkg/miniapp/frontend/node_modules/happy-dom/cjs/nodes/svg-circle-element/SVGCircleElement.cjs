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
const SVGGeometryElement_js_1 = __importDefault(require("../svg-geometry-element/SVGGeometryElement.cjs"));
const PropertySymbol = __importStar(require("../../PropertySymbol.cjs"));
const SVGAnimatedLength_js_1 = __importDefault(require("../../svg/SVGAnimatedLength.cjs"));
/**
 * SVG Circle Element.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGCircleElement
 */
class SVGCircleElement extends SVGGeometryElement_js_1.default {
    // Internal properties
    [PropertySymbol.cx] = null;
    [PropertySymbol.cy] = null;
    [PropertySymbol.r] = null;
    /**
     * Returns cx.
     *
     * @returns Cx.
     */
    get cx() {
        if (!this[PropertySymbol.cx]) {
            this[PropertySymbol.cx] = new SVGAnimatedLength_js_1.default(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('cx'),
                setAttribute: (value) => this.setAttribute('cx', value)
            });
        }
        return this[PropertySymbol.cx];
    }
    /**
     * Returns cy.
     *
     * @returns Cy.
     */
    get cy() {
        if (!this[PropertySymbol.cy]) {
            this[PropertySymbol.cy] = new SVGAnimatedLength_js_1.default(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('cy'),
                setAttribute: (value) => this.setAttribute('cy', value)
            });
        }
        return this[PropertySymbol.cy];
    }
    /**
     * Returns r.
     *
     * @returns R.
     */
    get r() {
        if (!this[PropertySymbol.r]) {
            this[PropertySymbol.r] = new SVGAnimatedLength_js_1.default(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('r'),
                setAttribute: (value) => this.setAttribute('r', value)
            });
        }
        return this[PropertySymbol.r];
    }
}
exports.default = SVGCircleElement;
//# sourceMappingURL=SVGCircleElement.cjs.map