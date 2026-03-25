/**
 * Style declaration value parser.
 */
export default class CSSStyleDeclarationValueParser {
    /**
     * Returns length.
     *
     * @param value Value.
     * @returns Parsed value.
     */
    static getLength(value: string): string;
    /**
     * Returns percentance.
     *
     * @param value Value.
     * @returns Parsed value.
     */
    static getPercentage(value: string): string;
    /**
     * Returns degree.
     *
     * @param value Value.
     * @returns Parsed value.
     */
    static getDegree(value: string): string;
    /**
     * Returns calc.
     *
     * @param value Value.
     * @returns Parsed value.
     */
    static getCalc(value: string): string;
    /**
     * Returns fit content.
     *
     * @param value Value.
     * @returns Parsed value.
     */
    static getFitContent(value: string): string;
    /**
     * Returns measurement.
     *
     * @param value Value.
     * @returns Parsed value.
     */
    static getMeasurement(value: string): string;
    /**
     * Returns measurement or auto, min-content, max-content or fit-content.
     *
     * @param value Value.
     * @returns Parsed value.
     */
    static getContentMeasurement(value: string): string;
    /**
     * Returns measurement or auto, min-content, max-content or fit-content.
     *
     * @param value Value.
     * @returns Parsed value.
     */
    static getAutoMeasurement(value: string): string;
    /**
     * Returns integer.
     *
     * @param value Value.
     * @returns Parsed value.
     */
    static getInteger(value: string): string;
    /**
     * Returns float.
     *
     * @param value Value.
     * @returns Parsed value.
     */
    static getFloat(value: string): string;
    /**
     * Returns gradient.
     *
     * @param value Value.
     * @returns Parsed value.
     */
    static getGradient(value: string): string;
    /**
     * Returns color.
     *
     * @param value Value.
     * @returns Parsed value.
     */
    static getColor(value: string): string;
    /**
     * Returns URL.
     *
     * Based on:
     * https://github.com/jsdom/cssstyle/blob/master/lib/parsers.js#L222
     *
     * @param value Value.
     * @returns Parsed value.
     */
    static getURL(value: string): string;
    /**
     * Returns global initial value.
     *
     * @param value Value.
     * @returns Parsed value.
     */
    static getInitial(value: string): string;
    /**
     * Returns CSS variable.
     *
     * @param value Value.
     * @returns Parsed value.
     */
    static getVariable(value: string): string;
    /**
     * Returns global.
     *
     * @param value Value.
     * @returns Parsed value.
     */
    static getGlobal(value: string): string;
    /**
     * Returns global, unless it is not set to 'initial' as it is sometimes treated different.
     *
     * @param value Value.
     * @returns Parsed value.
     */
    static getGlobalExceptInitial(value: string): string;
}
//# sourceMappingURL=CSSStyleDeclarationValueParser.d.ts.map