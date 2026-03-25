import ErrorEvent from '../../event/events/ErrorEvent.js';
import Event from '../../event/Event.js';
import DOMExceptionNameEnum from '../../exception/DOMExceptionNameEnum.js';
import HTMLElement from '../html-element/HTMLElement.js';
import TimeRanges from './TimeRanges.js';
import DOMTokenList from '../../dom/DOMTokenList.js';
import RemotePlayback from './RemotePlayback.js';
import MediaStream from './MediaStream.js';
import TextTrackList from './TextTrackList.js';
import TextTrack from './TextTrack.js';
import TextTrackKindEnum from './TextTrackKindEnum.js';
import * as PropertySymbol from '../../PropertySymbol.js';

interface IMediaError {
	code: number;
	message: string;
}

/**
 * HTML Media Element.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/HTMLMediaElement
 *
 */
export default class HTMLMediaElement extends HTMLElement {
	// Public properties
	public declare cloneNode: (deep?: boolean) => HTMLMediaElement;

	// Events
	public onabort: ((event: Event) => void) | null = null;
	public oncanplay: ((event: Event) => void) | null = null;
	public oncanplaythrough: ((event: Event) => void) | null = null;
	public ondurationchange: ((event: Event) => void) | null = null;
	public onemptied: ((event: Event) => void) | null = null;
	public onended: ((event: Event) => void) | null = null;
	public onerror: ((event: ErrorEvent) => void) | null = null;
	public onloadeddata: ((event: Event) => void) | null = null;
	public onloadedmetadata: ((event: Event) => void) | null = null;
	public onloadstart: ((event: Event) => void) | null = null;
	public onpause: ((event: Event) => void) | null = null;
	public onplay: ((event: Event) => void) | null = null;
	public onplaying: ((event: Event) => void) | null = null;
	public onprogress: ((event: Event) => void) | null = null;
	public onratechange: ((event: Event) => void) | null = null;
	public onresize: ((event: Event) => void) | null = null;
	public onseeked: ((event: Event) => void) | null = null;
	public onseeking: ((event: Event) => void) | null = null;
	public onstalled: ((event: Event) => void) | null = null;
	public onsuspend: ((event: Event) => void) | null = null;
	public ontimeupdate: ((event: Event) => void) | null = null;
	public onvolumechange: ((event: Event) => void) | null = null;
	public onwaiting: ((event: Event) => void) | null = null;

	// Internal Properties
	public [PropertySymbol.volume] = 1;
	public [PropertySymbol.paused] = true;
	public [PropertySymbol.currentTime] = 0;
	public [PropertySymbol.playbackRate] = 1;
	public [PropertySymbol.defaultPlaybackRate] = 1;
	public [PropertySymbol.muted] = false;
	public [PropertySymbol.defaultMuted] = false;
	public [PropertySymbol.preservesPitch] = true;
	public [PropertySymbol.buffered] = new TimeRanges(PropertySymbol.illegalConstructor);
	public [PropertySymbol.duration] = NaN;
	public [PropertySymbol.error]: IMediaError = null;
	public [PropertySymbol.ended] = false;
	public [PropertySymbol.networkState] = 0;
	public [PropertySymbol.readyState] = 0;
	public [PropertySymbol.seeking] = false;
	public [PropertySymbol.seekable] = new TimeRanges(PropertySymbol.illegalConstructor);
	public [PropertySymbol.sinkId]: string = '';
	public [PropertySymbol.played] = new TimeRanges(PropertySymbol.illegalConstructor);
	public [PropertySymbol.remote] = new this[PropertySymbol.window].RemotePlayback();
	public [PropertySymbol.controlsList]: DOMTokenList | null = null;
	public [PropertySymbol.mediaKeys]: object | null = null;
	public [PropertySymbol.srcObject]: MediaStream | null = null;
	public [PropertySymbol.textTracks]: TextTrack[] = [];

	/**
	 * Returns buffered.
	 *
	 * @returns Buffered.
	 */
	public get buffered(): TimeRanges {
		return this[PropertySymbol.buffered];
	}

	/**
	 * Returns duration.
	 *
	 * @returns Duration.
	 */
	public get duration(): number {
		return this[PropertySymbol.duration];
	}

	/**
	 * Returns error.
	 *
	 * @returns Error.
	 */
	public get error(): IMediaError {
		return this[PropertySymbol.error];
	}

	/**
	 * Returns ended.
	 *
	 * @returns Ended.
	 */
	public get ended(): boolean {
		return this[PropertySymbol.ended];
	}

	/**
	 * Returns networkState.
	 *
	 * @returns NetworkState.
	 */
	public get networkState(): number {
		return this[PropertySymbol.networkState];
	}

	/**
	 * Returns readyState.
	 *
	 * @returns ReadyState.
	 */
	public get readyState(): number {
		return this[PropertySymbol.readyState];
	}

	/**
	 * Return a RemotePlayback object instance associated with the media element.
	 *
	 * @returns RemotePlayback.
	 */
	public get remote(): RemotePlayback {
		return this[PropertySymbol.remote];
	}

	/**
	 * Returns seeking.
	 *
	 * @returns Seeking.
	 */
	public get seeking(): boolean {
		return this[PropertySymbol.seeking];
	}

	/**
	 * Returns seekable.
	 *
	 * @returns Seekable.
	 */
	public get seekable(): TimeRanges {
		return this[PropertySymbol.seekable];
	}

	/**
	 * Returns sinkId.
	 *
	 * @returns SinkId.
	 */
	public get sinkId(): string {
		return this[PropertySymbol.sinkId];
	}

	/**
	 * Returns played.
	 *
	 * @returns Played.
	 */
	public get played(): TimeRanges {
		return this[PropertySymbol.played];
	}

	/**
	 * Returns autoplay.
	 *
	 * @returns Autoplay.
	 */
	public get autoplay(): boolean {
		return this.getAttribute('autoplay') !== null;
	}

	/**
	 * Sets autoplay.
	 *
	 * @param autoplay Autoplay.
	 */
	public set autoplay(autoplay: boolean) {
		if (!autoplay) {
			this.removeAttribute('autoplay');
		} else {
			this.setAttribute('autoplay', '');
		}
	}

	/**
	 * Returns controls.
	 *
	 * @returns Controls.
	 */
	public get controls(): boolean {
		return this.getAttribute('controls') !== null;
	}

	/**
	 * Sets controls.
	 *
	 * @param controls Controls.
	 */
	public set controls(controls: boolean) {
		if (!controls) {
			this.removeAttribute('controls');
		} else {
			this.setAttribute('controls', '');
		}
	}

	/**
	 * Returns loop.
	 *
	 * @returns Loop.
	 */
	public get loop(): boolean {
		return this.getAttribute('loop') !== null;
	}

	/**
	 * Sets loop.
	 *
	 * @param loop Loop.
	 */
	public set loop(loop: boolean) {
		if (!loop) {
			this.removeAttribute('loop');
		} else {
			this.setAttribute('loop', '');
		}
	}

	/**
	 * Returns preload.
	 *
	 * @returns preload.
	 */
	public get preload(): string {
		return this.getAttribute('preload') || 'auto';
	}

	/**
	 * Sets preload.
	 *
	 * @param preload preload.
	 */
	public set preload(preload: string) {
		this.setAttribute('preload', preload);
	}

	/**
	 * Returns src.
	 *
	 * @returns Src.
	 */
	public get src(): string {
		if (!this.hasAttribute('src')) {
			return '';
		}

		try {
			return new URL(this.getAttribute('src'), this[PropertySymbol.ownerDocument].location.href)
				.href;
		} catch (e) {
			return this.getAttribute('src');
		}
	}

	/**
	 * Sets src.
	 *
	 * @param src Src.
	 */
	public set src(src: string) {
		this.setAttribute('src', src);

		if (Boolean(src)) {
			this.dispatchEvent(new Event('canplay', { bubbles: false, cancelable: false }));
			this.dispatchEvent(new Event('durationchange', { bubbles: false, cancelable: false }));
		}
	}

	/**
	 * Returns controlsList.
	 *
	 * @returns ControlsList.
	 */
	public get controlsList(): DOMTokenList {
		if (this[PropertySymbol.controlsList] === null) {
			this[PropertySymbol.controlsList] = new DOMTokenList(
				PropertySymbol.illegalConstructor,
				this,
				'controlslist'
			);
		}

		return this[PropertySymbol.controlsList];
	}

	/**
	 * Returns mediaKeys.
	 *
	 * @returns MediaKeys.
	 */
	public get mediaKeys(): object | null {
		return this[PropertySymbol.mediaKeys];
	}

	/**
	 * Returns muted.
	 *
	 * @returns Muted.
	 */
	public get muted(): boolean {
		if (this[PropertySymbol.muted]) {
			return this[PropertySymbol.muted];
		}

		if (!this[PropertySymbol.defaultMuted]) {
			return this.getAttribute('muted') !== null;
		}

		return false;
	}

	/**
	 * Sets muted.
	 *
	 * @param muted Muted.
	 */
	public set muted(muted: boolean) {
		this[PropertySymbol.muted] = !!muted;
		if (!muted && !this[PropertySymbol.defaultMuted]) {
			this.removeAttribute('muted');
		} else {
			this.setAttribute('muted', '');
		}
	}

	/**
	 * Returns defaultMuted.
	 *
	 * @returns DefaultMuted.
	 */
	public get defaultMuted(): boolean {
		return this[PropertySymbol.defaultMuted];
	}

	/**
	 * Sets defaultMuted.
	 *
	 * @param defaultMuted DefaultMuted.
	 */
	public set defaultMuted(defaultMuted: boolean) {
		this[PropertySymbol.defaultMuted] = !!defaultMuted;
		if (!this[PropertySymbol.defaultMuted] && !this[PropertySymbol.muted]) {
			this.removeAttribute('muted');
		} else {
			this.setAttribute('muted', '');
		}
	}

	/**
	 * Returns disableRemotePlayback.
	 *
	 * @returns DisableRemotePlayback.
	 */
	public get disableRemotePlayback(): boolean {
		return this.getAttribute('disableremoteplayback') !== null;
	}

	/**
	 * Sets disableRemotePlayback.
	 *
	 * @param disableRemotePlayback DisableRemotePlayback.
	 */
	public set disableRemotePlayback(disableRemotePlayback: boolean) {
		if (!disableRemotePlayback) {
			this.removeAttribute('disableremoteplayback');
		} else {
			this.setAttribute('disableremoteplayback', '');
		}
	}

	/**
	 * A MediaStream representing the media to play or that has played in the current HTMLMediaElement, or null if not assigned.
	 *
	 * @returns MediaStream.
	 */
	public get srcObject(): MediaStream | null {
		return this[PropertySymbol.srcObject];
	}

	/**
	 * Sets src object.
	 *
	 * @param srcObject SrcObject.
	 */
	public set srcObject(srcObject: MediaStream | null) {
		if (srcObject !== null && !(srcObject instanceof MediaStream)) {
			throw new this[PropertySymbol.window].TypeError(
				`Failed to set the 'srcObject' property on 'HTMLMediaElement': The provided value is not of type 'MediaStream'.`
			);
		}
		this[PropertySymbol.srcObject] = srcObject;
	}

	/**
	 * Returns text track list.
	 *
	 * @returns Text track list.
	 */
	public get textTracks(): TextTrackList {
		const items = [];
		for (const track of this[PropertySymbol.textTracks]) {
			items.push(track);
		}
		for (const track of this.querySelectorAll('track')[PropertySymbol.items]) {
			items.push(track.track);
		}
		return new this[PropertySymbol.window].TextTrackList(PropertySymbol.illegalConstructor, items);
	}

	/**
	 * Returns currentSrc.
	 *
	 * @returns CurrentrSrc.
	 */
	public get currentSrc(): string {
		const src = this.src;
		if (src) {
			return src;
		}
		const sourceElement = this.querySelector('source');
		return sourceElement ? sourceElement.src : '';
	}

	/**
	 * Returns volume.
	 *
	 * @returns Volume.
	 */
	public get volume(): number {
		return this[PropertySymbol.volume];
	}

	/**
	 * Sets volume.
	 *
	 * @param volume Volume.
	 */
	public set volume(volume: number | string) {
		const parsedVolume = Number(volume);

		if (isNaN(parsedVolume)) {
			throw new this[PropertySymbol.window].TypeError(
				`Failed to set the 'volume' property on 'HTMLMediaElement': The provided double value is non-finite.`
			);
		}
		if (parsedVolume < 0 || parsedVolume > 1) {
			throw new this[PropertySymbol.window].DOMException(
				`Failed to set the 'volume' property on 'HTMLMediaElement': The volume provided (${parsedVolume}) is outside the range [0, 1].`,
				DOMExceptionNameEnum.indexSizeError
			);
		}
		// TODO: volumechange event https://developer.mozilla.org/en-US/docs/Web/API/HTMLMediaElement/volumechange_event
		this[PropertySymbol.volume] = parsedVolume;
	}

	/**
	 * Returns crossOrigin.
	 *
	 * @returns CrossOrigin.
	 */
	public get crossOrigin(): string {
		const crossOrigin = this.getAttribute('crossorigin');
		if (crossOrigin === 'use-credentials') {
			return 'use-credentials';
		}
		if (crossOrigin !== null) {
			return 'anonymous';
		}
		return null;
	}

	/**
	 * Sets crossOrigin.
	 *
	 * @param crossOrigin CrossOrigin.
	 */
	public set crossOrigin(crossOrigin: string | null) {
		this.setAttribute('crossorigin', crossOrigin);
	}

	/**
	 * Returns currentTime.
	 *
	 * @returns CurrentTime.
	 */
	public get currentTime(): number {
		return this[PropertySymbol.currentTime];
	}

	/**
	 * Sets currentTime.
	 *
	 * @param currentTime CurrentTime.
	 */
	public set currentTime(currentTime: number | string) {
		const parsedCurrentTime = Number(currentTime);
		if (isNaN(parsedCurrentTime)) {
			throw new this[PropertySymbol.window].TypeError(
				`Failed to set the 'currentTime' property on 'HTMLMediaElement': The provided double value is non-finite.`
			);
		}
		this[PropertySymbol.currentTime] = parsedCurrentTime;
	}

	/**
	 * Returns playbackRate.
	 *
	 * @returns PlaybackRate.
	 */
	public get playbackRate(): number {
		return this[PropertySymbol.playbackRate];
	}

	/**
	 * Sets playbackRate.
	 *
	 * @param playbackRate PlaybackRate.
	 */
	public set playbackRate(playbackRate: number | string) {
		const parsedPlaybackRate = Number(playbackRate);
		if (isNaN(parsedPlaybackRate)) {
			throw new this[PropertySymbol.window].TypeError(
				`Failed to set the 'playbackRate' property on 'HTMLMediaElement': The provided double value is non-finite.`
			);
		}
		this[PropertySymbol.playbackRate] = parsedPlaybackRate;
	}

	/**
	 * Returns defaultPlaybackRate.
	 *
	 * @returns DefaultPlaybackRate.
	 */
	public get defaultPlaybackRate(): number {
		return this[PropertySymbol.defaultPlaybackRate];
	}

	/**
	 * Sets defaultPlaybackRate.
	 *
	 * @param defaultPlaybackRate DefaultPlaybackRate.
	 */
	public set defaultPlaybackRate(defaultPlaybackRate: number | string) {
		const parsedDefaultPlaybackRate = Number(defaultPlaybackRate);
		if (isNaN(parsedDefaultPlaybackRate)) {
			throw new this[PropertySymbol.window].TypeError(
				`Failed to set the 'defaultPlaybackRate' property on 'HTMLMediaElement': The provided double value is non-finite.`
			);
		}
		this[PropertySymbol.defaultPlaybackRate] = parsedDefaultPlaybackRate;
	}

	/**
	 * Returns preservesPitch.
	 *
	 * @returns PlaybackRate.
	 */
	public get preservesPitch(): boolean {
		return this[PropertySymbol.preservesPitch];
	}

	/**
	 * Sets preservesPitch.
	 *
	 * @param preservesPitch PreservesPitch.
	 */
	public set preservesPitch(preservesPitch: boolean) {
		this[PropertySymbol.preservesPitch] = Boolean(preservesPitch);
	}

	/**
	 * Returns paused.
	 *
	 * @returns Paused.
	 */
	public get paused(): boolean {
		return this[PropertySymbol.paused];
	}

	/**
	 * Adds a new text track to the media element.
	 *
	 * @param kind The kind of text track.
	 * @param label The label of the text track.
	 * @param language The language of the text track data.
	 */
	public addTextTrack(kind: TextTrackKindEnum, label?: string, language?: string): TextTrack {
		const window = this[PropertySymbol.window];

		if (arguments.length === 0) {
			throw new window.TypeError(
				`Failed to execute 'addTextTrack' on 'HTMLMediaElement': 1 argument required, but only 0 present.`
			);
		}

		if (!TextTrackKindEnum[kind]) {
			throw new window.TypeError(
				`Failed to execute 'addTextTrack' on 'HTMLMediaElement': The provided value '${kind}' is not a valid enum value of type TextTrackKind.`
			);
		}

		const track = new window.TextTrack(PropertySymbol.illegalConstructor);
		track[PropertySymbol.kind] = kind;
		track[PropertySymbol.label] = label || '';
		track[PropertySymbol.language] = language || '';
		this[PropertySymbol.textTracks].push(track);
		return track;
	}

	/**
	 * Pause played media.
	 */
	public pause(): void {
		if (this[PropertySymbol.paused]) {
			return;
		}
		this[PropertySymbol.paused] = true;
		this.dispatchEvent(new Event('pause', { bubbles: false, cancelable: false }));
	}

	/**
	 * Start playing media.
	 */
	public async play(): Promise<void> {
		if (!this[PropertySymbol.paused]) {
			return;
		}
		this[PropertySymbol.paused] = false;
		this.dispatchEvent(new Event('play', { bubbles: false, cancelable: false }));
		this.dispatchEvent(new Event('playing', { bubbles: false, cancelable: false }));
	}

	/**
	 * Reports how likely it is that the current browser will be able to play media of a given MIME type.
	 *
	 * @param _type MIME type.
	 * @returns Can play type.
	 */
	public canPlayType(_type: string): string {
		// TODO: Implement this method
		return '';
	}

	/**
	 * Quickly seeks the media to the new time with precision tradeoff.
	 *
	 * @param _time Time.
	 */
	public fastSeek(_time: number): void {
		// TODO: Implement this method
	}

	/**
	 * Load media.
	 */
	public load(): void {
		this.dispatchEvent(new Event('emptied', { bubbles: false, cancelable: false }));
	}

	/**
	 * Sets media keys.
	 *
	 * @param mediaKeys MediaKeys.
	 * @returns Promise.
	 */
	public async setMediaKeys(mediaKeys: object | null): Promise<void> {
		this[PropertySymbol.mediaKeys] = mediaKeys;
	}

	/**
	 * Sets sink id.
	 *
	 * @param sinkId SinkId.
	 * @returns Promise.
	 */
	public async setSinkId(sinkId: string): Promise<void> {
		this[PropertySymbol.sinkId] = sinkId;
	}

	/**
	 * Returns MediaStream, captures a stream of the media content.
	 *
	 * @returns MediaStream.
	 */
	public captureStream(): MediaStream {
		return new this[PropertySymbol.window].MediaStream();
	}

	/**
	 * @override
	 */
	public override [PropertySymbol.cloneNode](deep = false): HTMLMediaElement {
		return <HTMLMediaElement>super[PropertySymbol.cloneNode](deep);
	}
}
