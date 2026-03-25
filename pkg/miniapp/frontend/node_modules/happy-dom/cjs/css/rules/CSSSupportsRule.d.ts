import CSSRule from '../CSSRule.cjs';
/**
 * CSSRule interface.
 */
export default class CSSSupportsRule extends CSSRule {
    readonly type: import("../CSSRuleTypeEnum.cjs").default;
    readonly cssRules: CSSRule[];
    readonly conditionText = "";
    /**
     * Returns css text.
     *
     * @returns CSS text.
     */
    get cssText(): string;
}
//# sourceMappingURL=CSSSupportsRule.d.ts.map