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
const SVGElement_js_1 = __importDefault(require("../svg-element/SVGElement.cjs"));
const SVGStringList_js_1 = __importDefault(require("../../svg/SVGStringList.cjs"));
const PropertySymbol = __importStar(require("../../PropertySymbol.cjs"));
/**
 * SVG Animation Element.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGAnimationElement
 */
class SVGAnimationElement extends SVGElement_js_1.default {
    // Internal properties
    [PropertySymbol.requiredExtensions] = null;
    [PropertySymbol.systemLanguage] = null;
    // Events
    onbegin = null;
    onend = null;
    onrepeat = null;
    /**
     * Returns required extensions.
     *
     * @returns Required extensions.
     */
    get requiredExtensions() {
        if (!this[PropertySymbol.requiredExtensions]) {
            this[PropertySymbol.requiredExtensions] = new SVGStringList_js_1.default(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('requiredExtensions'),
                setAttribute: (value) => this.setAttribute('requiredExtensions', value)
            });
        }
        return this[PropertySymbol.requiredExtensions];
    }
    /**
     * Returns system language.
     *
     * @returns System language.
     */
    get systemLanguage() {
        if (!this[PropertySymbol.systemLanguage]) {
            this[PropertySymbol.systemLanguage] = new SVGStringList_js_1.default(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('systemLanguage'),
                setAttribute: (value) => this.setAttribute('systemLanguage', value)
            });
        }
        return this[PropertySymbol.systemLanguage];
    }
    /**
     * Returns target element.
     *
     * @returns Target element.
     */
    get targetElement() {
        // TODO: Implement targetElement
        return null;
    }
}
exports.default = SVGAnimationElement;
//# sourceMappingURL=SVGAnimationElement.cjs.map