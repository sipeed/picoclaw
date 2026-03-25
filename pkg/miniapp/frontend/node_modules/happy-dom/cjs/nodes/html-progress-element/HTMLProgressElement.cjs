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
const HTMLLabelElementUtility_js_1 = __importDefault(require("../html-label-element/HTMLLabelElementUtility.cjs"));
const PropertySymbol = __importStar(require("../../PropertySymbol.cjs"));
/**
 * HTMLProgressElement
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/HTMLProgressElement
 */
class HTMLProgressElement extends HTMLElement_js_1.default {
    /**
     * Returns max.
     *
     * @returns Max.
     */
    get max() {
        if (!this.hasAttribute('max')) {
            return 1;
        }
        const parsedValue = parseFloat(this.getAttribute('max') || '');
        if (isNaN(parsedValue) || parsedValue < 0) {
            return 1;
        }
        return parsedValue;
    }
    /**
     * Sets max.
     *
     * @param max Max.
     */
    set max(max) {
        max = typeof max !== 'number' ? Number(max) : max;
        if (isNaN(max)) {
            throw new this[PropertySymbol.window].TypeError("Failed to set the 'max' property on 'HTMLProgressElement': The provided double value is non-finite.");
        }
        this.setAttribute('max', max < 0 ? '1' : String(max));
    }
    /**
     * Returns value.
     *
     * @returns Value.
     */
    get value() {
        if (!this.hasAttribute('value')) {
            return 0;
        }
        const parsedValue = parseFloat(this.getAttribute('value') || '');
        if (isNaN(parsedValue) || parsedValue < 0) {
            return 0;
        }
        return parsedValue;
    }
    /**
     * Sets value.
     *
     * @param value Value.
     */
    set value(value) {
        value = typeof value !== 'number' ? Number(value) : value;
        if (isNaN(value)) {
            throw new this[PropertySymbol.window].TypeError("Failed to set the 'value' property on 'HTMLProgressElement': The provided double value is non-finite.");
        }
        this.setAttribute('value', value < 0 ? '0' : String(value));
    }
    /**
     * Returns position.
     *
     * @returns Position.
     */
    get position() {
        // If the progress bar is an indeterminate progress bar, it should return -1.
        // It is considered indeterminate if the value attribute is not set.
        // @see https://developer.mozilla.org/en-US/docs/Web/API/HTMLProgressElement/position#value
        if (!this.hasAttribute('value')) {
            return -1;
        }
        return this.value / this.max;
    }
    /**
     * Returns the associated label elements.
     *
     * @returns Label elements.
     */
    get labels() {
        return HTMLLabelElementUtility_js_1.default.getAssociatedLabelElements(this);
    }
}
exports.default = HTMLProgressElement;
//# sourceMappingURL=HTMLProgressElement.cjs.map