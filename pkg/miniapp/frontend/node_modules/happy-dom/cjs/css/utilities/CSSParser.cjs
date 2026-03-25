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
const PropertySymbol = __importStar(require("../../PropertySymbol.cjs"));
const CSSStyleRule_js_1 = __importDefault(require("../rules/CSSStyleRule.cjs"));
const CSSKeyframeRule_js_1 = __importDefault(require("../rules/CSSKeyframeRule.cjs"));
const CSSKeyframesRule_js_1 = __importDefault(require("../rules/CSSKeyframesRule.cjs"));
const CSSMediaRule_js_1 = __importDefault(require("../rules/CSSMediaRule.cjs"));
const CSSContainerRule_js_1 = __importDefault(require("../rules/CSSContainerRule.cjs"));
const CSSSupportsRule_js_1 = __importDefault(require("../rules/CSSSupportsRule.cjs"));
const CSSFontFaceRule_js_1 = __importDefault(require("../rules/CSSFontFaceRule.cjs"));
const SelectorParser_js_1 = __importDefault(require("../../query-selector/SelectorParser.cjs"));
const COMMENT_REGEXP = /\/\*[\s\S]*?\*\//gm;
/**
 * CSS parser.
 */
class CSSParser {
    /**
     * Parses HTML and returns a root element.
     *
     * @param parentStyleSheet Parent style sheet.
     * @param cssText CSS code.
     * @returns Root element.
     */
    static parseFromString(parentStyleSheet, cssText) {
        const window = parentStyleSheet[PropertySymbol.window];
        const css = cssText.replace(COMMENT_REGEXP, '');
        const cssRules = [];
        const regExp = /{|}/gm;
        const stack = [];
        let parentRule = null;
        let lastIndex = 0;
        let match;
        while ((match = regExp.exec(css))) {
            if (match[0] === '{') {
                const selectorText = css.substring(lastIndex, match.index).trim();
                if (selectorText.startsWith('@keyframes') ||
                    selectorText.startsWith('@-webkit-keyframes')) {
                    const newRule = new CSSKeyframesRule_js_1.default(PropertySymbol.illegalConstructor, window);
                    newRule.name = selectorText.replace(/@(-webkit-){0,1}keyframes +/, '');
                    newRule.parentStyleSheet = parentStyleSheet;
                    cssRules.push(newRule);
                    parentRule = newRule;
                }
                else if (selectorText.startsWith('@media')) {
                    const mediums = selectorText.replace('@media', '').split(',');
                    const newRule = new CSSMediaRule_js_1.default(PropertySymbol.illegalConstructor, window);
                    for (const medium of mediums) {
                        newRule.media.appendMedium(medium.trim());
                    }
                    newRule.parentStyleSheet = parentStyleSheet;
                    cssRules.push(newRule);
                    parentRule = newRule;
                }
                else if (selectorText.startsWith('@container') ||
                    selectorText.startsWith('@-webkit-container')) {
                    const conditionText = selectorText.replace(/@(-webkit-){0,1}container +/, '');
                    const newRule = new CSSContainerRule_js_1.default(PropertySymbol.illegalConstructor, window);
                    newRule.conditionText = conditionText;
                    newRule.parentStyleSheet = parentStyleSheet;
                    cssRules.push(newRule);
                    parentRule = newRule;
                }
                else if (selectorText.startsWith('@supports') ||
                    selectorText.startsWith('@-webkit-supports')) {
                    const conditionText = selectorText.replace(/@(-webkit-){0,1}supports +/, '');
                    const newRule = new CSSSupportsRule_js_1.default(PropertySymbol.illegalConstructor, window);
                    newRule.conditionText = conditionText;
                    newRule.parentStyleSheet = parentStyleSheet;
                    cssRules.push(newRule);
                    parentRule = newRule;
                }
                else if (selectorText.startsWith('@font-face')) {
                    const conditionText = selectorText.replace('@font-face', '');
                    const newRule = new CSSFontFaceRule_js_1.default(PropertySymbol.illegalConstructor, window);
                    newRule[PropertySymbol.cssText] = conditionText;
                    newRule.parentStyleSheet = parentStyleSheet;
                    cssRules.push(newRule);
                    parentRule = newRule;
                }
                else if (selectorText[0] === '@') {
                    // Unknown rule.
                    // We will create a new rule to let it grab its content, but we will not add it to the cssRules array.
                    const newRule = new CSSRule_js_1.default(PropertySymbol.illegalConstructor, window);
                    newRule.parentStyleSheet = parentStyleSheet;
                    parentRule = newRule;
                }
                else if (parentRule && parentRule.type === CSSRule_js_1.default.KEYFRAMES_RULE) {
                    const newRule = new CSSKeyframeRule_js_1.default(PropertySymbol.illegalConstructor, window);
                    newRule.keyText = selectorText.trim();
                    newRule.parentStyleSheet = parentStyleSheet;
                    newRule.parentRule = parentRule;
                    parentRule.cssRules.push(newRule);
                    parentRule = newRule;
                }
                else if (parentRule &&
                    (parentRule.type === CSSRule_js_1.default.MEDIA_RULE ||
                        parentRule.type === CSSRule_js_1.default.CONTAINER_RULE ||
                        parentRule.type === CSSRule_js_1.default.SUPPORTS_RULE)) {
                    if (this.validateSelectorText(selectorText)) {
                        const newRule = new CSSStyleRule_js_1.default(PropertySymbol.illegalConstructor, window);
                        newRule.selectorText = selectorText;
                        newRule.parentStyleSheet = parentStyleSheet;
                        newRule.parentRule = parentRule;
                        parentRule.cssRules.push(newRule);
                        parentRule = newRule;
                    }
                }
                else {
                    if (this.validateSelectorText(selectorText)) {
                        const newRule = new CSSStyleRule_js_1.default(PropertySymbol.illegalConstructor, window);
                        newRule.selectorText = selectorText;
                        newRule.parentStyleSheet = parentStyleSheet;
                        newRule.parentRule = parentRule;
                        if (!parentRule) {
                            cssRules.push(newRule);
                        }
                        parentRule = newRule;
                    }
                }
                stack.push(parentRule);
            }
            else {
                if (parentRule) {
                    const cssText = css
                        .substring(lastIndex, match.index)
                        .trim()
                        .replace(/([^;])$/, '$1;'); // Ensure last semicolon
                    switch (parentRule.type) {
                        case CSSRule_js_1.default.FONT_FACE_RULE:
                        case CSSRule_js_1.default.KEYFRAME_RULE:
                        case CSSRule_js_1.default.STYLE_RULE:
                            parentRule[PropertySymbol.cssText] = cssText;
                            break;
                    }
                }
                stack.pop();
                parentRule = stack[stack.length - 1] || null;
            }
            lastIndex = match.index + 1;
        }
        return cssRules;
    }
    /**
     * Validates a selector text.
     *
     * @see https://www.w3.org/TR/CSS21/syndata.html#rule-sets
     * @param selectorText Selector text.
     * @returns True if valid, false otherwise.
     */
    static validateSelectorText(selectorText) {
        try {
            SelectorParser_js_1.default.getSelectorGroups(selectorText);
        }
        catch (e) {
            return false;
        }
        return true;
    }
}
exports.default = CSSParser;
//# sourceMappingURL=CSSParser.cjs.map