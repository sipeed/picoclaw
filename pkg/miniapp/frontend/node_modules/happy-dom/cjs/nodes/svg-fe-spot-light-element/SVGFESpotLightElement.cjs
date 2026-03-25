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
 * SVGFESpotLightElement.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGFESpotLightElement
 */
class SVGFESpotLightElement extends SVGElement_js_1.default {
    // Internal properties
    [PropertySymbol.x] = null;
    [PropertySymbol.y] = null;
    [PropertySymbol.z] = null;
    [PropertySymbol.pointsAtX] = null;
    [PropertySymbol.pointsAtY] = null;
    [PropertySymbol.pointsAtZ] = null;
    [PropertySymbol.specularExponent] = null;
    [PropertySymbol.limitingConeAngle] = null;
    /**
     * Returns x.
     *
     * @returns X.
     */
    get x() {
        if (!this[PropertySymbol.x]) {
            this[PropertySymbol.x] = new SVGAnimatedNumber_js_1.default(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
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
            this[PropertySymbol.y] = new SVGAnimatedNumber_js_1.default(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('y'),
                setAttribute: (value) => this.setAttribute('y', value)
            });
        }
        return this[PropertySymbol.y];
    }
    /**
     * Returns z.
     *
     * @returns Z.
     */
    get z() {
        if (!this[PropertySymbol.z]) {
            this[PropertySymbol.z] = new SVGAnimatedNumber_js_1.default(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('z'),
                setAttribute: (value) => this.setAttribute('z', value)
            });
        }
        return this[PropertySymbol.z];
    }
    /**
     * Returns pointsAtX.
     *
     * @returns PointsAtX.
     */
    get pointsAtX() {
        if (!this[PropertySymbol.pointsAtX]) {
            this[PropertySymbol.pointsAtX] = new SVGAnimatedNumber_js_1.default(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('pointsAtX'),
                setAttribute: (value) => this.setAttribute('pointsAtX', value)
            });
        }
        return this[PropertySymbol.pointsAtX];
    }
    /**
     * Returns pointsAtY.
     *
     * @returns PointsAtY.
     */
    get pointsAtY() {
        if (!this[PropertySymbol.pointsAtY]) {
            this[PropertySymbol.pointsAtY] = new SVGAnimatedNumber_js_1.default(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('pointsAtY'),
                setAttribute: (value) => this.setAttribute('pointsAtY', value)
            });
        }
        return this[PropertySymbol.pointsAtY];
    }
    /**
     * Returns pointsAtZ.
     *
     * @returns PointsAtZ.
     */
    get pointsAtZ() {
        if (!this[PropertySymbol.pointsAtZ]) {
            this[PropertySymbol.pointsAtZ] = new SVGAnimatedNumber_js_1.default(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('pointsAtZ'),
                setAttribute: (value) => this.setAttribute('pointsAtZ', value)
            });
        }
        return this[PropertySymbol.pointsAtZ];
    }
    /**
     * Returns specularExponent.
     *
     * @returns SpecularExponent.
     */
    get specularExponent() {
        if (!this[PropertySymbol.specularExponent]) {
            this[PropertySymbol.specularExponent] = new SVGAnimatedNumber_js_1.default(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('specularExponent'),
                setAttribute: (value) => this.setAttribute('specularExponent', value),
                defaultValue: 1
            });
        }
        return this[PropertySymbol.specularExponent];
    }
    /**
     * Returns limitingConeAngle.
     *
     * @returns LimitingConeAngle.
     */
    get limitingConeAngle() {
        if (!this[PropertySymbol.limitingConeAngle]) {
            this[PropertySymbol.limitingConeAngle] = new SVGAnimatedNumber_js_1.default(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('limitingConeAngle'),
                setAttribute: (value) => this.setAttribute('limitingConeAngle', value)
            });
        }
        return this[PropertySymbol.limitingConeAngle];
    }
}
exports.default = SVGFESpotLightElement;
//# sourceMappingURL=SVGFESpotLightElement.cjs.map