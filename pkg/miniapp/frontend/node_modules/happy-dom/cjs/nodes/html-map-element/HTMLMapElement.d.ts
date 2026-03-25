import HTMLCollection from '../element/HTMLCollection.cjs';
import HTMLElement from '../html-element/HTMLElement.cjs';
import * as PropertySymbol from '../../PropertySymbol.cjs';
import HTMLAreaElement from '../html-area-element/HTMLAreaElement.cjs';
/**
 * HTMLMapElement
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/HTMLMapElement
 */
export default class HTMLMapElement extends HTMLElement {
    [PropertySymbol.areas]: HTMLCollection<HTMLAreaElement> | null;
    /**
     * Returns areas.
     *
     * @returns Areas.
     */
    get areas(): HTMLCollection<HTMLAreaElement>;
    /**
     * Returns name.
     *
     * @returns Name.
     */
    get name(): string;
    /**
     * Sets name.
     *
     * @param name Name.
     */
    set name(name: string);
}
//# sourceMappingURL=HTMLMapElement.d.ts.map