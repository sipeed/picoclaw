import Event from '../Event.cjs';
import IProgressEventInit from './IProgressEventInit.cjs';
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