"use strict";
var __createBinding = (this && this.__createBinding) || (Object.create ? (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    var desc = Object.getOwnPropertyDescriptor(m, k);
    if (!desc || ("get" in desc ? !m.__esModule : desc.writable || desc.configurable)) {
      desc = { enumerable: true, get: function() { return m[k]; } };
    }
    Object.defineProperty(o, k2, desc);
}) : (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    o[k2] = m[k];
}));
var __setModuleDefault = (this && this.__setModuleDefault) || (Object.create ? (function(o, v) {
    Object.defineProperty(o, "default", { enumerable: true, value: v });
}) : function(o, v) {
    o["default"] = v;
});
var __importStar = (this && this.__importStar) || function (mod) {
    if (mod && mod.__esModule) return mod;
    var result = {};
    if (mod != null) for (var k in mod) if (k !== "default" && Object.prototype.hasOwnProperty.call(mod, k)) __createBinding(result, mod, k);
    __setModuleDefault(result, mod);
    return result;
};
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const CSSRuleTypeEnum_js_1 = __importDefault(require("./CSSRuleTypeEnum.cjs"));
const PropertySymbol = __importStar(require("../PropertySymbol.cjs"));
/**
 * CSSRule interface.
 */
class CSSRule {
    // Static properties
    static CONTAINER_RULE = CSSRuleTypeEnum_js_1.default.containerRule;
    static STYLE_RULE = CSSRuleTypeEnum_js_1.default.styleRule;
    static IMPORT_RULE = CSSRuleTypeEnum_js_1.default.importRule;
    static MEDIA_RULE = CSSRuleTypeEnum_js_1.default.mediaRule;
    static FONT_FACE_RULE = CSSRuleTypeEnum_js_1.default.fontFaceRule;
    static PAGE_RULE = CSSRuleTypeEnum_js_1.default.pageRule;
    static KEYFRAMES_RULE = CSSRuleTypeEnum_js_1.default.keyframesRule;
    static KEYFRAME_RULE = CSSRuleTypeEnum_js_1.default.keyframeRule;
    static NAMESPACE_RULE = CSSRuleTypeEnum_js_1.default.namespaceRule;
    static COUNTER_STYLE_RULE = CSSRuleTypeEnum_js_1.default.counterStyleRule;
    static SUPPORTS_RULE = CSSRuleTypeEnum_js_1.default.supportsRule;
    static DOCUMENT_RULE = CSSRuleTypeEnum_js_1.default.documentRule;
    static FONT_FEATURE_VALUES_RULE = CSSRuleTypeEnum_js_1.default.fontFeatureValuesRule;
    static REGION_STYLE_RULE = CSSRuleTypeEnum_js_1.default.regionStyleRule;
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
exports.default = CSSRule;
//# sourceMappingURL=CSSRule.cjs.map