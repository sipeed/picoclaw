import DOMException from '../../exception/DOMException.js';
/**
 * HTML input element value stepping.
 */
export default class HTMLInputElementValueStepping {
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
                throw new DOMException('This form element is not steppable.');
        }
    }
}
//# sourceMappingURL=HTMLInputElementValueStepping.js.map