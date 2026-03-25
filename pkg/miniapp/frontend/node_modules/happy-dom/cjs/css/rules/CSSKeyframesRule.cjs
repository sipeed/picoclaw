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
const CSSRule_js_1 = __importDefault(require("../CSSRule.cjs"));
const CSSStyleDeclaration_js_1 = __importDefault(require("../declaration/CSSStyleDeclaration.cjs"));
const CSSKeyframeRule_js_1 = __importDefault(require("./CSSKeyframeRule.cjs"));
const PropertySymbol = __importStar(require("../../PropertySymbol.cjs"));
const CSS_RULE_REGEXP = /([^{]+){([^}]+)}/;
/**
 * CSSRule interface.
 */
class CSSKeyframesRule extends CSSRule_js_1.default {
    type = CSSRule_js_1.default.KEYFRAMES_RULE;
    cssRules = [];
    name = null;
    /**
     * Returns css text.
     *
     * @returns CSS text.
     */
    get cssText() {
        let cssText = '';
        for (const cssRule of this.cssRules) {
            cssText += cssRule.cssText + ' ';
        }
        return `@keyframes ${this.name} { ${cssText}}`;
    }
    /**
     * Appends a rule.
     *
     * @param rule Rule. E.g. "0% { transform: rotate(360deg); }".
     */
    appendRule(rule) {
        const match = rule.match(CSS_RULE_REGEXP);
        if (match) {
            const cssRule = new CSSKeyframeRule_js_1.default(PropertySymbol.illegalConstructor, this[PropertySymbol.window]);
            const style = new CSSStyleDeclaration_js_1.default(PropertySymbol.illegalConstructor, this[PropertySymbol.window]);
            cssRule.parentRule = this;
            cssRule.keyText = match[1].trim();
            style.cssText = match[2].trim();
            style.parentRule = this;
            cssRule.style = style;
        }
    }
    /**
     * Removes a rule.
     *
     * @param rule Rule. E.g. "0%".
     */
    deleteRule(rule) {
        for (let i = 0, max = this.cssRules.length; i < max; i++) {
            if (this.cssRules[i].keyText === rule) {
                this.cssRules.splice(i, 1);
                break;
            }
        }
    }
}
exports.default = CSSKeyframesRule;
//# sourceMappingURL=CSSKeyframesRule.cjs.map