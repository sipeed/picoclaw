import DOMExceptionNameEnum from '../exception/DOMExceptionNameEnum.js';
import CSSParser from './utilities/CSSParser.js';
import CSSRule from './CSSRule.js';
import MediaList from './MediaList.js';
import BrowserWindow from '../window/BrowserWindow.js';
import * as PropertySymbol from '../PropertySymbol.js';

/**
 * CSS StyleSheet.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/CSSStyleSheet.
 */
export default class CSSStyleSheet {
	// Injected by WindowContextClassExtender
	protected declare [PropertySymbol.window]: BrowserWindow;

	public value: string = null;
	public name: string = null;
	public namespaceURI: string = null;
	public readonly cssRules: CSSRule[] = [];

	// TODO: MediaList is not fully implemented.
	public media: MediaList | string;
	public title: string;
	public alternate: boolean;
	public disabled: boolean;
	#currentText: string = null;

	/**
	 * Constructor.
	 *
	 * @param [options] Options.
	 * @param [options.media] Media.
	 * @param [options.title] Title.
	 * @param [options.alternate] Alternate.
	 * @param [options.disabled] Disabled.
	 */
	constructor(options?: {
		media?: MediaList | string;
		title?: string;
		alternate?: boolean;
		disabled?: boolean;
	}) {
		this.media = options && options.media ? options.media : '';
		this.title = options && options.title ? options.title : '';
		this.alternate = options && options.alternate ? options.alternate : false;
		this.disabled = options && options.disabled ? options.disabled : false;
	}

	/**
	 * Inserts a rule.
	 *
	 * @see https://developer.mozilla.org/en-US/docs/Web/API/CSSStyleSheet/insertRule
	 * @param rule Rule.
	 * @param [index] Index.
	 * @returns The newly inserterted rule's index.
	 */
	public insertRule(rule: string, index?: number): number {
		const rules = CSSParser.parseFromString(this, rule);

		if (rules.length === 0) {
			throw new this[PropertySymbol.window].DOMException(
				'Invalid CSS rule.',
				DOMExceptionNameEnum.hierarchyRequestError
			);
		}

		if (rules.length > 1) {
			throw new this[PropertySymbol.window].DOMException(
				'Only one rule is allowed to be added.',
				DOMExceptionNameEnum.syntaxError
			);
		}

		if (index !== undefined) {
			if (index > this.cssRules.length) {
				throw new this[PropertySymbol.window].DOMException(
					'Index is more than the length of CSSRuleList.',
					DOMExceptionNameEnum.indexSizeError
				);
			}
			this.cssRules.splice(index, 0, rules[0]);
			return index;
		}

		const newIndex = this.cssRules.length;

		this.cssRules.push(rules[0]);

		return newIndex;
	}

	/**
	 * Removes a rule.
	 *
	 * @see https://developer.mozilla.org/en-US/docs/Web/API/CSSStyleSheet/deleteRule
	 * @param index Index.
	 */
	public deleteRule(index: number): void {
		delete this.cssRules[index];
	}

	/**
	 * Replaces all CSS rules.
	 *
	 * @see https://developer.mozilla.org/en-US/docs/Web/API/CSSStyleSheet/replace
	 * @param text CSS text.
	 * @returns Promise.
	 */
	public async replace(text: string): Promise<void> {
		this.replaceSync(text);
	}

	/**
	 * Replaces all CSS rules.
	 *
	 * @see https://developer.mozilla.org/en-US/docs/Web/API/CSSStyleSheet/replaceSync
	 * @param text CSS text.
	 */
	public replaceSync(text: string): void {
		if (this.#currentText !== text) {
			this.#currentText = text;
			(<CSSRule[]>this.cssRules) = CSSParser.parseFromString(this, text);
		}
	}
}
