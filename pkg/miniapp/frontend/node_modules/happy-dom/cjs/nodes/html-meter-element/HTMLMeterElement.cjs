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
 * HTMLMeterElement
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/HTMLMeterElement
 */
class HTMLMeterElement extends HTMLElement_js_1.default {
    /**
     * Returns high.
     *
     * @returns High.
     */
    get high() {
        if (!this.hasAttribute('high')) {
            return 1;
        }
        const parsedValue = parseFloat(this.getAttribute('high') || '');
        if (isNaN(parsedValue) || parsedValue > 1) {
            return 1;
        }
        return parsedValue < 0 ? 0 : parsedValue;
    }
    /**
     * Sets high.
     *
     * @param high High.
     */
    set high(high) {
        high = typeof high !== 'number' ? Number(high) : high;
        if (isNaN(high)) {
            throw new this[PropertySymbol.window].TypeError("Failed to set the 'high' property on 'HTMLMeterElement': The provided double value is non-finite.");
        }
        this.setAttribute('high', String(high));
    }
    /**
     * Returns low.
     *
     * @returns Low.
     */
    get low() {
        if (!this.hasAttribute('low')) {
            return 0;
        }
        const parsedValue = parseFloat(this.getAttribute('low') || '');
        if (isNaN(parsedValue) || parsedValue < 0) {
            return 0;
        }
        return parsedValue > 1 ? 1 : parsedValue;
    }
    /**
     * Sets low.
     *
     * @param low Low.
     */
    set low(low) {
        low = typeof low !== 'number' ? Number(low) : low;
        if (isNaN(low)) {
            throw new this[PropertySymbol.window].TypeError("Failed to set the 'low' property on 'HTMLMeterElement': The provided double value is non-finite.");
        }
        this.setAttribute('low', String(low));
    }
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
        if (isNaN(parsedValue) || parsedValue > 1) {
            return 1;
        }
        return parsedValue < 0 ? 0 : parsedValue;
    }
    /**
     * Sets max.
     *
     * @param max Max.
     */
    set max(max) {
        max = typeof max !== 'number' ? Number(max) : max;
        if (isNaN(max)) {
            throw new this[PropertySymbol.window].TypeError("Failed to set the 'max' property on 'HTMLMeterElement': The provided double value is non-finite.");
        }
        this.setAttribute('max', String(max));
    }
    /**
     * Returns min.
     *
     * @returns Min.
     */
    get min() {
        if (!this.hasAttribute('min')) {
            return 0;
        }
        const parsedValue = parseFloat(this.getAttribute('min') || '');
        if (isNaN(parsedValue) || parsedValue < 0) {
            return 0;
        }
        return parsedValue > 1 ? 1 : parsedValue;
    }
    /**
     * Sets min.
     *
     * @param min Min.
     */
    set min(min) {
        min = typeof min !== 'number' ? Number(min) : min;
        if (isNaN(min)) {
            throw new this[PropertySymbol.window].TypeError("Failed to set the 'min' property on 'HTMLMeterElement': The provided double value is non-finite.");
        }
        this.setAttribute('min', String(min));
    }
    /**
     * Returns optimum.
     *
     * @returns Optimum.
     */
    get optimum() {
        if (!this.hasAttribute('optimum')) {
            return 0.5;
        }
        const parsedValue = parseFloat(this.getAttribute('optimum') || '');
        if (isNaN(parsedValue)) {
            return 0.5;
        }
        if (parsedValue < 0) {
            return 0;
        }
        return parsedValue > 1 ? 1 : parsedValue;
    }
    /**
     * Sets optimum.
     *
     * @param optimum Optimum.
     */
    set optimum(optimum) {
        optimum = typeof optimum !== 'number' ? Number(optimum) : optimum;
        if (isNaN(optimum)) {
            throw new this[PropertySymbol.window].TypeError("Failed to set the 'optimum' property on 'HTMLMeterElement': The provided double value is non-finite.");
        }
        this.setAttribute('optimum', String(optimum));
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
        return parsedValue > 1 ? 1 : parsedValue;
    }
    /**
     * Sets value.
     *
     * @param value Value.
     */
    set value(value) {
        value = typeof value !== 'number' ? Number(value) : value;
        if (isNaN(value)) {
            throw new this[PropertySymbol.window].TypeError("Failed to set the 'value' property on 'HTMLMeterElement': The provided double value is non-finite.");
        }
        this.setAttribute('value', String(value));
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
exports.default = HTMLMeterElement;
//# sourceMappingURL=HTMLMeterElement.cjs.map