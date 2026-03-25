import HTMLElement from '../html-element/HTMLElement.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import ParentNodeUtility from '../parent-node/ParentNodeUtility.js';
/**
 * HTMLDataListElement
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/HTMLDataListElement
 */
export default class HTMLDataListElement extends HTMLElement {
    [PropertySymbol.options] = null;
    /**
     * Returns options.
     *
     * @returns Options.
     */
    get options() {
        if (!this[PropertySymbol.options]) {
            this[PropertySymbol.options] = (ParentNodeUtility.getElementsByTagName(this, 'OPTION'));
        }
        return this[PropertySymbol.options];
    }
}
//# sourceMappingURL=HTMLDataListElement.js.map