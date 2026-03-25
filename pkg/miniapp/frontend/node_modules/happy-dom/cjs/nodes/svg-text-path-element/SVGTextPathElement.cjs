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
const SVGTextContentElement_js_1 = __importDefault(require("../svg-text-content-element/SVGTextContentElement.cjs"));
const PropertySymbol = __importStar(require("../../PropertySymbol.cjs"));
const SVGAnimatedString_js_1 = __importDefault(require("../../svg/SVGAnimatedString.cjs"));
const SVGAnimatedLength_js_1 = __importDefault(require("../../svg/SVGAnimatedLength.cjs"));
const SVGAnimatedEnumeration_js_1 = __importDefault(require("../../svg/SVGAnimatedEnumeration.cjs"));
/**
 * SVG Text Path Element.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGTextPathElement
 */
class SVGTextPathElement extends SVGTextContentElement_js_1.default {
    // Public static properties
    static TEXTPATH_METHODTYPE_UNKNOWN = 0;
    static TEXTPATH_METHODTYPE_ALIGN = 1;
    static TEXTPATH_METHODTYPE_STRETCH = 2;
    static TEXTPATH_SPACINGTYPE_UNKNOWN = 0;
    static TEXTPATH_SPACINGTYPE_AUTO = 1;
    static TEXTPATH_SPACINGTYPE_EXACT = 2;
    // Internal properties
    [PropertySymbol.href] = null;
    [PropertySymbol.startOffset] = null;
    [PropertySymbol.method] = null;
    [PropertySymbol.spacing] = null;
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
    /**
     * Returns start offset.
     *
     * @returns Start offset.
     */
    get startOffset() {
        if (!this[PropertySymbol.startOffset]) {
            this[PropertySymbol.startOffset] = new SVGAnimatedLength_js_1.default(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('startOffset'),
                setAttribute: (value) => this.setAttribute('startOffset', value)
            });
        }
        return this[PropertySymbol.startOffset];
    }
    /**
     * Returns method.
     *
     * @returns ClipPathUnits.
     */
    get method() {
        if (!this[PropertySymbol.method]) {
            this[PropertySymbol.method] = new SVGAnimatedEnumeration_js_1.default(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('method'),
                setAttribute: (value) => this.setAttribute('method', value),
                values: ['align', 'stretch'],
                defaultValue: 'align'
            });
        }
        return this[PropertySymbol.method];
    }
    /**
     * Returns spacing.
     *
     * @returns Spacing.
     */
    get spacing() {
        if (!this[PropertySymbol.spacing]) {
            this[PropertySymbol.spacing] = new SVGAnimatedEnumeration_js_1.default(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('spacing'),
                setAttribute: (value) => this.setAttribute('spacing', value),
                values: ['auto', 'exact'],
                defaultValue: 'exact'
            });
        }
        return this[PropertySymbol.spacing];
    }
}
exports.default = SVGTextPathElement;
//# sourceMappingURL=SVGTextPathElement.cjs.map