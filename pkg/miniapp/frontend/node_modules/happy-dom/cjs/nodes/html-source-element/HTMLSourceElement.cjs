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
/**
 * HTMLSourceElement
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/HTMLSourceElement
 */
class HTMLSourceElement extends HTMLElement_js_1.default {
    /**
     * Returns height.
     *
     * @returns Height.
     */
    get height() {
        const value = Number(this.getAttribute('height'));
        return isNaN(value) || value < 0 ? 0 : value;
    }
    /**
     * Sets height.
     *
     * @param value Height.
     */
    set height(value) {
        const parsedValue = Number(value);
        this.setAttribute('height', isNaN(parsedValue) || parsedValue < 0 ? '0' : String(parsedValue));
    }
    /**
     * Returns width.
     *
     * @returns Width.
     */
    get width() {
        const value = Number(this.getAttribute('width'));
        return isNaN(value) || value < 0 ? 0 : value;
    }
    /**
     * Sets width.
     *
     * @param value Width.
     */
    set width(value) {
        const parsedValue = Number(value);
        this.setAttribute('width', isNaN(parsedValue) || parsedValue < 0 ? '0' : String(parsedValue));
    }
    /**
     * Returns media.
     *
     * @returns Media.
     */
    get media() {
        return this.getAttribute('media') || '';
    }
    /**
     * Sets media.
     *
     * @param value Media.
     */
    set media(value) {
        this.setAttribute('media', value);
    }
    /**
     * Returns sizes.
     *
     * @returns Sizes.
     */
    get sizes() {
        return this.getAttribute('sizes') || '';
    }
    /**
     * Sets sizes.
     *
     * @param value Sizes.
     */
    set sizes(value) {
        this.setAttribute('sizes', value);
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
     * Returns source set.
     *
     * @returns Source set.
     */
    get srcset() {
        return this.getAttribute('srcset') || '';
    }
    /**
     * Sets source set.
     *
     * @param value Source set.
     */
    set srcset(value) {
        this.setAttribute('srcset', value);
    }
    /**
     * Returns type.
     *
     * @returns Type.
     */
    get type() {
        return this.getAttribute('type') || '';
    }
    /**
     * Sets type.
     *
     * @param type Type.
     */
    set type(type) {
        this.setAttribute('type', type);
    }
}
exports.default = HTMLSourceElement;
//# sourceMappingURL=HTMLSourceElement.cjs.map