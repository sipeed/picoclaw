import Event from '../Event.js';
import IMediaQueryListInit from './IMediaQueryListInit.js';
/**
 *
 */
export default class MediaQueryListEvent extends Event {
    readonly matches: boolean;
    readonly media: string;
    /**
     * Constructor.
     *
     * @param type Event type.
     * @param [eventInit] Event init.
     */
    constructor(type: string, eventInit?: IMediaQueryListInit | null);
}
//# sourceMappingURL=MediaQueryListEvent.d.ts.map