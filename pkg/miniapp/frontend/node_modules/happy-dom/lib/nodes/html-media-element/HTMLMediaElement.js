import Event from '../../event/Event.js';
import DOMExceptionNameEnum from '../../exception/DOMExceptionNameEnum.js';
import HTMLElement from '../html-element/HTMLElement.js';
import TimeRanges from './TimeRanges.js';
import DOMTokenList from '../../dom/DOMTokenList.js';
import MediaStream from './MediaStream.js';
import TextTrackKindEnum from './TextTrackKindEnum.js';
import * as PropertySymbol from '../../PropertySymbol.js';
/**
 * HTML Media Element.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/HTMLMediaElement
 *
 */
export default class HTMLMediaElement extends HTMLElement {
    // Events
    onabort = null;
    oncanplay = null;
    oncanplaythrough = null;
    ondurationchange = null;
    onemptied = null;
    onended = null;
    onerror = null;
    onloadeddata = null;
    onloadedmetadata = null;
    onloadstart = null;
    onpause = null;
    onplay = null;
    onplaying = null;
    onprogress = null;
    onratechange = null;
    onresize = null;
    onseeked = null;
    onseeking = null;
    onstalled = null;
    onsuspend = null;
    ontimeupdate = null;
    onvolumechange = null;
    onwaiting = null;
    // Internal Properties
    [PropertySymbol.volume] = 1;
    [PropertySymbol.paused] = true;
    [PropertySymbol.currentTime] = 0;
    [PropertySymbol.playbackRate] = 1;
    [PropertySymbol.defaultPlaybackRate] = 1;
    [PropertySymbol.muted] = false;
    [PropertySymbol.defaultMuted] = false;
    [PropertySymbol.preservesPitch] = true;
    [PropertySymbol.buffered] = new TimeRanges(PropertySymbol.illegalConstructor);
    [PropertySymbol.duration] = NaN;
    [PropertySymbol.error] = null;
    [PropertySymbol.ended] = false;
    [PropertySymbol.networkState] = 0;
    [PropertySymbol.readyState] = 0;
    [PropertySymbol.seeking] = false;
    [PropertySymbol.seekable] = new TimeRanges(PropertySymbol.illegalConstructor);
    [PropertySymbol.sinkId] = '';
    [PropertySymbol.played] = new TimeRanges(PropertySymbol.illegalConstructor);
    [PropertySymbol.remote] = new this[PropertySymbol.window].RemotePlayback();
    [PropertySymbol.controlsList] = null;
    [PropertySymbol.mediaKeys] = null;
    [PropertySymbol.srcObject] = null;
    [PropertySymbol.textTracks] = [];
    /**
     * Returns buffered.
     *
     * @returns Buffered.
     */
    get buffered() {
        return this[PropertySymbol.buffered];
    }
    /**
     * Returns duration.
     *
     * @returns Duration.
     */
    get duration() {
        return this[PropertySymbol.duration];
    }
    /**
     * Returns error.
     *
     * @returns Error.
     */
    get error() {
        return this[PropertySymbol.error];
    }
    /**
     * Returns ended.
     *
     * @returns Ended.
     */
    get ended() {
        return this[PropertySymbol.ended];
    }
    /**
     * Returns networkState.
     *
     * @returns NetworkState.
     */
    get networkState() {
        return this[PropertySymbol.networkState];
    }
    /**
     * Returns readyState.
     *
     * @returns ReadyState.
     */
    get readyState() {
        return this[PropertySymbol.readyState];
    }
    /**
     * Return a RemotePlayback object instance associated with the media element.
     *
     * @returns RemotePlayback.
     */
    get remote() {
        return this[PropertySymbol.remote];
    }
    /**
     * Returns seeking.
     *
     * @returns Seeking.
     */
    get seeking() {
        return this[PropertySymbol.seeking];
    }
    /**
     * Returns seekable.
     *
     * @returns Seekable.
     */
    get seekable() {
        return this[PropertySymbol.seekable];
    }
    /**
     * Returns sinkId.
     *
     * @returns SinkId.
     */
    get sinkId() {
        return this[PropertySymbol.sinkId];
    }
    /**
     * Returns played.
     *
     * @returns Played.
     */
    get played() {
        return this[PropertySymbol.played];
    }
    /**
     * Returns autoplay.
     *
     * @returns Autoplay.
     */
    get autoplay() {
        return this.getAttribute('autoplay') !== null;
    }
    /**
     * Sets autoplay.
     *
     * @param autoplay Autoplay.
     */
    set autoplay(autoplay) {
        if (!autoplay) {
            this.removeAttribute('autoplay');
        }
        else {
            this.setAttribute('autoplay', '');
        }
    }
    /**
     * Returns controls.
     *
     * @returns Controls.
     */
    get controls() {
        return this.getAttribute('controls') !== null;
    }
    /**
     * Sets controls.
     *
     * @param controls Controls.
     */
    set controls(controls) {
        if (!controls) {
            this.removeAttribute('controls');
        }
        else {
            this.setAttribute('controls', '');
        }
    }
    /**
     * Returns loop.
     *
     * @returns Loop.
     */
    get loop() {
        return this.getAttribute('loop') !== null;
    }
    /**
     * Sets loop.
     *
     * @param loop Loop.
     */
    set loop(loop) {
        if (!loop) {
            this.removeAttribute('loop');
        }
        else {
            this.setAttribute('loop', '');
        }
    }
    /**
     * Returns preload.
     *
     * @returns preload.
     */
    get preload() {
        return this.getAttribute('preload') || 'auto';
    }
    /**
     * Sets preload.
     *
     * @param preload preload.
     */
    set preload(preload) {
        this.setAttribute('preload', preload);
    }
    /**
     * Returns src.
     *
     * @returns Src.
     */
    get src() {
        if (!this.hasAttribute('src')) {
            return '';
        }
        try {
            return new URL(this.getAttribute('src'), this[PropertySymbol.ownerDocument].location.href)
                .href;
        }
        catch (e) {
            return this.getAttribute('src');
        }
    }
    /**
     * Sets src.
     *
     * @param src Src.
     */
    set src(src) {
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
    get controlsList() {
        if (this[PropertySymbol.controlsList] === null) {
            this[PropertySymbol.controlsList] = new DOMTokenList(PropertySymbol.illegalConstructor, this, 'controlslist');
        }
        return this[PropertySymbol.controlsList];
    }
    /**
     * Returns mediaKeys.
     *
     * @returns MediaKeys.
     */
    get mediaKeys() {
        return this[PropertySymbol.mediaKeys];
    }
    /**
     * Returns muted.
     *
     * @returns Muted.
     */
    get muted() {
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
    set muted(muted) {
        this[PropertySymbol.muted] = !!muted;
        if (!muted && !this[PropertySymbol.defaultMuted]) {
            this.removeAttribute('muted');
        }
        else {
            this.setAttribute('muted', '');
        }
    }
    /**
     * Returns defaultMuted.
     *
     * @returns DefaultMuted.
     */
    get defaultMuted() {
        return this[PropertySymbol.defaultMuted];
    }
    /**
     * Sets defaultMuted.
     *
     * @param defaultMuted DefaultMuted.
     */
    set defaultMuted(defaultMuted) {
        this[PropertySymbol.defaultMuted] = !!defaultMuted;
        if (!this[PropertySymbol.defaultMuted] && !this[PropertySymbol.muted]) {
            this.removeAttribute('muted');
        }
        else {
            this.setAttribute('muted', '');
        }
    }
    /**
     * Returns disableRemotePlayback.
     *
     * @returns DisableRemotePlayback.
     */
    get disableRemotePlayback() {
        return this.getAttribute('disableremoteplayback') !== null;
    }
    /**
     * Sets disableRemotePlayback.
     *
     * @param disableRemotePlayback DisableRemotePlayback.
     */
    set disableRemotePlayback(disableRemotePlayback) {
        if (!disableRemotePlayback) {
            this.removeAttribute('disableremoteplayback');
        }
        else {
            this.setAttribute('disableremoteplayback', '');
        }
    }
    /**
     * A MediaStream representing the media to play or that has played in the current HTMLMediaElement, or null if not assigned.
     *
     * @returns MediaStream.
     */
    get srcObject() {
        return this[PropertySymbol.srcObject];
    }
    /**
     * Sets src object.
     *
     * @param srcObject SrcObject.
     */
    set srcObject(srcObject) {
        if (srcObject !== null && !(srcObject instanceof MediaStream)) {
            throw new this[PropertySymbol.window].TypeError(`Failed to set the 'srcObject' property on 'HTMLMediaElement': The provided value is not of type 'MediaStream'.`);
        }
        this[PropertySymbol.srcObject] = srcObject;
    }
    /**
     * Returns text track list.
     *
     * @returns Text track list.
     */
    get textTracks() {
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
    get currentSrc() {
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
    get volume() {
        return this[PropertySymbol.volume];
    }
    /**
     * Sets volume.
     *
     * @param volume Volume.
     */
    set volume(volume) {
        const parsedVolume = Number(volume);
        if (isNaN(parsedVolume)) {
            throw new this[PropertySymbol.window].TypeError(`Failed to set the 'volume' property on 'HTMLMediaElement': The provided double value is non-finite.`);
        }
        if (parsedVolume < 0 || parsedVolume > 1) {
            throw new this[PropertySymbol.window].DOMException(`Failed to set the 'volume' property on 'HTMLMediaElement': The volume provided (${parsedVolume}) is outside the range [0, 1].`, DOMExceptionNameEnum.indexSizeError);
        }
        // TODO: volumechange event https://developer.mozilla.org/en-US/docs/Web/API/HTMLMediaElement/volumechange_event
        this[PropertySymbol.volume] = parsedVolume;
    }
    /**
     * Returns crossOrigin.
     *
     * @returns CrossOrigin.
     */
    get crossOrigin() {
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
    set crossOrigin(crossOrigin) {
        this.setAttribute('crossorigin', crossOrigin);
    }
    /**
     * Returns currentTime.
     *
     * @returns CurrentTime.
     */
    get currentTime() {
        return this[PropertySymbol.currentTime];
    }
    /**
     * Sets currentTime.
     *
     * @param currentTime CurrentTime.
     */
    set currentTime(currentTime) {
        const parsedCurrentTime = Number(currentTime);
        if (isNaN(parsedCurrentTime)) {
            throw new this[PropertySymbol.window].TypeError(`Failed to set the 'currentTime' property on 'HTMLMediaElement': The provided double value is non-finite.`);
        }
        this[PropertySymbol.currentTime] = parsedCurrentTime;
    }
    /**
     * Returns playbackRate.
     *
     * @returns PlaybackRate.
     */
    get playbackRate() {
        return this[PropertySymbol.playbackRate];
    }
    /**
     * Sets playbackRate.
     *
     * @param playbackRate PlaybackRate.
     */
    set playbackRate(playbackRate) {
        const parsedPlaybackRate = Number(playbackRate);
        if (isNaN(parsedPlaybackRate)) {
            throw new this[PropertySymbol.window].TypeError(`Failed to set the 'playbackRate' property on 'HTMLMediaElement': The provided double value is non-finite.`);
        }
        this[PropertySymbol.playbackRate] = parsedPlaybackRate;
    }
    /**
     * Returns defaultPlaybackRate.
     *
     * @returns DefaultPlaybackRate.
     */
    get defaultPlaybackRate() {
        return this[PropertySymbol.defaultPlaybackRate];
    }
    /**
     * Sets defaultPlaybackRate.
     *
     * @param defaultPlaybackRate DefaultPlaybackRate.
     */
    set defaultPlaybackRate(defaultPlaybackRate) {
        const parsedDefaultPlaybackRate = Number(defaultPlaybackRate);
        if (isNaN(parsedDefaultPlaybackRate)) {
            throw new this[PropertySymbol.window].TypeError(`Failed to set the 'defaultPlaybackRate' property on 'HTMLMediaElement': The provided double value is non-finite.`);
        }
        this[PropertySymbol.defaultPlaybackRate] = parsedDefaultPlaybackRate;
    }
    /**
     * Returns preservesPitch.
     *
     * @returns PlaybackRate.
     */
    get preservesPitch() {
        return this[PropertySymbol.preservesPitch];
    }
    /**
     * Sets preservesPitch.
     *
     * @param preservesPitch PreservesPitch.
     */
    set preservesPitch(preservesPitch) {
        this[PropertySymbol.preservesPitch] = Boolean(preservesPitch);
    }
    /**
     * Returns paused.
     *
     * @returns Paused.
     */
    get paused() {
        return this[PropertySymbol.paused];
    }
    /**
     * Adds a new text track to the media element.
     *
     * @param kind The kind of text track.
     * @param label The label of the text track.
     * @param language The language of the text track data.
     */
    addTextTrack(kind, label, language) {
        const window = this[PropertySymbol.window];
        if (arguments.length === 0) {
            throw new window.TypeError(`Failed to execute 'addTextTrack' on 'HTMLMediaElement': 1 argument required, but only 0 present.`);
        }
        if (!TextTrackKindEnum[kind]) {
            throw new window.TypeError(`Failed to execute 'addTextTrack' on 'HTMLMediaElement': The provided value '${kind}' is not a valid enum value of type TextTrackKind.`);
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
    pause() {
        if (this[PropertySymbol.paused]) {
            return;
        }
        this[PropertySymbol.paused] = true;
        this.dispatchEvent(new Event('pause', { bubbles: false, cancelable: false }));
    }
    /**
     * Start playing media.
     */
    async play() {
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
    canPlayType(_type) {
        // TODO: Implement this method
        return '';
    }
    /**
     * Quickly seeks the media to the new time with precision tradeoff.
     *
     * @param _time Time.
     */
    fastSeek(_time) {
        // TODO: Implement this method
    }
    /**
     * Load media.
     */
    load() {
        this.dispatchEvent(new Event('emptied', { bubbles: false, cancelable: false }));
    }
    /**
     * Sets media keys.
     *
     * @param mediaKeys MediaKeys.
     * @returns Promise.
     */
    async setMediaKeys(mediaKeys) {
        this[PropertySymbol.mediaKeys] = mediaKeys;
    }
    /**
     * Sets sink id.
     *
     * @param sinkId SinkId.
     * @returns Promise.
     */
    async setSinkId(sinkId) {
        this[PropertySymbol.sinkId] = sinkId;
    }
    /**
     * Returns MediaStream, captures a stream of the media content.
     *
     * @returns MediaStream.
     */
    captureStream() {
        return new this[PropertySymbol.window].MediaStream();
    }
    /**
     * @override
     */
    [PropertySymbol.cloneNode](deep = false) {
        return super[PropertySymbol.cloneNode](deep);
    }
}
//# sourceMappingURL=HTMLMediaElement.js.map