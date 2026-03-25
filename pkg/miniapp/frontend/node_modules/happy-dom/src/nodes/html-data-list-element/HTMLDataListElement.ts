import HTMLElement from '../html-element/HTMLElement.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import HTMLCollection from '../element/HTMLCollection.js';
import HTMLOptionElement from '../html-option-element/HTMLOptionElement.js';
import ParentNodeUtility from '../parent-node/ParentNodeUtility.js';

/**
 * HTMLDataListElement
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/HTMLDataListElement
 */
export default class HTMLDataListElement extends HTMLElement {
	public [PropertySymbol.options]: HTMLCollection<HTMLOptionElement> | null = null;

	/**
	 * Returns options.
	 *
	 * @returns Options.
	 */
	public get options(): HTMLCollection<HTMLOptionElement> {
		if (!this[PropertySymbol.options]) {
			this[PropertySymbol.options] = <HTMLCollection<HTMLOptionElement>>(
				ParentNodeUtility.getElementsByTagName(this, 'OPTION')
			);
		}
		return this[PropertySymbol.options];
	}
}
