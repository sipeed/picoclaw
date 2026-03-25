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
const TextTrackCue_js_1 = __importDefault(require("./TextTrackCue.cjs"));
const PropertySymbol = __importStar(require("../../PropertySymbol.cjs"));
/**
 * VTTCue.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/VTTCue
 */
class VTTCue extends TextTrackCue_js_1.default {
    region = null;
    vertical = '';
    snapToLines = true;
    line = 0;
    lineAlign = '';
    position = 'auto';
    positionAlign = 'auto';
    size = 100;
    align = '';
    text = '';
    /**
     * Constructor.
     *
     * @param startTime The start time for the cue.
     * @param endTime The end time for the cue.
     * @param text The text of the cue.
     */
    constructor(startTime, endTime, text) {
        super(PropertySymbol.illegalConstructor);
        const window = this[PropertySymbol.window];
        // TODO: Can we find a better solution for counting arguments by using the "arguments" property?
        let argumentCount = 0;
        if (startTime !== undefined) {
            argumentCount++;
        }
        if (endTime !== undefined) {
            argumentCount++;
        }
        if (text !== undefined) {
            argumentCount++;
        }
        if (argumentCount < 3) {
            throw new window.TypeError(`Failed to construct 'VTTCue': 3 arguments required, but only ${argumentCount} present.`);
        }
        startTime = Number(startTime);
        endTime = Number(endTime);
        if (isNaN(startTime) || isNaN(endTime)) {
            throw new window.TypeError(`Failed to construct 'VTTCue': The provided double value is non-finite.`);
        }
        this.startTime = startTime;
        this.endTime = endTime;
        this.text = String(text);
    }
    /**
     * Returns the cue as HTML.
     *
     * @returns DocumentFragment
     */
    getCueAsHTML() {
        const window = this[PropertySymbol.window];
        const fragment = window.document.createDocumentFragment();
        fragment.appendChild(window.document.createTextNode(this.text));
        return fragment;
    }
}
exports.default = VTTCue;
//# sourceMappingURL=VTTCue.cjs.map