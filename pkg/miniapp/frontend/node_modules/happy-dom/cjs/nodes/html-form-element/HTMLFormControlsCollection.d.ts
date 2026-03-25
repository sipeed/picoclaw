import * as PropertySymbol from '../../PropertySymbol.cjs';
import HTMLCollection from '../element/HTMLCollection.cjs';
import HTMLFormElement from './HTMLFormElement.cjs';
import RadioNodeList from './RadioNodeList.cjs';
import THTMLFormControlElement from './THTMLFormControlElement.cjs';
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