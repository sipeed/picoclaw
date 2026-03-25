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
const SVGAnimatedLength_js_1 = __importDefault(require("../../svg/SVGAnimatedLength.cjs"));
const SVGAnimatedString_js_1 = __importDefault(require("../../svg/SVGAnimatedString.cjs"));
const SVGAnimatedNumber_js_1 = __importDefault(require("../../svg/SVGAnimatedNumber.cjs"));
const SVGAnimatedInteger_js_1 = __importDefault(require("../../svg/SVGAnimatedInteger.cjs"));
const SVGAnimatedEnumeration_js_1 = __importDefault(require("../../svg/SVGAnimatedEnumeration.cjs"));
/**
 * SVGFETurbulenceElement.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGFETurbulenceElement
 */
class SVGFETurbulenceElement extends SVGElement_js_1.default {
    // Static properties
    static SVG_TURBULENCE_TYPE_UNKNOWN = 0;
    static SVG_TURBULENCE_TYPE_FRACTALNOISE = 1;
    static SVG_TURBULENCE_TYPE_TURBULENCE = 2;
    static SVG_STITCHTYPE_UNKNOWN = 0;
    static SVG_STITCHTYPE_STITCH = 1;
    static SVG_STITCHTYPE_NOSTITCH = 2;
    // Internal properties
    [PropertySymbol.baseFrequencyX] = null;
    [PropertySymbol.baseFrequencyY] = null;
    [PropertySymbol.height] = null;
    [PropertySymbol.numOctaves] = null;
    [PropertySymbol.result] = null;
    [PropertySymbol.seed] = null;
    [PropertySymbol.stitchTiles] = null;
    [PropertySymbol.type] = null;
    [PropertySymbol.width] = null;
    [PropertySymbol.x] = null;
    [PropertySymbol.y] = null;
    /**
     * Returns baseFrequencyX.
     *
     * @returns Base frequency x.
     */
    get baseFrequencyX() {
        if (!this[PropertySymbol.baseFrequencyX]) {
            this[PropertySymbol.baseFrequencyX] = new SVGAnimatedNumber_js_1.default(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('baseFrequencyX'),
                setAttribute: (value) => this.setAttribute('baseFrequencyX', value)
            });
        }
        return this[PropertySymbol.baseFrequencyX];
    }
    /**
     * Returns baseFrequencyY.
     *
     * @returns Base frequency y.
     */
    get baseFrequencyY() {
        if (!this[PropertySymbol.baseFrequencyY]) {
            this[PropertySymbol.baseFrequencyY] = new SVGAnimatedNumber_js_1.default(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('baseFrequencyY'),
                setAttribute: (value) => this.setAttribute('baseFrequencyY', value)
            });
        }
        return this[PropertySymbol.baseFrequencyY];
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
     * Returns numOctaves.
     *
     * @returns Num octaves.
     */
    get numOctaves() {
        if (!this[PropertySymbol.numOctaves]) {
            this[PropertySymbol.numOctaves] = new SVGAnimatedInteger_js_1.default(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('numOctaves'),
                setAttribute: (value) => this.setAttribute('numOctaves', value)
            });
        }
        return this[PropertySymbol.numOctaves];
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
     * Returns seed.
     *
     * @returns Seed.
     */
    get seed() {
        if (!this[PropertySymbol.seed]) {
            this[PropertySymbol.seed] = new SVGAnimatedNumber_js_1.default(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('seed'),
                setAttribute: (value) => this.setAttribute('seed', value)
            });
        }
        return this[PropertySymbol.seed];
    }
    /**
     * Returns stitchTiles.
     *
     * @returns Stitch tiles.
     */
    get stitchTiles() {
        if (!this[PropertySymbol.stitchTiles]) {
            this[PropertySymbol.stitchTiles] = new SVGAnimatedEnumeration_js_1.default(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('stitchTiles'),
                setAttribute: (value) => this.setAttribute('stitchTiles', value),
                values: ['stitch', 'noStitch'],
                defaultValue: 'stitch'
            });
        }
        return this[PropertySymbol.stitchTiles];
    }
    /**
     * Returns type.
     *
     * @returns Type.
     */
    get type() {
        if (!this[PropertySymbol.type]) {
            this[PropertySymbol.type] = new SVGAnimatedEnumeration_js_1.default(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('type'),
                setAttribute: (value) => this.setAttribute('type', value),
                values: ['fractalNoise', 'turbulence'],
                defaultValue: 'turbulence'
            });
        }
        return this[PropertySymbol.type];
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
     * Returns x.
     *
     * @returns X.
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
     * Returns y.
     *
     * @returns Y.
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
exports.default = SVGFETurbulenceElement;
//# sourceMappingURL=SVGFETurbulenceElement.cjs.map