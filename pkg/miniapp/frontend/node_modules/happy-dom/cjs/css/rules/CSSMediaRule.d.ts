import CSSRule from '../CSSRule.cjs';
import MediaList from '../MediaList.cjs';
/**
 * CSSRule interface.
 */
export default class CSSMediaRule extends CSSRule {
    readonly type: import("../CSSRuleTypeEnum.cjs").default;
    readonly cssRules: CSSRule[];
    readonly media: MediaList;
    /**
     * Returns css text.
     *
     * @returns CSS text.
     */
    get cssText(): string;
    /**
     * Returns conditional text.
     *
     * @returns Conditional text.
     */
    get conditionText(): string;
}
//# sourceMappingURL=CSSMediaRule.d.ts.map