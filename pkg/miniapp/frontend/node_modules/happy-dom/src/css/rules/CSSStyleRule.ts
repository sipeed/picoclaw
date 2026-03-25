import CSSRule from '../CSSRule.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import CSSStyleDeclaration from '../declaration/CSSStyleDeclaration.js';

/**
 * CSSRule interface.
 */
export default class CSSStyleRule extends CSSRule {
	public readonly type = CSSRule.STYLE_RULE;
	public readonly styleMap = new Map();
	public selectorText = '';
	public [PropertySymbol.cssText] = '';
	#style: CSSStyleDeclaration | null = null;

	/**
	 * Returns style.
	 *
	 * @returns Style.
	 */
	public get style(): CSSStyleDeclaration {
		if (!this.#style) {
			this.#style = new CSSStyleDeclaration(
				PropertySymbol.illegalConstructor,
				this[PropertySymbol.window]
			);
			(<CSSRule>this.#style.parentRule) = this;
			this.#style.cssText = this[PropertySymbol.cssText];
		}
		return this.#style;
	}

	/**
	 * Returns css text.
	 *
	 * @returns CSS text.
	 */
	public get cssText(): string {
		return `${this.selectorText} { ${this.style.cssText} }`;
	}
}
