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
const DOMExceptionNameEnum_js_1 = __importDefault(require("../exception/DOMExceptionNameEnum.cjs"));
const CSSParser_js_1 = __importDefault(require("./utilities/CSSParser.cjs"));
const PropertySymbol = __importStar(require("../PropertySymbol.cjs"));
/**
 * CSS StyleSheet.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/CSSStyleSheet.
 */
class CSSStyleSheet {
    value = null;
    name = null;
    namespaceURI = null;
    cssRules = [];
    // TODO: MediaList is not fully implemented.
    media;
    title;
    alternate;
    disabled;
    #currentText = null;
    /**
     * Constructor.
     *
     * @param [options] Options.
     * @param [options.media] Media.
     * @param [options.title] Title.
     * @param [options.alternate] Alternate.
     * @param [options.disabled] Disabled.
     */
    constructor(options) {
        this.media = options && options.media ? options.media : '';
        this.title = options && options.title ? options.title : '';
        this.alternate = options && options.alternate ? options.alternate : false;
        this.disabled = options && options.disabled ? options.disabled : false;
    }
    /**
     * Inserts a rule.
     *
     * @see https://developer.mozilla.org/en-US/docs/Web/API/CSSStyleSheet/insertRule
     * @param rule Rule.
     * @param [index] Index.
     * @returns The newly inserterted rule's index.
     */
    insertRule(rule, index) {
        const rules = CSSParser_js_1.default.parseFromString(this, rule);
        if (rules.length === 0) {
            throw new this[PropertySymbol.window].DOMException('Invalid CSS rule.', DOMExceptionNameEnum_js_1.default.hierarchyRequestError);
        }
        if (rules.length > 1) {
            throw new this[PropertySymbol.window].DOMException('Only one rule is allowed to be added.', DOMExceptionNameEnum_js_1.default.syntaxError);
        }
        if (index !== undefined) {
            if (index > this.cssRules.length) {
                throw new this[PropertySymbol.window].DOMException('Index is more than the length of CSSRuleList.', DOMExceptionNameEnum_js_1.default.indexSizeError);
            }
            this.cssRules.splice(index, 0, rules[0]);
            return index;
        }
        const newIndex = this.cssRules.length;
        this.cssRules.push(rules[0]);
        return newIndex;
    }
    /**
     * Removes a rule.
     *
     * @see https://developer.mozilla.org/en-US/docs/Web/API/CSSStyleSheet/deleteRule
     * @param index Index.
     */
    deleteRule(index) {
        delete this.cssRules[index];
    }
    /**
     * Replaces all CSS rules.
     *
     * @see https://developer.mozilla.org/en-US/docs/Web/API/CSSStyleSheet/replace
     * @param text CSS text.
     * @returns Promise.
     */
    async replace(text) {
        this.replaceSync(text);
    }
    /**
     * Replaces all CSS rules.
     *
     * @see https://developer.mozilla.org/en-US/docs/Web/API/CSSStyleSheet/replaceSync
     * @param text CSS text.
     */
    replaceSync(text) {
        if (this.#currentText !== text) {
            this.#currentText = text;
            this.cssRules = CSSParser_js_1.default.parseFromString(this, text);
        }
    }
}
exports.default = CSSStyleSheet;
//# sourceMappingURL=CSSStyleSheet.cjs.map