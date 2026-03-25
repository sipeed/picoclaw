import HTMLElement from '../html-element/HTMLElement.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import ParentNodeUtility from '../parent-node/ParentNodeUtility.js';
/**
 * HTMLMapElement
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/HTMLMapElement
 */
export default class HTMLMapElement extends HTMLElement {
    [PropertySymbol.areas] = null;
    /**
     * Returns areas.
     *
     * @returns Areas.
     */
    get areas() {
        if (!this[PropertySymbol.areas]) {
            this[PropertySymbol.areas] = ParentNodeUtility.getElementsByTagName(this, 'area');
        }
        return this[PropertySymbol.areas];
    }
    /**
     * Returns name.
     *
     * @returns Name.
     */
    get name() {
        return this.getAttribute('name') || '';
    }
    /**
     * Sets name.
     *
     * @param name Name.
     */
    set name(name) {
        this.setAttribute('name', name);
    }
}
//# sourceMappingURL=HTMLMapElement.js.map