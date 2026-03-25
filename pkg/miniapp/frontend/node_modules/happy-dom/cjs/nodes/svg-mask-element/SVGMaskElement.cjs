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
const SVGAnimatedEnumeration_js_1 = __importDefault(require("../../svg/SVGAnimatedEnumeration.cjs"));
const SVGAnimatedLength_js_1 = __importDefault(require("../../svg/SVGAnimatedLength.cjs"));
/**
 * SVG Mask Element.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGMaskElement
 */
class SVGMaskElement extends SVGElement_js_1.default {
    // Internal properties
    [PropertySymbol.maskUnits] = null;
    [PropertySymbol.maskContentUnits] = null;
    [PropertySymbol.x] = null;
    [PropertySymbol.y] = null;
    [PropertySymbol.width] = null;
    [PropertySymbol.height] = null;
    /**
     * Returns mask units.
     *
     * @returns Mask units.
     */
    get maskUnits() {
        if (!this[PropertySymbol.maskUnits]) {
            this[PropertySymbol.maskUnits] = new SVGAnimatedEnumeration_js_1.default(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('maskUnits'),
                setAttribute: (value) => this.setAttribute('maskUnits', value),
                values: ['userSpaceOnUse', 'objectBoundingBox'],
                defaultValue: 'userSpaceOnUse'
            });
        }
        return this[PropertySymbol.maskUnits];
    }
    /**
     * Returns mask content units.
     *
     * @returns Mask content units.
     */
    get maskContentUnits() {
        if (!this[PropertySymbol.maskContentUnits]) {
            this[PropertySymbol.maskContentUnits] = new SVGAnimatedEnumeration_js_1.default(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('maskContentUnits'),
                setAttribute: (value) => this.setAttribute('maskContentUnits', value),
                values: ['userSpaceOnUse', 'objectBoundingBox'],
                defaultValue: 'userSpaceOnUse'
            });
        }
        return this[PropertySymbol.maskContentUnits];
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
}
exports.default = SVGMaskElement;
//# sourceMappingURL=SVGMaskElement.cjs.map