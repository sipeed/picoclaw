import CSSStyleSheet from './CSSStyleSheet.js';
import CSSRuleTypeEnum from './CSSRuleTypeEnum.js';
import * as PropertySymbol from '../PropertySymbol.js';
import BrowserWindow from '../window/BrowserWindow.js';

/**
 * CSSRule interface.
 */
export default class CSSRule {
	// Static properties
	public static CONTAINER_RULE = CSSRuleTypeEnum.containerRule;
	public static STYLE_RULE = CSSRuleTypeEnum.styleRule;
	public static IMPORT_RULE = CSSRuleTypeEnum.importRule;
	public static MEDIA_RULE = CSSRuleTypeEnum.mediaRule;
	public static FONT_FACE_RULE = CSSRuleTypeEnum.fontFaceRule;
	public static PAGE_RULE = CSSRuleTypeEnum.pageRule;
	public static KEYFRAMES_RULE = CSSRuleTypeEnum.keyframesRule;
	public static KEYFRAME_RULE = CSSRuleTypeEnum.keyframeRule;
	public static NAMESPACE_RULE = CSSRuleTypeEnum.namespaceRule;
	public static COUNTER_STYLE_RULE = CSSRuleTypeEnum.counterStyleRule;
	public static SUPPORTS_RULE = CSSRuleTypeEnum.supportsRule;
	public static DOCUMENT_RULE = CSSRuleTypeEnum.documentRule;
	public static FONT_FEATURE_VALUES_RULE = CSSRuleTypeEnum.fontFeatureValuesRule;
	public static REGION_STYLE_RULE = CSSRuleTypeEnum.regionStyleRule;

	// Internal properties
	public [PropertySymbol.window]: BrowserWindow;

	// Public properties
	public parentRule: CSSRule = null;
	public parentStyleSheet: CSSStyleSheet = null;
	public type: number = null;

	/**
	 * Constructor.
	 *
	 * @param illegalConstructorSymbol Illegal constructor symbol.
	 * @param window Window.
	 */
	constructor(illegalConstructorSymbol: Symbol, window: BrowserWindow) {
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
	public get cssText(): string {
		return '';
	}
}
