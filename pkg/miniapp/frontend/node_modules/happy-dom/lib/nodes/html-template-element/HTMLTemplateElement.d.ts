import HTMLElement from '../html-element/HTMLElement.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import DocumentFragment from '../document-fragment/DocumentFragment.js';
import Node from '../node/Node.js';
import ShadowRoot from '../shadow-root/ShadowRoot.js';
/**
 * HTML Template Element.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/HTMLTemplateElement.
 */
export default class HTMLTemplateElement extends HTMLElement {
    cloneNode: (deep?: boolean) => HTMLTemplateElement;
    [PropertySymbol.content]: DocumentFragment;
    /**
     * Returns content.
     *
     * @returns Content.
     */
    get content(): DocumentFragment;
    /**
     * @override
     */
    get innerHTML(): string;
    /**
     * @override
     */
    set innerHTML(html: string);
    /**
     * @override
     */
    get firstChild(): Node;
    /**
     * @override
     */
    get lastChild(): Node;
    /**
     * @deprecated
     * @override
     */
    getInnerHTML(_options?: {
        includeShadowRoots?: boolean;
    }): string;
    /**
     * @override
     */
    getHTML(_options?: {
        serializableShadowRoots?: boolean;
        shadowRoots?: ShadowRoot[];
    }): string;
    /**
     * @override
     */
    [PropertySymbol.appendChild](node: Node, disableValidations?: boolean): Node;
    /**
     * @override
     */
    [PropertySymbol.removeChild](node: Node): Node;
    /**
     * @override
     */
    [PropertySymbol.insertBefore](newNode: Node, referenceNode: Node, disableValidations?: boolean): Node;
    /**
     * @override
     */
    [PropertySymbol.replaceChild](newChild: Node, oldChild: Node): Node;
    /**
     * @override
     */
    [PropertySymbol.cloneNode](deep?: boolean): HTMLTemplateElement;
}
//# sourceMappingURL=HTMLTemplateElement.d.ts.map