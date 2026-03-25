"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const DOMException_js_1 = __importDefault(require("../../exception/DOMException.cjs"));
/**
 * HTML input element value stepping.
 */
class HTMLInputElementValueStepping {
    /**
     * Steps up or down.
     *
     * @param type Type.
     * @param value Value.
     * @param direction Direction.
     * @param [increment] Increment.
     * @returns New value.
     */
    static step(type, value, direction, increment) {
        switch (type) {
            case 'number':
                return String(Number(value) + (increment !== undefined ? increment * direction : direction));
            case 'date':
            case 'month':
            case 'week':
            case 'time':
            case 'datetime-local':
            case 'range':
                // TODO: Implement support for additional types
                return null;
            default:
                throw new DOMException_js_1.default('This form element is not steppable.');
        }
    }
}
exports.default = HTMLInputElementValueStepping;
//# sourceMappingURL=HTMLInputElementValueStepping.cjs.map