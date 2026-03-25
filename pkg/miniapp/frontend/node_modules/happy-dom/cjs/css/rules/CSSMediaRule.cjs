"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const CSSRule_js_1 = __importDefault(require("../CSSRule.cjs"));
const MediaList_js_1 = __importDefault(require("../MediaList.cjs"));
/**
 * CSSRule interface.
 */
class CSSMediaRule extends CSSRule_js_1.default {
    type = CSSRule_js_1.default.MEDIA_RULE;
    cssRules = [];
    media = new MediaList_js_1.default();
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
        return `@media ${this.conditionText} { ${cssText} }`;
    }
    /**
     * Returns conditional text.
     *
     * @returns Conditional text.
     */
    get conditionText() {
        return this.media.mediaText;
    }
}
exports.default = CSSMediaRule;
//# sourceMappingURL=CSSMediaRule.cjs.map