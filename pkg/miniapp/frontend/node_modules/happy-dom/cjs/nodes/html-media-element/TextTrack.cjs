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
const EventTarget_js_1 = __importDefault(require("../../event/EventTarget.cjs"));
const TextTrackCueList_js_1 = __importDefault(require("./TextTrackCueList.cjs"));
const PropertySymbol = __importStar(require("../../PropertySymbol.cjs"));
const TextTrackKindEnum_js_1 = __importDefault(require("./TextTrackKindEnum.cjs"));
/**
 * TextTrack.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/TextTrack
 */
class TextTrack extends EventTarget_js_1.default {
    // Internal properties
    [PropertySymbol.kind] = TextTrackKindEnum_js_1.default.subtitles;
    [PropertySymbol.label] = '';
    [PropertySymbol.language] = '';
    [PropertySymbol.id] = '';
    [PropertySymbol.mode] = 'disabled';
    [PropertySymbol.cues] = new TextTrackCueList_js_1.default(PropertySymbol.illegalConstructor);
    [PropertySymbol.activeCues] = new TextTrackCueList_js_1.default(PropertySymbol.illegalConstructor);
    // Events
    oncuechange = null;
    /**
     * Constructor.
     *
     * @param illegalConstructorSymbol Illegal constructor symbol.
     */
    constructor(illegalConstructorSymbol) {
        super();
        if (illegalConstructorSymbol !== PropertySymbol.illegalConstructor) {
            throw new TypeError('Illegal constructor');
        }
        if (!this[PropertySymbol.window]) {
            throw new TypeError(`Failed to construct '${this.constructor.name}': '${this.constructor.name}' was constructed outside a Window context.`);
        }
    }
    /**
     * Returns the kind of the text track.
     *
     * @returns Kind.
     */
    get kind() {
        return this[PropertySymbol.kind];
    }
    /**
     * Returns the label of the text track.
     *
     * @returns Label.
     */
    get label() {
        return this[PropertySymbol.label];
    }
    /**
     * Returns the language of the text track.
     *
     * @returns Language.
     */
    get language() {
        return this[PropertySymbol.language];
    }
    /**
     * Returns the id of the text track.
     *
     * @returns Id.
     */
    get id() {
        return this[PropertySymbol.id];
    }
    /**
     * Returns the mode of the text track.
     *
     * @returns Mode.
     */
    get mode() {
        return this[PropertySymbol.mode];
    }
    /**
     * Sets the mode of the text track.
     *
     * @param mode Mode.
     */
    set mode(mode) {
        if (mode !== 'disabled' && mode !== 'showing') {
            // TODO: Browser outputs a warning here.
            return;
        }
        this[PropertySymbol.mode] = mode;
    }
    /**
     * Returns the list of cues in the track list.
     *
     * @returns List of cues.
     */
    get cues() {
        if (this[PropertySymbol.mode] === 'disabled') {
            return null;
        }
        return this[PropertySymbol.cues];
    }
    /**
     * Returns the list of active cues in the track list.
     *
     * @returns List of active cues.
     */
    get activeCues() {
        if (this[PropertySymbol.mode] === 'disabled') {
            return null;
        }
        return this[PropertySymbol.activeCues];
    }
    /**
     * Adds a cue to the track list.
     *
     * @param cue Text track cue.
     */
    addCue(cue) {
        if (this[PropertySymbol.cues].includes(cue)) {
            return;
        }
        cue[PropertySymbol.track] = this;
        this[PropertySymbol.cues].push(cue);
    }
    /**
     * Removes a cue from the track list.
     *
     * @param cue Text track cue.
     */
    removeCue(cue) {
        const index = this[PropertySymbol.cues].indexOf(cue);
        if (index !== -1) {
            cue[PropertySymbol.track] = null;
            this[PropertySymbol.cues].splice(index, 1);
        }
    }
}
exports.default = TextTrack;
//# sourceMappingURL=TextTrack.cjs.map