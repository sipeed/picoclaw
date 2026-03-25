import HTMLInputElement from './HTMLInputElement.js';
/**
 * HTML input element value sanitizer.
 */
export default class HTMLInputElementValueSanitizer {
    /**
     * Sanitizes a value.
     *
     * @param input Input.
     * @param value Value.
     */
    static sanitize(input: HTMLInputElement, value: string): string;
    /**
     * Checks if a value is within the boundaries of min and max.
     *
     * @param value
     * @param min
     * @param max
     */
    private static checkBoundaries;
    /**
     * Parses the month component of a date string.
     *
     * @param value
     */
    private static parseMonthComponent;
    /**
     * Returns the last ISO week of a year.
     *
     * @param year
     */
    private static lastIsoWeekOfYear;
    /**
     * Sanitizes a date string.
     *
     * @param value
     */
    private static sanitizeDate;
    /**
     * Sanitizes a time string.
     *
     * @param value
     */
    private static sanitizeTime;
}
//# sourceMappingURL=HTMLInputElementValueSanitizer.d.ts.map