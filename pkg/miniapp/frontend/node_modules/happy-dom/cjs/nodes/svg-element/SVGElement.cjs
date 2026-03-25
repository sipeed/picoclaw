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
const CSSStyleDeclaration_js_1 = __importDefault(require("../../css/declaration/CSSStyleDeclaration.cjs"));
const PropertySymbol = __importStar(require("../../PropertySymbol.cjs"));
const Element_js_1 = __importDefault(require("../element/Element.cjs"));
const HTMLElementUtility_js_1 = __importDefault(require("../html-element/HTMLElementUtility.cjs"));
const DOMStringMap_js_1 = __importDefault(require("../../dom/DOMStringMap.cjs"));
/**
 * SVG Element.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/SVGElement.
 */
class SVGElement extends Element_js_1.default {
    // Events
    onabort = null;
    onerror = null;
    onload = null;
    onresize = null;
    onscroll = null;
    onunload = null;
    // Internal properties
    [PropertySymbol.style] = null;
    // Private properties
    #dataset = null;
    /**
     * Returns current translate.
     *
     * @returns Element.
     */
    get ownerSVGElement() {
        let parent = this[PropertySymbol.parentNode];
        while (parent) {
            if (parent[PropertySymbol.localName] === 'svg') {
                return parent;
            }
            parent = parent[PropertySymbol.parentNode];
        }
        return null;
    }
    /**
     * Returns the SVGElement which established the current viewport. Often the nearest ancestor <svg> element. null if the given element is the outermost <svg> element.
     *
     * @returns SVG element.
     */
    get viewportElement() {
        return this.ownerSVGElement;
    }
    /**
     * Returns data set.
     *
     * @returns Data set.
     */
    get dataset() {
        return (this.#dataset ??= new DOMStringMap_js_1.default(PropertySymbol.illegalConstructor, this));
    }
    /**
     * Returns style.
     *
     * @returns Style.
     */
    get style() {
        if (!this[PropertySymbol.style]) {
            this[PropertySymbol.style] = new CSSStyleDeclaration_js_1.default(PropertySymbol.illegalConstructor, this[PropertySymbol.window], { element: this });
        }
        return this[PropertySymbol.style];
    }
    /**
     * Returns tab index.
     *
     * @returns Tab index.
     */
    get tabIndex() {
        const tabIndex = this.getAttribute('tabindex');
        return tabIndex !== null ? Number(tabIndex) : -1;
    }
    /**
     * Returns tab index.
     *
     * @param tabIndex Tab index.
     */
    set tabIndex(tabIndex) {
        if (tabIndex === -1) {
            this.removeAttribute('tabindex');
        }
        else {
            this.setAttribute('tabindex', String(tabIndex));
        }
    }
    /**
     * Triggers a blur event.
     */
    blur() {
        HTMLElementUtility_js_1.default.blur(this);
    }
    /**
     * Triggers a focus event.
     */
    focus() {
        HTMLElementUtility_js_1.default.focus(this);
    }
}
exports.default = SVGElement;
//# sourceMappingURL=SVGElement.cjs.map