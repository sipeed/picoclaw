/**
 * The CSS interface holds useful CSS-related methods.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/CSS.
 */
export default class CSS {
    /**
     * Constructor.
     */
    constructor();
    /**
     * Returns a Boolean indicating if the pair property-value, or the condition, given in parameter is supported.
     *
     * TODO: Always returns "true" for now, but it should probably be improved in the future.
     *
     * @param _condition Property name or condition.
     * @param [_value] Value when using property name.
     * @returns "true" if supported.
     */
    supports(_condition: string, _value?: string): boolean;
    /**
     * Escapes a value.
     *
     * @param value Value to escape.
     * @returns Escaped string.
     */
    escape(value: string): string;
}
//# sourceMappingURL=CSS.d.ts.map