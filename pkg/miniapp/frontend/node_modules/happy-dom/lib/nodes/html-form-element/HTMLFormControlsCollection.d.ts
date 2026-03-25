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
export default class HTMLFormControlsCollection extends HTMLCollection<THTMLFormControlElement, THTMLFormControlElement | RadioNodeList> {
    private [PropertySymbol.ownerElement];
    /**
     * Constructor.
     *
     * @param illegalConstructorSymbol Illegal constructor symbol.
     * @param ownerElement Form element.
     */
    constructor(illegalConstructorSymbol: symbol, ownerElement: HTMLFormElement);
    /**
     * @override
     */
    namedItem(name: string): THTMLFormControlElement | RadioNodeList | null;
}
//# sourceMappingURL=HTMLFormControlsCollection.d.ts.map