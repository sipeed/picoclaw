import DataTransfer from '../DataTransfer.cjs';
import Event from '../Event.cjs';
import IClipboardEventInit from './IClipboardEventInit.cjs';
/**
 *
 */
export default class ClipboardEvent extends Event {
    clipboardData: DataTransfer | null;
    /**
     * Constructor.
     *
     * @param type Event type.
     * @param [eventInit] Event init.
     */
    constructor(type: string, eventInit?: IClipboardEventInit | null);
}
//# sourceMappingURL=ClipboardEvent.d.ts.map