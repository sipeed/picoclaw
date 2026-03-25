import * as PropertySymbol from '../../PropertySymbol.js';
import HTMLCollection from '../element/HTMLCollection.js';
/**
 * HTMLFormControlsCollection.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/HTMLFormControlsCollection
 */
export default class HTMLFormControlsCollection extends HTMLCollection {
    /**
     * Constructor.
     *
     * @param illegalConstructorSymbol Illegal constructor symbol.
     * @param ownerElement Form element.
     */
    constructor(illegalConstructorSymbol, ownerElement) {
        super(illegalConstructorSymbol, () => ownerElement[PropertySymbol.getFormControlItems]());
        this[PropertySymbol.ownerElement] = ownerElement;
    }
    /**
     * @override
     */
    namedItem(name) {
        return this[PropertySymbol.ownerElement][PropertySymbol.getFormControlNamedItem](name);
    }
}
//# sourceMappingURL=HTMLFormControlsCollection.js.map