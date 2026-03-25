"use strict";
var __createBinding = (this && this.__createBinding) || (Object.create ? (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    var desc = Object.getOwnPropertyDescriptor(m, k);
    if (!desc || ("get" in desc ? !m.__esModule : desc.writable || desc.configurable)) {
      desc = { enumerable: true, get: function() { return m[k]; } };
    }
    Object.defineProperty(o, k2, desc);
}) : (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    o[k2] = m[k];
}));
var __setModuleDefault = (this && this.__setModuleDefault) || (Object.create ? (function(o, v) {
    Object.defineProperty(o, "default", { enumerable: true, value: v });
}) : function(o, v) {
    o["default"] = v;
});
var __importStar = (this && this.__importStar) || function (mod) {
    if (mod && mod.__esModule) return mod;
    var result = {};
    if (mod != null) for (var k in mod) if (k !== "default" && Object.prototype.hasOwnProperty.call(mod, k)) __createBinding(result, mod, k);
    __setModuleDefault(result, mod);
    return result;
};
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const PropertySymbol = __importStar(require("../../PropertySymbol.cjs"));
const crypto_1 = __importDefault(require("crypto"));
const EventTarget_js_1 = __importDefault(require("../../event/EventTarget.cjs"));
const MediaStreamTrackEvent_js_1 = __importDefault(require("../../event/events/MediaStreamTrackEvent.cjs"));
/**
 * MediaStream.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/MediaStream
 */
class MediaStream extends EventTarget_js_1.default {
    // Public properties
    active = true;
    id = crypto_1.default.randomUUID();
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
        this.dispatchEvent(new MediaStreamTrackEvent_js_1.default('addtrack', { track }));
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
        this.dispatchEvent(new MediaStreamTrackEvent_js_1.default('removetrack', { track }));
    }
}
exports.default = MediaStream;
//# sourceMappingURL=MediaStream.cjs.map