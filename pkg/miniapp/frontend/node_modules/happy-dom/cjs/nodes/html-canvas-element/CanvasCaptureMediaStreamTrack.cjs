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
const MediaStreamTrack_js_1 = __importDefault(require("../html-media-element/MediaStreamTrack.cjs"));
const PropertySymbol = __importStar(require("../../PropertySymbol.cjs"));
/**
 * Canvas Capture Media Stream Track.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/CanvasCaptureMediaStreamTrack
 */
class CanvasCaptureMediaStreamTrack extends MediaStreamTrack_js_1.default {
    [PropertySymbol.canvas];
    /**
     * Constructor.
     *
     * @param illegalConstructorSymbol Illegal constructor symbol.
     * @param canvas Canvas.
     */
    constructor(illegalConstructorSymbol, canvas) {
        super(illegalConstructorSymbol);
        this[PropertySymbol.canvas] = canvas;
    }
    /**
     * Returns the canvas.
     *
     * @returns Canvas.
     */
    get canvas() {
        return this[PropertySymbol.canvas];
    }
    /**
     * Requests a frame.
     */
    requestFrame() {
        // Do nothing
    }
    /**
     * Clones the track.
     *
     * @returns Clone.
     */
    clone() {
        const clone = super.clone();
        clone[PropertySymbol.canvas] = this.canvas;
        return clone;
    }
}
exports.default = CanvasCaptureMediaStreamTrack;
//# sourceMappingURL=CanvasCaptureMediaStreamTrack.cjs.map