import CSSRuleTypeEnum from './CSSRuleTypeEnum.js';
import * as PropertySymbol from '../PropertySymbol.js';
/**
 * CSSRule interface.
 */
export default class CSSRule {
    // Static properties
    static CONTAINER_RULE = CSSRuleTypeEnum.containerRule;
    static STYLE_RULE = CSSRuleTypeEnum.styleRule;
    static IMPORT_RULE = CSSRuleTypeEnum.importRule;
    static MEDIA_RULE = CSSRuleTypeEnum.mediaRule;
    static FONT_FACE_RULE = CSSRuleTypeEnum.fontFaceRule;
    static PAGE_RULE = CSSRuleTypeEnum.pageRule;
    static KEYFRAMES_RULE = CSSRuleTypeEnum.keyframesRule;
    static KEYFRAME_RULE = CSSRuleTypeEnum.keyframeRule;
    static NAMESPACE_RULE = CSSRuleTypeEnum.namespaceRule;
    static COUNTER_STYLE_RULE = CSSRuleTypeEnum.counterStyleRule;
    static SUPPORTS_RULE = CSSRuleTypeEnum.supportsRule;
    static DOCUMENT_RULE = CSSRuleTypeEnum.documentRule;
    static FONT_FEATURE_VALUES_RULE = CSSRuleTypeEnum.fontFeatureValuesRule;
    static REGION_STYLE_RULE = CSSRuleTypeEnum.regionStyleRule;
    // Internal properties
    [PropertySymbol.window];
    // Public properties
    parentRule = null;
    parentStyleSheet = null;
    type = null;
    /**
     * Constructor.
     *
     * @param illegalConstructorSymbol Illegal constructor symbol.
     * @param window Window.
     */
    constructor(illegalConstructorSymbol, window) {
        if (illegalConstructorSymbol !== PropertySymbol.illegalConstructor) {
            throw new TypeError('Illegal constructor');
        }
        this[PropertySymbol.window] = window;
    }
    /**
     * Returns selector text.
     *
     * @returns Selector text.
     */
    get cssText() {
        return '';
    }
}
//# sourceMappingURL=CSSRule.js.map