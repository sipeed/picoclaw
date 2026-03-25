import CSSRule from '../CSSRule.js';

/**
 * CSSRule interface.
 */
export default class CSSContainerRule extends CSSRule {
	public readonly type = CSSRule.CONTAINER_RULE;
	public readonly cssRules: CSSRule[] = [];
	public readonly conditionText = '';

	/**
	 * Returns css text.
	 *
	 * @returns CSS text.
	 */
	public get cssText(): string {
		let cssText = '';
		for (const cssRule of this.cssRules) {
			cssText += cssRule.cssText;
		}
		return `@container ${this.conditionText} { ${cssText} }`;
	}
}
