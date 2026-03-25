import CSSRule from '../CSSRule.js';
/**
 * CSSRule interface.
 */
export default class CSSContainerRule extends CSSRule {
    readonly type: import("../CSSRuleTypeEnum.js").default;
    readonly cssRules: CSSRule[];
    readonly conditionText = "";
    /**
     * Returns css text.
     *
     * @returns CSS text.
     */
    get cssText(): string;
}
//# sourceMappingURL=CSSContainerRule.d.ts.map