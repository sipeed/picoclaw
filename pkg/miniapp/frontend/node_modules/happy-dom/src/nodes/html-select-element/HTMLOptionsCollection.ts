import HTMLCollection from '../element/HTMLCollection.js';
import HTMLSelectElement from './HTMLSelectElement.js';
import HTMLOptionElement from '../html-option-element/HTMLOptionElement.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import QuerySelector from '../../query-selector/QuerySelector.js';

/**
 * HTML Options Collection.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/HTMLOptionsCollection.
 */
export default class HTMLOptionsCollection extends HTMLCollection<HTMLOptionElement> {
	private declare [PropertySymbol.ownerElement]: HTMLSelectElement;

	/**
	 * Constructor.
	 *
	 * @param illegalConstructorSymbol Illegal constructor symbol.
	 * @param ownerElement Select element.
	 */
	constructor(illegalConstructorSymbol: symbol, ownerElement: HTMLSelectElement) {
		super(
			illegalConstructorSymbol,
			() =>
				<HTMLOptionElement[]>(
					QuerySelector.querySelectorAll(ownerElement, 'option')[PropertySymbol.items]
				)
		);

		this[PropertySymbol.ownerElement] = ownerElement;
	}

	/**
	 * Returns selectedIndex.
	 *
	 * @returns SelectedIndex.
	 */
	public get selectedIndex(): number {
		return this[PropertySymbol.ownerElement].selectedIndex;
	}

	/**
	 * Sets selectedIndex.
	 *
	 * @param selectedIndex SelectedIndex.
	 */
	public set selectedIndex(selectedIndex: number) {
		this[PropertySymbol.ownerElement].selectedIndex = selectedIndex;
	}

	/**
	 *
	 * @param element
	 * @param before
	 */
	public add(element: HTMLOptionElement, before?: number | HTMLOptionElement): void {
		this[PropertySymbol.ownerElement].add(element, before);
	}

	/**
	 * Removes indexed element from collection.
	 *
	 * @param index Index.
	 */
	public remove(index: number): void {
		this[PropertySymbol.ownerElement].remove(index);
	}
}
