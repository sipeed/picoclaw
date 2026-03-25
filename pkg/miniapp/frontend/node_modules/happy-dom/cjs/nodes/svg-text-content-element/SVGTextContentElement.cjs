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
const SVGGraphicsElement_js_1 = __importDefault(require("../svg-graphics-element/SVGGraphicsElement.cjs"));
const PropertySymbol = __importStar(require("../../PropertySymbol.cjs"));
const SVGAnimatedLength_js_1 = __importDefault(require("../../svg/SVGAnimatedLength.cjs"));
const SVGAnimatedEnumeration_js_1 = __importDefault(require("../../svg/SVGAnimatedEnumeration.cjs"));
const SVGPoint_js_1 = __importDefault(require("../../svg/SVGPoint.cjs"));
const SVGRect_js_1 = __importDefault(require("../../svg/SVGRect.cjs"));
/**
 * SVG Text Content Element.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGTextContentElement
 */
class SVGTextContentElement extends SVGGraphicsElement_js_1.default {
    // Public static properties
    static LENGTHADJUST_UNKNOWN = 0;
    static LENGTHADJUST_SPACING = 1;
    static LENGTHADJUST_SPACINGANDGLYPHS = 2;
    // Internal properties
    [PropertySymbol.textLength] = null;
    [PropertySymbol.lengthAdjust] = null;
    /**
     * Returns textLength.
     *
     * @returns Text length.
     */
    get textLength() {
        if (!this[PropertySymbol.textLength]) {
            this[PropertySymbol.textLength] = new SVGAnimatedLength_js_1.default(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('textLength'),
                setAttribute: (value) => this.setAttribute('textLength', value)
            });
        }
        return this[PropertySymbol.textLength];
    }
    /**
     * Returns lengthAdjust.
     *
     * @returns Length adjust.
     */
    get lengthAdjust() {
        if (!this[PropertySymbol.lengthAdjust]) {
            this[PropertySymbol.lengthAdjust] = new SVGAnimatedEnumeration_js_1.default(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('lengthAdjust'),
                setAttribute: (value) => this.setAttribute('lengthAdjust', value),
                values: ['spacing', 'spacingAndGlyphs'],
                defaultValue: 'spacing'
            });
        }
        return this[PropertySymbol.lengthAdjust];
    }
    /**
     * Returns the number of characters available for rendering.
     *
     * @returns Number of characters.
     */
    getNumberOfChars() {
        // TODO: Implement.
        return 0;
    }
    /**
     * Returns a float representing the computed length for the text within the element.
     *
     * @returns Computed text length.
     */
    getComputedTextLength() {
        // TODO: Implement.
        return 0;
    }
    /**
     * Returns a float representing the computed length of the formatted text advance distance for a substring of text within the element. Note that this method only accounts for the widths of the glyphs in the substring and any extra spacing inserted by the CSS 'letter-spacing' and 'word-spacing' properties. Visual spacing adjustments made by the 'x' attribute is ignored.
     *
     * @param _charnum The index of the first character in the substring.
     * @param _nchars The number of characters in the substring.
     */
    getSubStringLength(_charnum, _nchars) {
        // TODO: Implement.
        return 0;
    }
    /**
     * Returns a SVGPoint representing the position of a typographic character after text layout has been performed.
     *
     * @param _charnum The index of the character.
     */
    getStartPositionOfChar(_charnum) {
        // TODO: Implement.
        return new SVGPoint_js_1.default(PropertySymbol.illegalConstructor, this[PropertySymbol.window]);
    }
    /**
     * Returns a SVGPoint representing the trailing position of a typographic character after text layout has been performed.
     *
     * @param _charnum The index of the character.
     */
    getEndPositionOfChar(_charnum) {
        // TODO: Implement.
        return new SVGPoint_js_1.default(PropertySymbol.illegalConstructor, this[PropertySymbol.window]);
    }
    /**
     * Returns a SVGRect representing the computed tight bounding box of the glyph cell that corresponds to a given typographic character.
     *
     * @param _charnum The index of the character.
     */
    getExtentOfChar(_charnum) {
        // TODO: Implement.
        return new SVGRect_js_1.default(PropertySymbol.illegalConstructor, this[PropertySymbol.window]);
    }
    /**
     * Returns a float representing the rotation of typographic character.
     *
     * @param _charnum The index of the character.
     */
    getRotationOfChar(_charnum) {
        // TODO: Implement.
        return 0;
    }
    /**
     * Returns a long representing the character which caused a text glyph to be rendered at a given position in the coordinate system. Because the relationship between characters and glyphs is not one-to-one, only the first character of the relevant typographic character is returned
     *
     * @param _point The point to be tested.
     */
    getCharNumAtPosition(_point) {
        // TODO: Implement.
        return 0;
    }
}
exports.default = SVGTextContentElement;
//# sourceMappingURL=SVGTextContentElement.cjs.map