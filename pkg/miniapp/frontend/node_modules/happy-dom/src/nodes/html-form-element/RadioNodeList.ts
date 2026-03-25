import HTMLInputElement from '../html-input-element/HTMLInputElement.js';
import NodeList from '../node/NodeList.js';
import THTMLFormControlElement from './THTMLFormControlElement.js';
import * as PropertySymbol from '../../PropertySymbol.js';

/**
 * RadioNodeList
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/RadioNodeList
 */
export default class RadioNodeList extends NodeList<THTMLFormControlElement> {
	/**
	 * Returns value.
	 *
	 * @returns Value.
	 */
	public get value(): string {
		for (const node of this[PropertySymbol.items]) {
			if ((<HTMLInputElement>node).checked) {
				return (<HTMLInputElement>node).value;
			}
		}
		return null;
	}
}
