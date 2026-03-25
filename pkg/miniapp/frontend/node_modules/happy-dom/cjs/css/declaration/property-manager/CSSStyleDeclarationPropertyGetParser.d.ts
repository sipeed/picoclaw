import ICSSStyleDeclarationPropertyValue from './ICSSStyleDeclarationPropertyValue.cjs';
/**
 * Computed style property parser.
 */
export default class CSSStyleDeclarationPropertyGetParser {
    /**
     * Returns margin.
     *
     * @param properties Properties.
     * @returns Property value
     */
    static getMargin(properties: {
        [k: string]: ICSSStyleDeclarationPropertyValue;
    }): ICSSStyleDeclarationPropertyValue | null;
    /**
     * Returns padding.
     *
     * @param properties Properties.
     * @returns Property value
     */
    static getPadding(properties: {
        [k: string]: ICSSStyleDeclarationPropertyValue;
    }): ICSSStyleDeclarationPropertyValue | null;
    /**
     * Returns outline.
     *
     * @param properties Properties.
     * @returns Property value
     */
    static getOutline(properties: {
        [k: string]: ICSSStyleDeclarationPropertyValue;
    }): ICSSStyleDeclarationPropertyValue | null;
    /**
     * Returns border.
     *
     * @param properties Properties.
     * @returns Property value
     */
    static getBorder(properties: {
        [k: string]: ICSSStyleDeclarationPropertyValue;
    }): ICSSStyleDeclarationPropertyValue | null;
    /**
     * Returns border.
     *
     * @param properties Properties.
     * @returns Property value
     */
    static getBorderTop(properties: {
        [k: string]: ICSSStyleDeclarationPropertyValue;
    }): ICSSStyleDeclarationPropertyValue | null;
    /**
     * Returns border.
     *
     * @param properties Properties.
     * @returns Property value
     */
    static getBorderRight(properties: {
        [k: string]: ICSSStyleDeclarationPropertyValue;
    }): ICSSStyleDeclarationPropertyValue | null;
    /**
     * Returns border.
     *
     * @param properties Properties.
     * @returns Property value
     */
    static getBorderBottom(properties: {
        [k: string]: ICSSStyleDeclarationPropertyValue;
    }): ICSSStyleDeclarationPropertyValue | null;
    /**
     * Returns border.
     *
     * @param properties Properties.
     * @returns Property value
     */
    static getBorderLeft(properties: {
        [k: string]: ICSSStyleDeclarationPropertyValue;
    }): ICSSStyleDeclarationPropertyValue | null;
    /**
     * Returns border.
     *
     * @param properties Properties.
     * @returns Property value
     */
    static getBorderColor(properties: {
        [k: string]: ICSSStyleDeclarationPropertyValue;
    }): ICSSStyleDeclarationPropertyValue | null;
    /**
     * Returns border.
     *
     * @param properties Properties.
     * @returns Property value
     */
    static getBorderWidth(properties: {
        [k: string]: ICSSStyleDeclarationPropertyValue;
    }): ICSSStyleDeclarationPropertyValue | null;
    /**
     * Returns border.
     *
     * @param properties Properties.
     * @returns Property value
     */
    static getBorderStyle(properties: {
        [k: string]: ICSSStyleDeclarationPropertyValue;
    }): ICSSStyleDeclarationPropertyValue | null;
    /**
     * Returns border radius.
     *
     * @param properties Properties.
     * @returns Property value
     */
    static getBorderRadius(properties: {
        [k: string]: ICSSStyleDeclarationPropertyValue;
    }): ICSSStyleDeclarationPropertyValue | null;
    /**
     * Returns border image.
     *
     * @param properties Properties.
     * @returns Property value
     */
    static getBorderImage(properties: {
        [k: string]: ICSSStyleDeclarationPropertyValue;
    }): ICSSStyleDeclarationPropertyValue | null;
    /**
     * Returns background.
     *
     * @param properties Properties.
     * @returns Property value
     */
    static getBackground(properties: {
        [k: string]: ICSSStyleDeclarationPropertyValue;
    }): ICSSStyleDeclarationPropertyValue | null;
    /**
     * Returns background position.
     *
     * @param properties Properties.
     * @returns Property value
     */
    static getBackgroundPosition(properties: {
        [k: string]: ICSSStyleDeclarationPropertyValue;
    }): ICSSStyleDeclarationPropertyValue | null;
    /**
     * Returns flex.
     *
     * @param properties Properties.
     * @returns Property value
     */
    static getFlex(properties: {
        [k: string]: ICSSStyleDeclarationPropertyValue;
    }): ICSSStyleDeclarationPropertyValue | null;
    /**
     * Returns flex.
     *
     * @param properties Properties.
     * @returns Property value
     */
    static getFont(properties: {
        [k: string]: ICSSStyleDeclarationPropertyValue;
    }): ICSSStyleDeclarationPropertyValue | null;
    /**
     * Returns border.
     *
     * @param properties Properties.
     * @param position
     * @returns Property value
     */
    private static getBorderTopRightBottomLeft;
    /**
     * Returns a padding like property.
     *
     * @param properties Properties.
     * @param position
     * @param propertyNames
     * @returns Property value
     */
    private static getPaddingLikeProperty;
}
//# sourceMappingURL=CSSStyleDeclarationPropertyGetParser.d.ts.map