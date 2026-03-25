import Event from '../Event.js';
import IHashChangeEventInit from './IHashChangeEventInit.js';
/**
 * Hash change event.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/HashChangeEvent
 */
export default class HashChangeEvent extends Event {
    readonly newURL: string;
    readonly oldURL: string;
    /**
     * Constructor.
     *
     * @param type Event type.
     * @param [eventInit] Event init.
     */
    constructor(type: string, eventInit?: IHashChangeEventInit | null);
}
//# sourceMappingURL=HashChangeEvent.d.ts.map