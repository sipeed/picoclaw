"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const CSSEscaper_js_1 = __importDefault(require("./utilities/CSSEscaper.cjs"));
const CSSUnitValue_js_1 = __importDefault(require("./CSSUnitValue.cjs"));
const CSSUnits_js_1 = __importDefault(require("./CSSUnits.cjs"));
/**
 * The CSS interface holds useful CSS-related methods.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/CSS.
 */
class CSS {
    /**
     * Constructor.
     */
    constructor() {
        for (const unit of CSSUnits_js_1.default) {
            this[unit] = (value) => new CSSUnitValue_js_1.default(value, unit);
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
        return CSSEscaper_js_1.default.escape(value);
    }
}
exports.default = CSS;
//# sourceMappingURL=CSS.cjs.map