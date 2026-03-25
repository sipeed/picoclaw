import HTMLCollection from '../element/HTMLCollection.js';
import HTMLSelectElement from './HTMLSelectElement.js';
import HTMLOptionElement from '../html-option-element/HTMLOptionElement.js';
import * as PropertySymbol from '../../PropertySymbol.js';
/**
 * HTML Options Collection.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/HTMLOptionsCollection.
 */
export default class HTMLOptionsCollection extends HTMLCollection<HTMLOptionElement> {
    private [PropertySymbol.ownerElement];
    /**
     * Constructor.
     *
     * @param illegalConstructorSymbol Illegal constructor symbol.
     * @param ownerElement Select element.
     */
    constructor(illegalConstructorSymbol: symbol, ownerElement: HTMLSelectElement);
    /**
     * Returns selectedIndex.
     *
     * @returns SelectedIndex.
     */
    get selectedIndex(): number;
    /**
     * Sets selectedIndex.
     *
     * @param selectedIndex SelectedIndex.
     */
    set selectedIndex(selectedIndex: number);
    /**
     *
     * @param element
     * @param before
     */
    add(element: HTMLOptionElement, before?: number | HTMLOptionElement): void;
    /**
     * Removes indexed element from collection.
     *
     * @param index Index.
     */
    remove(index: number): void;
}
//# sourceMappingURL=HTMLOptionsCollection.d.ts.map