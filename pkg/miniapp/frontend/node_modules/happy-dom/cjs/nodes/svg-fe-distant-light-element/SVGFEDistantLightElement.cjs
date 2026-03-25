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
/**
 * SVGFEDistantLightElement.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGFEDistantLightElement
 */
class SVGFEDistantLightElement extends SVGElement_js_1.default {
    // Internal properties
    [PropertySymbol.azimuth] = null;
    [PropertySymbol.elevation] = null;
    /**
     * Returns azimuth.
     *
     * @returns Azimuth.
     */
    get azimuth() {
        if (!this[PropertySymbol.azimuth]) {
            this[PropertySymbol.azimuth] = new SVGAnimatedNumber_js_1.default(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('azimuth'),
                setAttribute: (value) => this.setAttribute('azimuth', value)
            });
        }
        return this[PropertySymbol.azimuth];
    }
    /**
     * Returns elevation.
     *
     * @returns Elevation.
     */
    get elevation() {
        if (!this[PropertySymbol.elevation]) {
            this[PropertySymbol.elevation] = new SVGAnimatedNumber_js_1.default(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('elevation'),
                setAttribute: (value) => this.setAttribute('elevation', value)
            });
        }
        return this[PropertySymbol.elevation];
    }
}
exports.default = SVGFEDistantLightElement;
//# sourceMappingURL=SVGFEDistantLightElement.cjs.map