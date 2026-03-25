import SVGGraphicsElement from '../svg-graphics-element/SVGGraphicsElement.cjs';
import * as PropertySymbol from '../../PropertySymbol.cjs';
import SVGAnimatedLength from '../../svg/SVGAnimatedLength.cjs';
import SVGAnimatedEnumeration from '../../svg/SVGAnimatedEnumeration.cjs';
import SVGPoint from '../../svg/SVGPoint.cjs';
import SVGRect from '../../svg/SVGRect.cjs';
/**
 * SVG Text Content Element.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGTextContentElement
 */
export default class SVGTextContentElement extends SVGGraphicsElement {
    static readonly LENGTHADJUST_UNKNOWN = 0;
    static readonly LENGTHADJUST_SPACING = 1;
    static readonly LENGTHADJUST_SPACINGANDGLYPHS = 2;
    [PropertySymbol.textLength]: SVGAnimatedLength | null;
    [PropertySymbol.lengthAdjust]: SVGAnimatedEnumeration | null;
    /**
     * Returns textLength.
     *
     * @returns Text length.
     */
    get textLength(): SVGAnimatedLength;
    /**
     * Returns lengthAdjust.
     *
     * @returns Length adjust.
     */
    get lengthAdjust(): SVGAnimatedEnumeration;
    /**
     * Returns the number of characters available for rendering.
     *
     * @returns Number of characters.
     */
    getNumberOfChars(): number;
    /**
     * Returns a float representing the computed length for the text within the element.
     *
     * @returns Computed text length.
     */
    getComputedTextLength(): number;
    /**
     * Returns a float representing the computed length of the formatted text advance distance for a substring of text within the element. Note that this method only accounts for the widths of the glyphs in the substring and any extra spacing inserted by the CSS 'letter-spacing' and 'word-spacing' properties. Visual spacing adjustments made by the 'x' attribute is ignored.
     *
     * @param _charnum The index of the first character in the substring.
     * @param _nchars The number of characters in the substring.
     */
    getSubStringLength(_charnum: number, _nchars: number): number;
    /**
     * Returns a SVGPoint representing the position of a typographic character after text layout has been performed.
     *
     * @param _charnum The index of the character.
     */
    getStartPositionOfChar(_charnum: number): SVGPoint;
    /**
     * Returns a SVGPoint representing the trailing position of a typographic character after text layout has been performed.
     *
     * @param _charnum The index of the character.
     */
    getEndPositionOfChar(_charnum: number): SVGPoint;
    /**
     * Returns a SVGRect representing the computed tight bounding box of the glyph cell that corresponds to a given typographic character.
     *
     * @param _charnum The index of the character.
     */
    getExtentOfChar(_charnum: number): SVGRect;
    /**
     * Returns a float representing the rotation of typographic character.
     *
     * @param _charnum The index of the character.
     */
    getRotationOfChar(_charnum: number): number;
    /**
     * Returns a long representing the character which caused a text glyph to be rendered at a given position in the coordinate system. Because the relationship between characters and glyphs is not one-to-one, only the first character of the relevant typographic character is returned
     *
     * @param _point The point to be tested.
     */
    getCharNumAtPosition(_point: SVGPoint): number;
}
//# sourceMappingURL=SVGTextContentElement.d.ts.map