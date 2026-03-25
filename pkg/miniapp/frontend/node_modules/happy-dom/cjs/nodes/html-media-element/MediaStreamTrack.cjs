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
const PropertySymbol = __importStar(require("../../PropertySymbol.cjs"));
const crypto_1 = __importDefault(require("crypto"));
const CAPABILITIES = {
    aspectRatio: {
        max: 300,
        min: 0.006666666666666667
    },
    deviceId: '',
    facingMode: [],
    frameRate: {
        max: 60,
        min: 0
    },
    height: {
        max: 150,
        min: 1
    },
    resizeMode: ['none', 'crop-and-scale'],
    width: {
        max: 300,
        min: 1
    }
};
const SETTINGS = {
    deviceId: '',
    frameRate: 60,
    resizeMode: 'none'
};
/**
 * Canvas Capture Media Stream Track.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/MediaStreamTrack
 */
class MediaStreamTrack extends EventTarget_js_1.default {
    contentHint = '';
    enabled = true;
    id = crypto_1.default.randomUUID();
    muted = false;
    readyState = 'live';
    label = '';
    [PropertySymbol.label] = '';
    [PropertySymbol.kind] = 'video';
    [PropertySymbol.constraints] = {};
    [PropertySymbol.capabilities] = JSON.parse(JSON.stringify(CAPABILITIES));
    [PropertySymbol.settings] = JSON.parse(JSON.stringify(SETTINGS));
    // Events
    onended = null;
    onmute = null;
    onunmute = null;
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
     * Returns the kind of the track.
     *
     * @returns Kind.
     */
    get kind() {
        return this[PropertySymbol.kind];
    }
    /**
     * Applies constraints.
     *
     * @param _constraints Constraints.
     * @param constraints
     */
    async applyConstraints(constraints) {
        this.#mergeObjects(this[PropertySymbol.constraints], constraints);
    }
    /**
     * Returns constraints.
     *
     * @returns Constraints.
     */
    getConstraints() {
        return this[PropertySymbol.constraints];
    }
    /**
     * Returns capabilities.
     *
     * @returns Capabilities.
     */
    getCapabilities() {
        return this[PropertySymbol.capabilities];
    }
    /**
     * Returns settings.
     *
     * @returns Settings.
     */
    getSettings() {
        return this[PropertySymbol.settings];
    }
    /**
     * Clones the track.
     *
     * @returns Clone.
     */
    clone() {
        const clone = new this.constructor(PropertySymbol.illegalConstructor);
        clone[PropertySymbol.kind] = this[PropertySymbol.kind];
        clone[PropertySymbol.constraints] = this[PropertySymbol.constraints];
        clone[PropertySymbol.capabilities] = this[PropertySymbol.capabilities];
        clone[PropertySymbol.settings] = this[PropertySymbol.settings];
        clone.contentHint = this.contentHint;
        clone.enabled = this.enabled;
        clone.label = this.label;
        clone.muted = this.muted;
        clone.readyState = this.readyState;
        return clone;
    }
    /**
     * Stops the track.
     */
    stop() {
        this.readyState = 'ended';
    }
    /**
     * Merges two objects.
     *
     * @param source Target.
     * @param target Source.
     */
    #mergeObjects(source, target) {
        for (const key in target) {
            if (target[key] !== null && typeof target[key] === 'object' && !Array.isArray(target[key])) {
                if (typeof source[key] !== 'object') {
                    source[key] = {};
                }
                this.#mergeObjects(source[key], target[key]);
            }
            else {
                source[key] = target[key];
            }
        }
    }
}
exports.default = MediaStreamTrack;
//# sourceMappingURL=MediaStreamTrack.cjs.map