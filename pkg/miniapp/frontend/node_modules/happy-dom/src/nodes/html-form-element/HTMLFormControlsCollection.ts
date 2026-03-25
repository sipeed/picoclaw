import * as PropertySymbol from '../../PropertySymbol.js';
import HTMLCollection from '../element/HTMLCollection.js';
import HTMLFormElement from './HTMLFormElement.js';
import RadioNodeList from './RadioNodeList.js';
import THTMLFormControlElement from './THTMLFormControlElement.js';

/**
 * HTMLFormControlsCollection.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/HTMLFormControlsCollection
 */
export default class HTMLFormControlsCollection extends HTMLCollection<
	THTMLFormControlElement,
	THTMLFormControlElement | RadioNodeList
> {
	private declare [PropertySymbol.ownerElement]: HTMLFormElement;

	/**
	 * Constructor.
	 *
	 * @param illegalConstructorSymbol Illegal constructor symbol.
	 * @param ownerElement Form element.
	 */
	constructor(illegalConstructorSymbol: symbol, ownerElement: HTMLFormElement) {
		super(illegalConstructorSymbol, () => ownerElement[PropertySymbol.getFormControlItems]());
		this[PropertySymbol.ownerElement] = ownerElement;
	}

	/**
	 * @override
	 */
	public namedItem(name: string): THTMLFormControlElement | RadioNodeList | null {
		return this[PropertySymbol.ownerElement][PropertySymbol.getFormControlNamedItem](name);
	}
}
