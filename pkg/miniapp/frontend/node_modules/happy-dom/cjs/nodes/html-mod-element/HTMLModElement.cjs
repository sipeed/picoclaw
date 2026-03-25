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
 * HTMLModElement
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/HTMLModElement
 */
class HTMLModElement extends HTMLElement_js_1.default {
    /**
     * Returns source.
     *
     * @returns Source.
     */
    get cite() {
        if (!this.hasAttribute('cite')) {
            return '';
        }
        try {
            return new URL(this.getAttribute('cite'), this[PropertySymbol.ownerDocument].location.href)
                .href;
        }
        catch (e) {
            return this.getAttribute('cite');
        }
    }
    /**
     * Sets source.
     *
     * @param cite Source.
     */
    set cite(cite) {
        this.setAttribute('cite', cite);
    }
    /**
     * Returns date time.
     *
     * @returns Date time.
     */
    get dateTime() {
        return this.getAttribute('datetime') || '';
    }
    /**
     * Sets date time.
     *
     * @param dateTime Date time.
     */
    set dateTime(dateTime) {
        this.setAttribute('datetime', dateTime);
    }
}
exports.default = HTMLModElement;
//# sourceMappingURL=HTMLModElement.cjs.map