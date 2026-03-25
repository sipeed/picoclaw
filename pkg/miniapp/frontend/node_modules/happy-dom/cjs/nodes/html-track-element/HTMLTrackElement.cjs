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
const HTMLElement_js_1 = __importDefault(require("../html-element/HTMLElement.cjs"));
const PropertySymbol = __importStar(require("../../PropertySymbol.cjs"));
const TextTrackKindEnum_js_1 = __importDefault(require("../html-media-element/TextTrackKindEnum.cjs"));
/**
 * HTMLTrackElement
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/HTMLTrackElement
 */
class HTMLTrackElement extends HTMLElement_js_1.default {
    // Events
    oncuechange = null;
    /**
     * Returns kind.
     *
     * @returns Kind.
     */
    get kind() {
        const kind = this.getAttribute('kind');
        if (kind === null) {
            return TextTrackKindEnum_js_1.default.subtitles;
        }
        if (!TextTrackKindEnum_js_1.default[kind]) {
            return TextTrackKindEnum_js_1.default.metadata;
        }
        return kind;
    }
    /**
     * Sets kind.
     *
     * @param value Value.
     */
    set kind(value) {
        if (!TextTrackKindEnum_js_1.default[value]) {
            value = TextTrackKindEnum_js_1.default.metadata;
        }
        this.setAttribute('kind', value);
    }
    /**
     * Returns source.
     *
     * @returns Source.
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
     * Sets source.
     *
     * @param src Source.
     */
    set src(src) {
        this.setAttribute('src', src);
    }
    /**
     * Returns source language.
     *
     * @returns Source language.
     */
    get srclang() {
        return this.getAttribute('srclang') || '';
    }
    /**
     * Sets source language.
     *
     * @param value Value.
     */
    set srclang(value) {
        this.setAttribute('srclang', value);
    }
    /**
     * Returns label.
     *
     * @returns Label.
     */
    get label() {
        return this.getAttribute('label') || '';
    }
    /**
     * Sets label.
     *
     * @param value Value.
     */
    set label(value) {
        this.setAttribute('label', value);
    }
    /**
     * Returns default.
     *
     * @returns Default.
     */
    get default() {
        return this.hasAttribute('default');
    }
    /**
     * Sets default.
     *
     * @param value Value.
     */
    set default(value) {
        if (value) {
            this.setAttribute('default', '');
        }
        else {
            this.removeAttribute('default');
        }
    }
    /**
     * Returns ready state.
     *
     * @returns Ready state.
     */
    get readyState() {
        return 0;
    }
    /**
     * Returns the TextTrack object corresponding to the track element.
     *
     * @returns TextTrack
     */
    get track() {
        const textTrack = new this[PropertySymbol.window].TextTrack(PropertySymbol.illegalConstructor);
        textTrack[PropertySymbol.kind] = this.kind;
        textTrack[PropertySymbol.label] = this.label;
        textTrack[PropertySymbol.language] = this.srclang;
        textTrack[PropertySymbol.mode] = this.default ? 'showing' : 'disabled';
        textTrack[PropertySymbol.id] = this.id;
        return textTrack;
    }
}
exports.default = HTMLTrackElement;
//# sourceMappingURL=HTMLTrackElement.cjs.map