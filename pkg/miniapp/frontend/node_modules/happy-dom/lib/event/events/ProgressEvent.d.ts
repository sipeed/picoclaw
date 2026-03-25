import Event from '../Event.js';
import IProgressEventInit from './IProgressEventInit.js';
/**
 *
 */
export default class ProgressEvent extends Event {
    readonly lengthComputable: boolean;
    readonly loaded: number;
    readonly total: number;
    /**
     * Constructor.
     *
     * @param type Event type.
     * @param [eventInit] Event init.
     */
    constructor(type: string, eventInit?: IProgressEventInit | null);
}
//# sourceMappingURL=ProgressEvent.d.ts.map