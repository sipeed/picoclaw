import CSSStyleSheet from './CSSStyleSheet.cjs';
import CSSRuleTypeEnum from './CSSRuleTypeEnum.cjs';
import * as PropertySymbol from '../PropertySymbol.cjs';
import BrowserWindow from '../window/BrowserWindow.cjs';
/**
 * CSSRule interface.
 */
export default class CSSRule {
    static CONTAINER_RULE: CSSRuleTypeEnum;
    static STYLE_RULE: CSSRuleTypeEnum;
    static IMPORT_RULE: CSSRuleTypeEnum;
    static MEDIA_RULE: CSSRuleTypeEnum;
    static FONT_FACE_RULE: CSSRuleTypeEnum;
    static PAGE_RULE: CSSRuleTypeEnum;
    static KEYFRAMES_RULE: CSSRuleTypeEnum;
    static KEYFRAME_RULE: CSSRuleTypeEnum;
    static NAMESPACE_RULE: CSSRuleTypeEnum;
    static COUNTER_STYLE_RULE: CSSRuleTypeEnum;
    static SUPPORTS_RULE: CSSRuleTypeEnum;
    static DOCUMENT_RULE: CSSRuleTypeEnum;
    static FONT_FEATURE_VALUES_RULE: CSSRuleTypeEnum;
    static REGION_STYLE_RULE: CSSRuleTypeEnum;
    [PropertySymbol.window]: BrowserWindow;
    parentRule: CSSRule;
    parentStyleSheet: CSSStyleSheet;
    type: number;
    /**
     * Constructor.
     *
     * @param illegalConstructorSymbol Illegal constructor symbol.
     * @param window Window.
     */
    constructor(illegalConstructorSymbol: Symbol, window: BrowserWindow);
    /**
     * Returns selector text.
     *
     * @returns Selector text.
     */
    get cssText(): string;
}
//# sourceMappingURL=CSSRule.d.ts.map