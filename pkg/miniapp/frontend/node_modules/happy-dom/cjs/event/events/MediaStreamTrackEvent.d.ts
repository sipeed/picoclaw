import MediaStreamTrack from '../../nodes/html-media-element/MediaStreamTrack.cjs';
import Event from '../Event.cjs';
import IMediaQueryListEventInit from './IMediaQueryListEventInit.cjs';
/**
 * Media Stream Track Event.
 */
export default class MediaStreamTrackEvent extends Event {
    readonly track: MediaStreamTrack | null;
    /**
     * Constructor.
     *
     * @param type Event type.
     * @param [eventInit] Event init.
     */
    constructor(type: string, eventInit?: IMediaQueryListEventInit | null);
}
//# sourceMappingURL=MediaStreamTrackEvent.d.ts.map