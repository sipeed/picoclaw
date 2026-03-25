import CSSRule from '../CSSRule.js';
/**
 * CSSRule interface.
 */
export default class CSSSupportsRule extends CSSRule {
    type = CSSRule.SUPPORTS_RULE;
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
//# sourceMappingURL=CSSSupportsRule.js.map