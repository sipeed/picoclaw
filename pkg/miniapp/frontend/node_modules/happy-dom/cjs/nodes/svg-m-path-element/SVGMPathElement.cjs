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
const SVGAnimatedString_js_1 = __importDefault(require("../../svg/SVGAnimatedString.cjs"));
const SVGElement_js_1 = __importDefault(require("../svg-element/SVGElement.cjs"));
const PropertySymbol = __importStar(require("../../PropertySymbol.cjs"));
/**
 * SVG MPath Element.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGMPathElement
 */
class SVGMPathElement extends SVGElement_js_1.default {
    // Internal properties
    [PropertySymbol.href] = null;
    /**
     * Returns href.
     *
     * @returns Href.
     */
    get href() {
        if (!this[PropertySymbol.href]) {
            this[PropertySymbol.href] = new SVGAnimatedString_js_1.default(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('href'),
                setAttribute: (value) => this.setAttribute('href', value)
            });
        }
        return this[PropertySymbol.href];
    }
}
exports.default = SVGMPathElement;
//# sourceMappingURL=SVGMPathElement.cjs.map