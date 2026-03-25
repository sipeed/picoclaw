import DataTransfer from '../DataTransfer.cjs';
import UIEvent from '../UIEvent.cjs';
import IInputEventInit from './IInputEventInit.cjs';
/**
 *
 */
export default class InputEvent extends UIEvent {
    readonly data: string;
    readonly dataTransfer: DataTransfer;
    readonly inputType: string;
    readonly isComposing: boolean;
    /**
     * Constructor.
     *
     * @param type Event type.
     * @param [eventInit] Event init.
     */
    constructor(type: string, eventInit?: IInputEventInit | null);
}
//# sourceMappingURL=InputEvent.d.ts.map