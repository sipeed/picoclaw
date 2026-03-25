import CSSRule from '../CSSRule.js';
import MediaList from '../MediaList.js';

/**
 * CSSRule interface.
 */
export default class CSSMediaRule extends CSSRule {
	public readonly type = CSSRule.MEDIA_RULE;
	public readonly cssRules: CSSRule[] = [];
	public readonly media = new MediaList();

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
		return `@media ${this.conditionText} { ${cssText} }`;
	}

	/**
	 * Returns conditional text.
	 *
	 * @returns Conditional text.
	 */
	public get conditionText(): string {
		return this.media.mediaText;
	}
}
