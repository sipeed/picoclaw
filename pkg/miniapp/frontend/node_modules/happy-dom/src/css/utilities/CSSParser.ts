import CSSRule from '../CSSRule.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import CSSStyleSheet from '../CSSStyleSheet.js';
import CSSStyleRule from '../rules/CSSStyleRule.js';
import CSSKeyframeRule from '../rules/CSSKeyframeRule.js';
import CSSKeyframesRule from '../rules/CSSKeyframesRule.js';
import CSSMediaRule from '../rules/CSSMediaRule.js';
import CSSContainerRule from '../rules/CSSContainerRule.js';
import CSSSupportsRule from '../rules/CSSSupportsRule.js';
import CSSFontFaceRule from '../rules/CSSFontFaceRule.js';
import SelectorParser from '../../query-selector/SelectorParser.js';

const COMMENT_REGEXP = /\/\*[\s\S]*?\*\//gm;

/**
 * CSS parser.
 */
export default class CSSParser {
	/**
	 * Parses HTML and returns a root element.
	 *
	 * @param parentStyleSheet Parent style sheet.
	 * @param cssText CSS code.
	 * @returns Root element.
	 */
	public static parseFromString(parentStyleSheet: CSSStyleSheet, cssText: string): CSSRule[] {
		const window = parentStyleSheet[PropertySymbol.window];
		const css = cssText.replace(COMMENT_REGEXP, '');
		const cssRules = [];
		const regExp = /{|}/gm;
		const stack: CSSRule[] = [];
		let parentRule: CSSRule = null;
		let lastIndex = 0;
		let match: RegExpMatchArray;

		while ((match = regExp.exec(css))) {
			if (match[0] === '{') {
				const selectorText = css.substring(lastIndex, match.index).trim();

				if (
					selectorText.startsWith('@keyframes') ||
					selectorText.startsWith('@-webkit-keyframes')
				) {
					const newRule = new CSSKeyframesRule(PropertySymbol.illegalConstructor, window);

					(<string>newRule.name) = selectorText.replace(/@(-webkit-){0,1}keyframes +/, '');
					newRule.parentStyleSheet = parentStyleSheet;
					cssRules.push(newRule);
					parentRule = newRule;
				} else if (selectorText.startsWith('@media')) {
					const mediums = selectorText.replace('@media', '').split(',');
					const newRule = new CSSMediaRule(PropertySymbol.illegalConstructor, window);

					for (const medium of mediums) {
						newRule.media.appendMedium(medium.trim());
					}

					newRule.parentStyleSheet = parentStyleSheet;
					cssRules.push(newRule);
					parentRule = newRule;
				} else if (
					selectorText.startsWith('@container') ||
					selectorText.startsWith('@-webkit-container')
				) {
					const conditionText = selectorText.replace(/@(-webkit-){0,1}container +/, '');
					const newRule = new CSSContainerRule(PropertySymbol.illegalConstructor, window);

					(<string>newRule.conditionText) = conditionText;
					newRule.parentStyleSheet = parentStyleSheet;
					cssRules.push(newRule);
					parentRule = newRule;
				} else if (
					selectorText.startsWith('@supports') ||
					selectorText.startsWith('@-webkit-supports')
				) {
					const conditionText = selectorText.replace(/@(-webkit-){0,1}supports +/, '');
					const newRule = new CSSSupportsRule(PropertySymbol.illegalConstructor, window);

					(<string>newRule.conditionText) = conditionText;
					newRule.parentStyleSheet = parentStyleSheet;
					cssRules.push(newRule);
					parentRule = newRule;
				} else if (selectorText.startsWith('@font-face')) {
					const conditionText = selectorText.replace('@font-face', '');
					const newRule = new CSSFontFaceRule(PropertySymbol.illegalConstructor, window);

					newRule[PropertySymbol.cssText] = conditionText;
					newRule.parentStyleSheet = parentStyleSheet;
					cssRules.push(newRule);
					parentRule = newRule;
				} else if (selectorText[0] === '@') {
					// Unknown rule.
					// We will create a new rule to let it grab its content, but we will not add it to the cssRules array.
					const newRule = new CSSRule(PropertySymbol.illegalConstructor, window);
					newRule.parentStyleSheet = parentStyleSheet;
					parentRule = newRule;
				} else if (parentRule && parentRule.type === CSSRule.KEYFRAMES_RULE) {
					const newRule = new CSSKeyframeRule(PropertySymbol.illegalConstructor, window);
					(<string>newRule.keyText) = selectorText.trim();
					newRule.parentStyleSheet = parentStyleSheet;
					newRule.parentRule = parentRule;

					(<CSSKeyframesRule>parentRule).cssRules.push(<CSSKeyframeRule>newRule);
					parentRule = newRule;
				} else if (
					parentRule &&
					(parentRule.type === CSSRule.MEDIA_RULE ||
						parentRule.type === CSSRule.CONTAINER_RULE ||
						parentRule.type === CSSRule.SUPPORTS_RULE)
				) {
					if (this.validateSelectorText(selectorText)) {
						const newRule = new CSSStyleRule(PropertySymbol.illegalConstructor, window);
						(<string>newRule.selectorText) = selectorText;
						newRule.parentStyleSheet = parentStyleSheet;
						newRule.parentRule = parentRule;
						(<CSSMediaRule>parentRule).cssRules.push(newRule);
						parentRule = newRule;
					}
				} else {
					if (this.validateSelectorText(selectorText)) {
						const newRule = new CSSStyleRule(PropertySymbol.illegalConstructor, window);
						(<string>newRule.selectorText) = selectorText;
						newRule.parentStyleSheet = parentStyleSheet;
						newRule.parentRule = parentRule;

						if (!parentRule) {
							cssRules.push(newRule);
						}

						parentRule = newRule;
					}
				}

				stack.push(parentRule);
			} else {
				if (parentRule) {
					const cssText = css
						.substring(lastIndex, match.index)
						.trim()
						.replace(/([^;])$/, '$1;'); // Ensure last semicolon
					switch (parentRule.type) {
						case CSSRule.FONT_FACE_RULE:
						case CSSRule.KEYFRAME_RULE:
						case CSSRule.STYLE_RULE:
							(<CSSStyleRule>parentRule)[PropertySymbol.cssText] = cssText;
							break;
					}
				}

				stack.pop();
				parentRule = stack[stack.length - 1] || null;
			}

			lastIndex = match.index + 1;
		}

		return cssRules;
	}

	/**
	 * Validates a selector text.
	 *
	 * @see https://www.w3.org/TR/CSS21/syndata.html#rule-sets
	 * @param selectorText Selector text.
	 * @returns True if valid, false otherwise.
	 */
	private static validateSelectorText(selectorText: string): boolean {
		try {
			SelectorParser.getSelectorGroups(selectorText);
		} catch (e) {
			return false;
		}
		return true;
	}
}
