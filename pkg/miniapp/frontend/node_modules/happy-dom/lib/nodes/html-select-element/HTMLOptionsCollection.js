import HTMLCollection from '../element/HTMLCollection.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import QuerySelector from '../../query-selector/QuerySelector.js';
/**
 * HTML Options Collection.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/HTMLOptionsCollection.
 */
export default class HTMLOptionsCollection extends HTMLCollection {
    /**
     * Constructor.
     *
     * @param illegalConstructorSymbol Illegal constructor symbol.
     * @param ownerElement Select element.
     */
    constructor(illegalConstructorSymbol, ownerElement) {
        super(illegalConstructorSymbol, () => (QuerySelector.querySelectorAll(ownerElement, 'option')[PropertySymbol.items]));
        this[PropertySymbol.ownerElement] = ownerElement;
    }
    /**
     * Returns selectedIndex.
     *
     * @returns SelectedIndex.
     */
    get selectedIndex() {
        return this[PropertySymbol.ownerElement].selectedIndex;
    }
    /**
     * Sets selectedIndex.
     *
     * @param selectedIndex SelectedIndex.
     */
    set selectedIndex(selectedIndex) {
        this[PropertySymbol.ownerElement].selectedIndex = selectedIndex;
    }
    /**
     *
     * @param element
     * @param before
     */
    add(element, before) {
        this[PropertySymbol.ownerElement].add(element, before);
    }
    /**
     * Removes indexed element from collection.
     *
     * @param index Index.
     */
    remove(index) {
        this[PropertySymbol.ownerElement].remove(index);
    }
}
//# sourceMappingURL=HTMLOptionsCollection.js.map