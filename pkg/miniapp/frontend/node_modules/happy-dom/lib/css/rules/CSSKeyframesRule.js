import CSSRule from '../CSSRule.js';
import CSSStyleDeclaration from '../declaration/CSSStyleDeclaration.js';
import CSSKeyframeRule from './CSSKeyframeRule.js';
import * as PropertySymbol from '../../PropertySymbol.js';
const CSS_RULE_REGEXP = /([^{]+){([^}]+)}/;
/**
 * CSSRule interface.
 */
export default class CSSKeyframesRule extends CSSRule {
    type = CSSRule.KEYFRAMES_RULE;
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
            const cssRule = new CSSKeyframeRule(PropertySymbol.illegalConstructor, this[PropertySymbol.window]);
            const style = new CSSStyleDeclaration(PropertySymbol.illegalConstructor, this[PropertySymbol.window]);
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
//# sourceMappingURL=CSSKeyframesRule.js.map