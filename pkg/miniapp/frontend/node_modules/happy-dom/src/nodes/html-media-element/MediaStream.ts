import * as PropertySymbol from '../../PropertySymbol.js';
import Crypto from 'crypto';
import EventTarget from '../../event/EventTarget.js';
import MediaStreamTrackEvent from '../../event/events/MediaStreamTrackEvent.js';
import MediaStreamTrack from './MediaStreamTrack.js';

/**
 * MediaStream.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/MediaStream
 */
export default class MediaStream extends EventTarget {
	// Public properties
	public active = true;
	public id: string = Crypto.randomUUID();

	// Events
	public onaddtrack: ((event: MediaStreamTrackEvent) => void) | null = null;
	public onremovetrack: ((event: MediaStreamTrackEvent) => void) | null = null;

	// Internal properties
	public [PropertySymbol.tracks]: MediaStreamTrack[] = [];

	/**
	 * Constructor.
	 *
	 * @param [streamOrTracks] Stream or tracks.
	 */
	constructor(streamOrTracks?: MediaStream | MediaStreamTrack[]) {
		super();

		if (!this[PropertySymbol.window]) {
			throw new TypeError(
				`Failed to construct '${this.constructor.name}': '${this.constructor.name}' was constructed outside a Window context.`
			);
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
	public addTrack(track: MediaStreamTrack): void {
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
	public clone(): MediaStream {
		return new (<typeof MediaStream>this.constructor)(this);
	}

	/**
	 * Returns audio tracks.
	 *
	 * @returns Audio tracks.
	 */
	public getAudioTracks(): MediaStreamTrack[] {
		return this[PropertySymbol.tracks].filter((track) => track.kind === 'audio');
	}

	/**
	 * Returns track by id.
	 *
	 * @param id Id.
	 * @returns Track.
	 */
	public getTrackById(id: string): MediaStreamTrack | null {
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
	public getVideoTracks(): MediaStreamTrack[] {
		return this[PropertySymbol.tracks].filter((track) => track.kind === 'video');
	}

	/**
	 * Removes a track.
	 *
	 * @param track Track.
	 */
	public removeTrack(track: MediaStreamTrack): void {
		const index = this[PropertySymbol.tracks].indexOf(track);
		if (index === -1) {
			return;
		}
		this[PropertySymbol.tracks].splice(index, 1);
		this.dispatchEvent(new MediaStreamTrackEvent('removetrack', { track }));
	}
}
