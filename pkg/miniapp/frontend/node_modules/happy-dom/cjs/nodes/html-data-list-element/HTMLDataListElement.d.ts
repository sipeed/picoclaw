import HTMLElement from '../html-element/HTMLElement.cjs';
import * as PropertySymbol from '../../PropertySymbol.cjs';
import HTMLCollection from '../element/HTMLCollection.cjs';
import HTMLOptionElement from '../html-option-element/HTMLOptionElement.cjs';
/**
 * HTMLDataListElement
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/HTMLDataListElement
 */
export default class HTMLDataListElement extends HTMLElement {
    [PropertySymbol.options]: HTMLCollection<HTMLOptionElement> | null;
    /**
     * Returns options.
     *
     * @returns Options.
     */
    get options(): HTMLCollection<HTMLOptionElement>;
}
//# sourceMappingURL=HTMLDataListElement.d.ts.map