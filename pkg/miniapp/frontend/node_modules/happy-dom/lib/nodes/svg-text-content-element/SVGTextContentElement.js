import SVGGraphicsElement from '../svg-graphics-element/SVGGraphicsElement.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import SVGAnimatedLength from '../../svg/SVGAnimatedLength.js';
import SVGAnimatedEnumeration from '../../svg/SVGAnimatedEnumeration.js';
import SVGPoint from '../../svg/SVGPoint.js';
import SVGRect from '../../svg/SVGRect.js';
/**
 * SVG Text Content Element.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGTextContentElement
 */
export default class SVGTextContentElement extends SVGGraphicsElement {
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
            this[PropertySymbol.textLength] = new SVGAnimatedLength(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
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
            this[PropertySymbol.lengthAdjust] = new SVGAnimatedEnumeration(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
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
        return new SVGPoint(PropertySymbol.illegalConstructor, this[PropertySymbol.window]);
    }
    /**
     * Returns a SVGPoint representing the trailing position of a typographic character after text layout has been performed.
     *
     * @param _charnum The index of the character.
     */
    getEndPositionOfChar(_charnum) {
        // TODO: Implement.
        return new SVGPoint(PropertySymbol.illegalConstructor, this[PropertySymbol.window]);
    }
    /**
     * Returns a SVGRect representing the computed tight bounding box of the glyph cell that corresponds to a given typographic character.
     *
     * @param _charnum The index of the character.
     */
    getExtentOfChar(_charnum) {
        // TODO: Implement.
        return new SVGRect(PropertySymbol.illegalConstructor, this[PropertySymbol.window]);
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
//# sourceMappingURL=SVGTextContentElement.js.map