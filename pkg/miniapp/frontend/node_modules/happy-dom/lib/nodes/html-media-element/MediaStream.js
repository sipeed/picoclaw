import * as PropertySymbol from '../../PropertySymbol.js';
import Crypto from 'crypto';
import EventTarget from '../../event/EventTarget.js';
import MediaStreamTrackEvent from '../../event/events/MediaStreamTrackEvent.js';
/**
 * MediaStream.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/MediaStream
 */
export default class MediaStream extends EventTarget {
    // Public properties
    active = true;
    id = Crypto.randomUUID();
    // Events
    onaddtrack = null;
    onremovetrack = null;
    // Internal properties
    [PropertySymbol.tracks] = [];
    /**
     * Constructor.
     *
     * @param [streamOrTracks] Stream or tracks.
     */
    constructor(streamOrTracks) {
        super();
        if (!this[PropertySymbol.window]) {
            throw new TypeError(`Failed to construct '${this.constructor.name}': '${this.constructor.name}' was constructed outside a Window context.`);
        }
        if (streamOrTracks !== undefined) {
            this[PropertySymbol.tracks] =
                streamOrTracks instanceof MediaStream
                    ? streamOrTracks[PropertySymbol.tracks].slice()
                    : streamOrTracks;
        }
    }
    /**
     * Adds a track.
     *
     * @param track Track.
     */
    addTrack(track) {
        if (this[PropertySymbol.tracks].includes(track)) {
            return;
        }
        this[PropertySymbol.tracks].push(track);
        this.dispatchEvent(new MediaStreamTrackEvent('addtrack', { track }));
    }
    /**
     * Returns a clone.
     *
     * @returns Clone.
     */
    clone() {
        return new this.constructor(this);
    }
    /**
     * Returns audio tracks.
     *
     * @returns Audio tracks.
     */
    getAudioTracks() {
        return this[PropertySymbol.tracks].filter((track) => track.kind === 'audio');
    }
    /**
     * Returns track by id.
     *
     * @param id Id.
     * @returns Track.
     */
    getTrackById(id) {
        for (const track of this[PropertySymbol.tracks]) {
            if (track.id === id) {
                return track;
            }
        }
        return null;
    }
    /**
     * Returns video tracks.
     *
     * @returns Video tracks.
     */
    getVideoTracks() {
        return this[PropertySymbol.tracks].filter((track) => track.kind === 'video');
    }
    /**
     * Removes a track.
     *
     * @param track Track.
     */
    removeTrack(track) {
        const index = this[PropertySymbol.tracks].indexOf(track);
        if (index === -1) {
            return;
        }
        this[PropertySymbol.tracks].splice(index, 1);
        this.dispatchEvent(new MediaStreamTrackEvent('removetrack', { track }));
    }
}
//# sourceMappingURL=MediaStream.js.map