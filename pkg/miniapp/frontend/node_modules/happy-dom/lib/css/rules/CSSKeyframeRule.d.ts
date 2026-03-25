import CSSRule from '../CSSRule.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import CSSStyleDeclaration from '../declaration/CSSStyleDeclaration.js';
/**
 * CSSRule interface.
 */
export default class CSSKeyframeRule extends CSSRule {
    #private;
    readonly type: import("../CSSRuleTypeEnum.js").default;
    readonly keyText: string;
    [PropertySymbol.cssText]: string;
    /**
     * Returns style.
     *
     * @returns Style.
     */
    get style(): CSSStyleDeclaration;
    /**
     * Returns css text.
     *
     * @returns CSS text.
     */
    get cssText(): string;
}
//# sourceMappingURL=CSSKeyframeRule.d.ts.map