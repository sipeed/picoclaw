"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const CSSRule_js_1 = __importDefault(require("../CSSRule.cjs"));
/**
 * CSSRule interface.
 */
class CSSSupportsRule extends CSSRule_js_1.default {
    type = CSSRule_js_1.default.SUPPORTS_RULE;
    cssRules = [];
    conditionText = '';
    /**
     * Returns css text.
     *
     * @returns CSS text.
     */
    get cssText() {
        let cssText = '';
        for (const cssRule of this.cssRules) {
            cssText += cssRule.cssText;
        }
        return `@supports ${this.conditionText} { ${cssText} }`;
    }
}
exports.default = CSSSupportsRule;
//# sourceMappingURL=CSSSupportsRule.cjs.map