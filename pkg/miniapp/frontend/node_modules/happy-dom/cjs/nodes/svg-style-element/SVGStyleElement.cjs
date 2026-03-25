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
const CSSStyleSheet_js_1 = __importDefault(require("../../css/CSSStyleSheet.cjs"));
const PropertySymbol = __importStar(require("../../PropertySymbol.cjs"));
const SVGElement_js_1 = __importDefault(require("../svg-element/SVGElement.cjs"));
/**
 * SVG Style Element.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGStyleElement
 */
class SVGStyleElement extends SVGElement_js_1.default {
    // Internal properties
    [PropertySymbol.sheet] = null;
    [PropertySymbol.styleNode] = this;
    [PropertySymbol.disabled] = false;
    /**
     * Returns media.
     *
     * @returns Media.
     */
    get media() {
        return this.getAttribute('media') || 'all';
    }
    /**
     * Sets media.
     *
     * @param media Media.
     */
    set media(media) {
        this.setAttribute('media', media);
    }
    /**
     * Returns type.
     *
     * @deprecated
     * @returns Type.
     */
    get type() {
        return this.getAttribute('type') || 'text/css';
    }
    /**
     * Sets type.
     *
     * @deprecated
     * @param type Type.
     */
    set type(type) {
        this.setAttribute('type', type);
    }
    /**
     * Returns title.
     *
     * @returns Title.
     */
    get title() {
        return this.getAttribute('title') || '';
    }
    /**
     * Sets title.
     *
     * @param title Title.
     */
    set title(title) {
        this.setAttribute('title', title);
    }
    /**
     * Returns disabled.
     *
     * @returns Disabled.
     */
    get disabled() {
        return this[PropertySymbol.disabled];
    }
    /**
     * Sets disabled.
     *
     * @param disabled Disabled.
     */
    set disabled(disabled) {
        this[PropertySymbol.disabled] = Boolean(disabled);
    }
    /**
     * Returns style.
     *
     * @returns Style.
     */
    get sheet() {
        if (!this[PropertySymbol.isConnected]) {
            return null;
        }
        if (!this[PropertySymbol.sheet]) {
            this[PropertySymbol.sheet] = new CSSStyleSheet_js_1.default();
            this[PropertySymbol.sheet].replaceSync(this.textContent);
        }
        return this[PropertySymbol.sheet];
    }
    /**
     * @override
     */
    [PropertySymbol.disconnectedFromDocument]() {
        super[PropertySymbol.disconnectedFromDocument]();
        this[PropertySymbol.sheet] = null;
    }
    /**
     * Updates the CSSStyleSheet with the text content.
     */
    [PropertySymbol.updateSheet]() {
        if (this[PropertySymbol.sheet]) {
            this[PropertySymbol.sheet].replaceSync(this.textContent);
        }
    }
}
exports.default = SVGStyleElement;
//# sourceMappingURL=SVGStyleElement.cjs.map