import CSSEscaper from './utilities/CSSEscaper.js';
import CSSUnitValue from './CSSUnitValue.js';
import CSSUnits from './CSSUnits.js';
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
    constructor() {
        for (const unit of CSSUnits) {
            this[unit] = (value) => new CSSUnitValue(value, unit);
        }
    }
    /**
     * Returns a Boolean indicating if the pair property-value, or the condition, given in parameter is supported.
     *
     * TODO: Always returns "true" for now, but it should probably be improved in the future.
     *
     * @param _condition Property name or condition.
     * @param [_value] Value when using property name.
     * @returns "true" if supported.
     */
    supports(_condition, _value) {
        return true;
    }
    /**
     * Escapes a value.
     *
     * @param value Value to escape.
     * @returns Escaped string.
     */
    escape(value) {
        return CSSEscaper.escape(value);
    }
}
//# sourceMappingURL=CSS.js.map