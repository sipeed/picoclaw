import Node from '../node/Node.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import QuerySelector from '../../query-selector/QuerySelector.js';
import ParentNodeUtility from '../parent-node/ParentNodeUtility.js';
import HTMLCollection from '../element/HTMLCollection.js';
import NodeTypeEnum from '../node/NodeTypeEnum.js';
/**
 * DocumentFragment.
 */
export default class DocumentFragment extends Node {
    [PropertySymbol.children] = null;
    [PropertySymbol.rootNode] = this;
    [PropertySymbol.nodeType] = NodeTypeEnum.documentFragmentNode;
    /**
     * Returns the document fragment children.
     */
    get children() {
        if (!this[PropertySymbol.children]) {
            const elements = this[PropertySymbol.elementArray];
            this[PropertySymbol.children] = new HTMLCollection(PropertySymbol.illegalConstructor, () => elements);
        }
        return this[PropertySymbol.children];
    }
    /**
     * Last element child.
     *
     * @returns Element.
     */
    get childElementCount() {
        return this[PropertySymbol.elementArray].length;
    }
    /**
     * First element child.
     *
     * @returns Element.
     */
    get firstElementChild() {
        return this[PropertySymbol.elementArray][0] ?? null;
    }
    /**
     * Last element child.
     *
     * @returns Element.
     */
    get lastElementChild() {
        const children = this[PropertySymbol.elementArray];
        return children[children.length - 1] ?? null;
    }
    /**
     * Get text value of children.
     *
     * @returns Text content.
     */
    get textContent() {
        let result = '';
        for (const childNode of this[PropertySymbol.nodeArray]) {
            if (childNode[PropertySymbol.nodeType] === NodeTypeEnum.elementNode ||
                childNode[PropertySymbol.nodeType] === NodeTypeEnum.textNode) {
                result += childNode.textContent;
            }
        }
        return result;
    }
    /**
     * Sets text content.
     *
     * @param textContent Text content.
     */
    set textContent(textContent) {
        const childNodes = this[PropertySymbol.nodeArray];
        while (childNodes.length) {
            this.removeChild(childNodes[0]);
        }
        if (textContent) {
            this.appendChild(this[PropertySymbol.ownerDocument].createTextNode(textContent));
        }
    }
    /**
     * Inserts a set of Node objects or DOMString objects after the last child of the ParentNode. DOMString objects are inserted as equivalent Text nodes.
     *
     * @param nodes List of Node or DOMString.
     */
    append(...nodes) {
        ParentNodeUtility.append(this, ...nodes);
    }
    /**
     * Inserts a set of Node objects or DOMString objects before the first child of the ParentNode. DOMString objects are inserted as equivalent Text nodes.
     *
     * @param nodes List of Node or DOMString.
     */
    prepend(...nodes) {
        ParentNodeUtility.prepend(this, ...nodes);
    }
    /**
     * Replaces the existing children of a node with a specified new set of children.
     *
     * @param nodes List of Node or DOMString.
     */
    replaceChildren(...nodes) {
        ParentNodeUtility.replaceChildren(this, ...nodes);
    }
    /**
     * Query CSS selector to find matching elments.
     *
     * @param selector CSS selector.
     * @returns Matching elements.
     */
    querySelectorAll(selector) {
        return QuerySelector.querySelectorAll(this, selector);
    }
    /**
     * Query CSS Selector to find a matching element.
     *
     * @param selector CSS selector.
     * @returns Matching element.
     */
    querySelector(selector) {
        return QuerySelector.querySelector(this, selector);
    }
    /**
     * Returns an element by ID.
     *
     * @param id ID.
     * @returns Matching element.
     */
    getElementById(id) {
        return ParentNodeUtility.getElementById(this, id);
    }
}
//# sourceMappingURL=DocumentFragment.js.map