import DataTransfer from '../DataTransfer.js';
import Event from '../Event.js';
import IClipboardEventInit from './IClipboardEventInit.js';
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