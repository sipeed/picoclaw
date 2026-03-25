import ICSSStyleDeclarationPropertyValue from './ICSSStyleDeclarationPropertyValue.cjs';
/**
 * Computed style property parser.
 */
export default class CSSStyleDeclarationPropertySetParser {
    /**
     * Returns border collapse.
     *
     * @param value Value.
     * @param important Important.
     * @returns Property values
     */
    static getBorderCollapse(value: string, important: boolean): {
        [key: string]: ICSSStyleDeclarationPropertyValue;
    };
    /**
     * Returns display.
     *
     * @param value Value.
     * @param important Important.
     * @returns Property values
     */
    static getDisplay(value: string, important: boolean): {
        [key: string]: ICSSStyleDeclarationPropertyValue;
    };
    /**
     * Returns direction.
     *
     * @param value Value.
     * @param important Important.
     * @returns Property values
     */
    static getDirection(value: string, important: boolean): {
        [key: string]: ICSSStyleDeclarationPropertyValue;
    };
    /**
     * Returns letter spacing.
     *
     * @param value Value.
     * @param important Important.
     * @returns Property values
     */
    static getLetterSpacing(value: string, important: boolean): {
        [key: string]: ICSSStyleDeclarationPropertyValue;
    };
    /**
     * Returns word spacing.
     *
     * @param value Value.
     * @param important Important.
     * @returns Property values
     */
    static getWordSpacing(value: string, important: boolean): {
        [key: string]: ICSSStyleDeclarationPropertyValue;
    };
    /**
     * Returns text indent.
     *
     * @param value Value.
     * @param important Important.
     * @returns Property values
     */
    static getTextIndent(value: string, important: boolean): {
        [key: string]: ICSSStyleDeclarationPropertyValue;
    };
    /**
     * Returns width.
     *
     * @param value Value.
     * @param important Important.
     * @returns Property values
     */
    static getWidth(value: string, important: boolean): {
        [key: string]: ICSSStyleDeclarationPropertyValue;
    };
    /**
     * Returns height.
     *
     * @param value Value.
     * @param important Important.
     * @returns Property values
     */
    static getHeight(value: string, important: boolean): {
        [key: string]: ICSSStyleDeclarationPropertyValue;
    };
    /**
     * Returns top.
     *
     * @param value Value.
     * @param important Important.
     * @returns Property values
     */
    static getTop(value: string, important: boolean): {
        [key: string]: ICSSStyleDeclarationPropertyValue;
    };
    /**
     * Returns top.
     *
     * @param value Value.
     * @param important Important.
     * @returns Property values
     */
    static getRight(value: string, important: boolean): {
        [key: string]: ICSSStyleDeclarationPropertyValue;
    };
    /**
     * Returns top.
     *
     * @param value Value.
     * @param important Important.
     * @returns Property values
     */
    static getBottom(value: string, important: boolean): {
        [key: string]: ICSSStyleDeclarationPropertyValue;
    };
    /**
     * Returns top.
     *
     * @param value Value.
     * @param important Important.
     * @returns Property values
     */
    static getLeft(value: string, important: boolean): {
        [key: string]: ICSSStyleDeclarationPropertyValue;
    };
    /**
     * Returns clear.
     *
     * @param value Value.
     * @param important Important.
     * @returns Property values
     */
    static getClear(value: string, important: boolean): {
        [key: string]: ICSSStyleDeclarationPropertyValue;
    };
    /**
     * Returns clip
     *
     * Based on:
     * https://github.com/jsdom/cssstyle/blob/master/lib/properties/clip.js
     *
     * @param value Value.
     * @param important Important.
     * @returns Property values
     */
    static getClip(value: string, important: boolean): {
        [key: string]: ICSSStyleDeclarationPropertyValue;
    };
    /**
     * Returns float.
     *
     * @param value Value.
     * @param important Important.
     * @returns Property values
     */
    static getFloat(value: string, important: boolean): {
        [key: string]: ICSSStyleDeclarationPropertyValue;
    };
    /**
     * Returns float.
     *
     * @param value Value.
     * @param important Important.
     * @returns Property values
     */
    static getCSSFloat(value: string, important: boolean): {
        [key: string]: ICSSStyleDeclarationPropertyValue;
    };
    /**
     * Returns outline.
     *
     * @param value Value.
     * @param important Important.
     * @returns Property values.
     */
    static getOutline(value: string, important: boolean): {
        [key: string]: ICSSStyleDeclarationPropertyValue;
    };
    /**
     * Returns outline color.
     *
     * @param value Value.
     * @param important Important.
     * @returns Property values
     */
    static getOutlineColor(value: string, important: boolean): {
        [key: string]: ICSSStyleDeclarationPropertyValue;
    };
    /**
     * Returns outline offset.
     *
     * @param value Value.
     * @param important Important.
     * @returns Property values
     */
    static getOutlineOffset(value: string, important: boolean): {
        [key: string]: ICSSStyleDeclarationPropertyValue;
    };
    /**
     * Returns outline style.
     *
     * @param value Value.
     * @param important Important.
     * @returns Property values
     */
    static getOutlineStyle(value: string, important: boolean): {
        [key: string]: ICSSStyleDeclarationPropertyValue;
    };
    /**
     * Returns outline width.
     *
     * @param value Value.
     * @param important Important.
     * @returns Property values
     */
    static getOutlineWidth(value: string, important: boolean): {
        [key: string]: ICSSStyleDeclarationPropertyValue;
    };
    /**
     * Returns border.
     *
     * @param value Value.
     * @param important Important.
     * @returns Property values.
     */
    static getBorder(value: string, important: boolean): {
        [key: string]: ICSSStyleDeclarationPropertyValue;
    };
    /**
     * Returns border width.
     *
     * @param value Value.
     * @param important Important.
     * @returns Property values
     */
    static getBorderWidth(value: string, important: boolean): {
        [key: string]: ICSSStyleDeclarationPropertyValue;
    };
    /**
     * Returns border style.
     *
     * @param value Value.
     * @param important Important.
     * @returns Property values
     */
    static getBorderStyle(value: string, important: boolean): {
        [key: string]: ICSSStyleDeclarationPropertyValue;
    };
    /**
     * Returns border color.
     *
     * @param value Value.
     * @param important Important.
     * @returns Property values
     */
    static getBorderColor(value: string, important: boolean): {
        [key: string]: ICSSStyleDeclarationPropertyValue;
    };
    /**
     * Returns border image.
     *
     * @param value Value.
     * @param important Important.
     * @returns Property values
     */
    static getBorderImage(value: string, important: boolean): {
        [key: string]: ICSSStyleDeclarationPropertyValue;
    };
    /**
     * Returns border source.
     *
     * @param value Value.
     * @param important Important.
     * @returns Property values
     */
    static getBorderImageSource(value: string, important: boolean): {
        [key: string]: ICSSStyleDeclarationPropertyValue;
    };
    /**
     * Returns border slice.
     *
     * @param value Value.
     * @param important Important.
     * @returns Property values
     */
    static getBorderImageSlice(value: string, important: boolean): {
        [key: string]: ICSSStyleDeclarationPropertyValue;
    };
    /**
     * Returns border width.
     *
     * @param value Value.
     * @param important Important.
     * @returns Property values
     */
    static getBorderImageWidth(value: string, important: boolean): {
        [key: string]: ICSSStyleDeclarationPropertyValue;
    };
    /**
     * Returns border outset.
     *
     * @param value Value.
     * @param important Important.
     * @returns Property values
     */
    static getBorderImageOutset(value: string, important: boolean): {
        [key: string]: ICSSStyleDeclarationPropertyValue;
    };
    /**
     * Returns border repeat.
     *
     * @param value Value.
     * @param important Important.
     * @returns Property values
     */
    static getBorderImageRepeat(value: string, important: boolean): {
        [key: string]: ICSSStyleDeclarationPropertyValue;
    };
    /**
     * Returns border width.
     *
     * @param value Value.
     * @param important Important.
     * @returns Property values
     */
    static getBorderTopWidth(value: string, important: boolean): {
        [key: string]: ICSSStyleDeclarationPropertyValue;
    };
    /**
     * Returns border width.
     *
     * @param value Value.
     * @param important Important.
     * @returns Property values
     */
    static getBorderRightWidth(value: string, important: boolean): {
        [key: string]: ICSSStyleDeclarationPropertyValue;
    };
    /**
     * Returns border width.
     *
     * @param value Value.
     * @param important Important.
     * @returns Property values
     */
    static getBorderBottomWidth(value: string, important: boolean): {
        [key: string]: ICSSStyleDeclarationPropertyValue;
    };
    /**
     * Returns border width.
     *
     * @param value Value.
     * @param important Important.
     * @returns Property values
     */
    static getBorderLeftWidth(value: string, important: boolean): {
        [key: string]: ICSSStyleDeclarationPropertyValue;
    };
    /**
     * Returns border style.
     *
     * @param value Value.
     * @param important Important.
     * @returns Property values
     */
    static getBorderTopStyle(value: string, important: boolean): {
        [key: string]: ICSSStyleDeclarationPropertyValue;
    };
    /**
     * Returns border style.
     *
     * @param value Value.
     * @param important Important.
     * @returns Property values
     */
    static getBorderRightStyle(value: string, important: boolean): {
        [key: string]: ICSSStyleDeclarationPropertyValue;
    };
    /**
     * Returns border style.
     *
     * @param value Value.
     * @param important Important.
     * @returns Property values
     */
    static getBorderBottomStyle(value: string, important: boolean): {
        [key: string]: ICSSStyleDeclarationPropertyValue;
    };
    /**
     * Returns border style.
     *
     * @param value Value.
     * @param important Important.
     * @returns Property values
     */
    static getBorderLeftStyle(value: string, important: boolean): {
        [key: string]: ICSSStyleDeclarationPropertyValue;
    };
    /**
     * Returns border color.
     *
     * @param value Value.
     * @param important Important.
     * @returns Property values
     */
    static getBorderTopColor(value: string, important: boolean): {
        [key: string]: ICSSStyleDeclarationPropertyValue;
    };
    /**
     * Returns border color.
     *
     * @param value Value.
     * @param important Important.
     * @returns Property values
     */
    static getBorderRightColor(value: string, important: boolean): {
        [key: string]: ICSSStyleDeclarationPropertyValue;
    };
    /**
     * Returns border color.
     *
     * @param value Value.
     * @param important Important.
     * @returns Property values
     */
    static getBorderBottomColor(value: string, important: boolean): {
        [key: string]: ICSSStyleDeclarationPropertyValue;
    };
    /**
     * Returns border color.
     *
     * @param value Value.
     * @param important Important.
     * @returns Property values
     */
    static getBorderLeftColor(value: string, important: boolean): {
        [key: string]: ICSSStyleDeclarationPropertyValue;
    };
    /**
     * Returns border radius.
     *
     * @param value Value.
     * @param important Important.
     * @returns Property values.
     */
    static getBorderRadius(value: string, important: boolean): {
        [key: string]: ICSSStyleDeclarationPropertyValue;
    };
    /**
     * Returns border radius.
     *
     * @param value Value.
     * @param important Important.
     * @returns Property values.
     */
    static getBorderTopLeftRadius(value: string, important: boolean): {
        [key: string]: ICSSStyleDeclarationPropertyValue;
    };
    /**
     * Returns border radius.
     *
     * @param value Value.
     * @param important Important.
     * @returns Property values.
     */
    static getBorderTopRightRadius(value: string, important: boolean): {
        [key: string]: ICSSStyleDeclarationPropertyValue;
    };
    /**
     * Returns border radius.
     *
     * @param value Value.
     * @param important Important.
     * @returns Property values.
     */
    static getBorderBottomRightRadius(value: string, important: boolean): {
        [key: string]: ICSSStyleDeclarationPropertyValue;
    };
    /**
     * Returns border radius.
     *
     * @param value Value.
     * @param important Important.
     * @returns Property values.
     */
    static getBorderBottomLeftRadius(value: string, important: boolean): {
        [key: string]: ICSSStyleDeclarationPropertyValue;
    };
    /**
     * Returns border top, right, bottom or left.
     *
     * @param value Value.
     * @param important Important.
     * @returns Property values.
     */
    static getBorderTop(value: string, important: boolean): {
        [key: string]: ICSSStyleDeclarationPropertyValue;
    };
    /**
     * Returns border top, right, bottom or left.
     *
     * @param value Value.
     * @param important Important.
     * @returns Property values.
     */
    static getBorderRight(value: string, important: boolean): {
        [key: string]: ICSSStyleDeclarationPropertyValue;
    };
    /**
     * Returns border top, right, bottom or left.
     *
     * @param value Value.
     * @param important Important.
     * @returns Property values.
     */
    static getBorderBottom(value: string, important: boolean): {
        [key: string]: ICSSStyleDeclarationPropertyValue;
    };
    /**
     * Returns border top, right, bottom or left.
     *
     * @param value Value.
     * @param important Important.
     * @returns Property values.
     */
    static getBorderLeft(value: string, important: boolean): {
        [key: string]: ICSSStyleDeclarationPropertyValue;
    };
    /**
     * Returns padding.
     *
     * @param value Value.
     * @param important Important.
     */
    static getPadding(value: string, important: boolean): {
        [key: string]: ICSSStyleDeclarationPropertyValue;
    };
    /**
     * Returns padding top.
     *
     * @param value Value.
     * @param important Important.
     * @returns Property values.
     */
    static getPaddingTop(value: string, important: boolean): {
        [key: string]: ICSSStyleDeclarationPropertyValue;
    };
    /**
     * Returns padding right.
     *
     * @param value Value.
     * @param important Important.
     * @returns Property values.
     */
    static getPaddingRight(value: string, important: boolean): {
        [key: string]: ICSSStyleDeclarationPropertyValue;
    };
    /**
     * Returns padding bottom.
     *
     * @param value Value.
     * @param important Important.
     * @returns Property values.
     */
    static getPaddingBottom(value: string, important: boolean): {
        [key: string]: ICSSStyleDeclarationPropertyValue;
    };
    /**
     * Returns padding left.
     *
     * @param value Value.
     * @param important Important.
     * @returns Property values.
     */
    static getPaddingLeft(value: string, important: boolean): {
        [key: string]: ICSSStyleDeclarationPropertyValue;
    };
    /**
     * Returns margin.
     *
     * @param value Value.
     * @param important Important.
     * @returns Property values.
     */
    static getMargin(value: string, important: boolean): {
        [key: string]: ICSSStyleDeclarationPropertyValue;
    };
    /**
     * Returns margin top.
     *
     * @param value Value.
     * @param important Important.
     * @returns Property values.
     */
    static getMarginTop(value: string, important: boolean): {
        [key: string]: ICSSStyleDeclarationPropertyValue;
    };
    /**
     * Returns margin right.
     *
     * @param value Value.
     * @param important Important.
     * @returns Property values.
     */
    static getMarginRight(value: string, important: boolean): {
        [key: string]: ICSSStyleDeclarationPropertyValue;
    };
    /**
     * Returns margin right.
     *
     * @param value Value.
     * @param important Important.
     * @returns Property values.
     */
    static getMarginBottom(value: string, important: boolean): {
        [key: string]: ICSSStyleDeclarationPropertyValue;
    };
    /**
     * Returns margin left.
     *
     * @param value Value.
     * @param important Important.
     * @returns Property values.
     */
    static getMarginLeft(value: string, important: boolean): {
        [key: string]: ICSSStyleDeclarationPropertyValue;
    };
    /**
     * Returns flex.
     *
     * @param value Value.
     * @param important Important.
     * @returns Property values.
     */
    static getFlex(value: string, important: boolean): {
        [key: string]: ICSSStyleDeclarationPropertyValue;
    };
    /**
     * Returns flex basis.
     *
     * @param value Value.
     * @param important Important.
     * @returns Property values
     */
    static getFlexBasis(value: string, important: boolean): {
        [key: string]: ICSSStyleDeclarationPropertyValue;
    };
    /**
     * Returns flex shrink.
     *
     * @param value Value.
     * @param important Important.
     * @returns Property values
     */
    static getFlexShrink(value: string, important: boolean): {
        [key: string]: ICSSStyleDeclarationPropertyValue;
    };
    /**
     * Returns flex grow.
     *
     * @param value Value.
     * @param important Important.
     * @returns Property values
     */
    static getFlexGrow(value: string, important: boolean): {
        [key: string]: ICSSStyleDeclarationPropertyValue;
    };
    /**
     * Returns background.
     *
     * @param name Name.
     * @param value Value.
     * @param important Important.
     * @returns Property values.
     */
    static getBackground(value: string, important: boolean): {
        [key: string]: ICSSStyleDeclarationPropertyValue;
    };
    /**
     * Returns background size.
     *
     * @param value Value.
     * @param important Important.
     * @returns Property values
     */
    static getBackgroundSize(value: string, important: boolean): {
        [key: string]: ICSSStyleDeclarationPropertyValue;
    };
    /**
     * Returns background origin.
     *
     * @param value Value.
     * @param important Important.
     * @returns Property values
     */
    static getBackgroundOrigin(value: string, important: boolean): {
        [key: string]: ICSSStyleDeclarationPropertyValue;
    };
    /**
     * Returns background clip.
     *
     * @param value Value.
     * @param important Important.
     * @returns Property values
     */
    static getBackgroundClip(value: string, important: boolean): {
        [key: string]: ICSSStyleDeclarationPropertyValue;
    };
    /**
     * Returns background repeat.
     *
     * @param value Value.
     * @param important Important.
     * @returns Property values
     */
    static getBackgroundRepeat(value: string, important: boolean): {
        [key: string]: ICSSStyleDeclarationPropertyValue;
    };
    /**
     * Returns background attachment.
     *
     * @param value Value.
     * @param important Important.
     * @returns Property values
     */
    static getBackgroundAttachment(value: string, important: boolean): {
        [key: string]: ICSSStyleDeclarationPropertyValue;
    };
    /**
     * Returns background position.
     *
     * @param value Value.
     * @param important Important.
     * @returns Property values
     */
    static getBackgroundPosition(value: string, important: boolean): {
        [key: string]: ICSSStyleDeclarationPropertyValue;
    };
    /**
     * Returns background position.
     *
     * @param value Value.
     * @param important Important.
     * @returns Property values
     */
    static getBackgroundPositionX(value: string, important: boolean): {
        [key: string]: ICSSStyleDeclarationPropertyValue;
    };
    /**
     * Returns background position.
     *
     * @param value Value.
     * @param important Important.
     * @returns Property values
     */
    static getBackgroundPositionY(value: string, important: boolean): {
        [key: string]: ICSSStyleDeclarationPropertyValue;
    };
    /**
     * Returns background color.
     *
     * @param value Value.
     * @param important Important.
     * @returns Property value.
     */
    static getBackgroundColor(value: string, important: boolean): {
        [key: string]: ICSSStyleDeclarationPropertyValue;
    };
    /**
     * Returns background image.
     *
     * @param value Value.
     * @param important Important.
     * @returns Property value.
     */
    static getBackgroundImage(value: string, important: boolean): {
        [key: string]: ICSSStyleDeclarationPropertyValue;
    };
    /**
     * Returns color.
     *
     * @param value Value.
     * @param important Important.
     * @returns Property value.
     */
    static getColor(value: string, important: boolean): {
        [key: string]: ICSSStyleDeclarationPropertyValue;
    };
    /**
     * Returns color.
     *
     * @param value Value.
     * @param important Important.
     * @returns Property value.
     */
    static getFloodColor(value: string, important: boolean): {
        [key: string]: ICSSStyleDeclarationPropertyValue;
    };
    /**
     * Returns font.
     *
     * @param value Value.
     * @param important Important.
     * @returns Property values
     */
    static getFont(value: string, important: boolean): {
        [key: string]: ICSSStyleDeclarationPropertyValue;
    };
    /**
     * Returns font style.
     *
     * @param value Value.
     * @param important Important.
     * @returns Property values
     */
    static getFontStyle(value: string, important: boolean): {
        [key: string]: ICSSStyleDeclarationPropertyValue;
    };
    /**
     * Returns font variant.
     *
     * @param value Value.
     * @param important Important.
     * @returns Property values
     */
    static getFontVariant(value: string, important: boolean): {
        [key: string]: ICSSStyleDeclarationPropertyValue;
    };
    /**
     * Returns font strech.
     *
     * @param value Value.
     * @param important Important.
     * @returns Property values
     */
    static getFontStretch(value: string, important: boolean): {
        [key: string]: ICSSStyleDeclarationPropertyValue;
    };
    /**
     * Returns font weight.
     *
     * @param value Value.
     * @param important Important.
     * @returns Property values
     */
    static getFontWeight(value: string, important: boolean): {
        [key: string]: ICSSStyleDeclarationPropertyValue;
    };
    /**
     * Returns font size.
     *
     * @param value Value.
     * @param important Important.
     * @returns Property values
     */
    static getFontSize(value: string, important: boolean): {
        [key: string]: ICSSStyleDeclarationPropertyValue;
    };
    /**
     * Returns line height.
     *
     * @param value Value.
     * @param important Important.
     * @returns Property values
     */
    static getLineHeight(value: string, important: boolean): {
        [key: string]: ICSSStyleDeclarationPropertyValue;
    };
    /**
     * Returns font family.
     *
     * @param value Value.
     * @param important Important.
     * @returns Property values
     */
    static getFontFamily(value: string, important: boolean): {
        [key: string]: ICSSStyleDeclarationPropertyValue;
    };
    /**
     * Returns font family.
     *
     * @param value Value.
     * @param important Important.
     * @returns Property values
     */
    static getTextTransform(value: string, important: boolean): {
        [key: string]: ICSSStyleDeclarationPropertyValue;
    };
    /**
     * Returns visibility.
     *
     * @param value Value.
     * @param important Important.
     * @returns Property
     */
    static getVisibility(value: string, important: boolean): {
        [key: string]: ICSSStyleDeclarationPropertyValue;
    };
    /**
     * Returns aspect ratio.
     *
     * @param value Value.
     * @param important Important.
     * @returns Property
     */
    static getAspectRatio(value: string, important: boolean): {
        [key: string]: ICSSStyleDeclarationPropertyValue;
    };
}
//# sourceMappingURL=CSSStyleDeclarationPropertySetParser.d.ts.map