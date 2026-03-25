import CSSRule from '../CSSRule.js';
import CSSStyleDeclaration from '../declaration/CSSStyleDeclaration.js';
import CSSKeyframeRule from './CSSKeyframeRule.js';
import * as PropertySymbol from '../../PropertySymbol.js';

const CSS_RULE_REGEXP = /([^{]+){([^}]+)}/;

/**
 * CSSRule interface.
 */
export default class CSSKeyframesRule extends CSSRule {
	public readonly type = CSSRule.KEYFRAMES_RULE;
	public readonly cssRules: CSSKeyframeRule[] = [];
	public readonly name: string = null;

	/**
	 * Returns css text.
	 *
	 * @returns CSS text.
	 */
	public get cssText(): string {
		let cssText = '';
		for (const cssRule of this.cssRules) {
			cssText += cssRule.cssText + ' ';
		}
		return `@keyframes ${this.name} { ${cssText}}`;
	}

	/**
	 * Appends a rule.
	 *
	 * @param rule Rule. E.g. "0% { transform: rotate(360deg); }".
	 */
	public appendRule(rule: string): void {
		const match = rule.match(CSS_RULE_REGEXP);
		if (match) {
			const cssRule = new CSSKeyframeRule(
				PropertySymbol.illegalConstructor,
				this[PropertySymbol.window]
			);
			const style = new CSSStyleDeclaration(
				PropertySymbol.illegalConstructor,
				this[PropertySymbol.window]
			);

			(<CSSRule>cssRule.parentRule) = this;
			(<string>cssRule.keyText) = match[1].trim();

			style.cssText = match[2].trim();

			(<CSSRule>style.parentRule) = this;
			(<CSSStyleDeclaration>cssRule.style) = style;
		}
	}

	/**
	 * Removes a rule.
	 *
	 * @param rule Rule. E.g. "0%".
	 */
	public deleteRule(rule: string): void {
		for (let i = 0, max = this.cssRules.length; i < max; i++) {
			if (this.cssRules[i].keyText === rule) {
				this.cssRules.splice(i, 1);
				break;
			}
		}
	}
}
