import CSSRule from '../CSSRule.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import CSSStyleDeclaration from '../declaration/CSSStyleDeclaration.js';
/**
 * CSSRule interface.
 */
export default class CSSFontFaceRule extends CSSRule {
    type = CSSRule.FONT_FACE_RULE;
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
        return `@font-face { ${this.style.cssText} }`;
    }
}
//# sourceMappingURL=CSSFontFaceRule.js.map