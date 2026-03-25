import CSSRule from '../CSSRule.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import CSSStyleDeclaration from '../declaration/CSSStyleDeclaration.js';
/**
 * CSSRule interface.
 */
export default class CSSKeyframeRule extends CSSRule {
    type = CSSRule.KEYFRAME_RULE;
    keyText;
    [PropertySymbol.cssText] = '';
    #style = null;
    /**
     * Returns style.
     *
     * @returns Style.
     */
    get style() {
        if (!this.#style) {
            this.#style = new CSSStyleDeclaration(PropertySymbol.illegalConstructor, this[PropertySymbol.window]);
            this.#style.parentRule = this;
            this.#style.cssText = this[PropertySymbol.cssText];
        }
        return this.#style;
    }
    /**
     * Returns css text.
     *
     * @returns CSS text.
     */
    get cssText() {
        return `${this.keyText} { ${this.style.cssText} }`;
    }
}
//# sourceMappingURL=CSSKeyframeRule.js.map