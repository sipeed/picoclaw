import * as PropertySymbol from '../../PropertySymbol.js';
import EventTarget from '../../event/EventTarget.js';
import MediaStreamTrackEvent from '../../event/events/MediaStreamTrackEvent.js';
import MediaStreamTrack from './MediaStreamTrack.js';
/**
 * MediaStream.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/MediaStream
 */
export default class MediaStream extends EventTarget {
    active: boolean;
    id: string;
    onaddtrack: ((event: MediaStreamTrackEvent) => void) | null;
    onremovetrack: ((event: MediaStreamTrackEvent) => void) | null;
    [PropertySymbol.tracks]: MediaStreamTrack[];
    /**
     * Constructor.
     *
     * @param [streamOrTracks] Stream or tracks.
     */
    constructor(streamOrTracks?: MediaStream | MediaStreamTrack[]);
    /**
     * Adds a track.
     *
     * @param track Track.
     */
    addTrack(track: MediaStreamTrack): void;
    /**
     * Returns a clone.
     *
     * @returns Clone.
     */
    clone(): MediaStream;
    /**
     * Returns audio tracks.
     *
     * @returns Audio tracks.
     */
    getAudioTracks(): MediaStreamTrack[];
    /**
     * Returns track by id.
     *
     * @param id Id.
     * @returns Track.
     */
    getTrackById(id: string): MediaStreamTrack | null;
    /**
     * Returns video tracks.
     *
     * @returns Video tracks.
     */
    getVideoTracks(): MediaStreamTrack[];
    /**
     * Removes a track.
     *
     * @param track Track.
     */
    removeTrack(track: MediaStreamTrack): void;
}
//# sourceMappingURL=MediaStream.d.ts.map