import UIEvent from '../UIEvent.cjs';
import IErrorEventInit from './IErrorEventInit.cjs';
/**
 *
 */
export default class ErrorEvent extends UIEvent {
    readonly message: string;
    readonly filename: string;
    readonly lineno: number;
    readonly colno: number;
    readonly error: Error | null;
    /**
     * Constructor.
     *
     * @param type Event type.
     * @param [eventInit] Event init.
     */
    constructor(type: string, eventInit?: IErrorEventInit | null);
}
//# sourceMappingURL=ErrorEvent.d.ts.map