import DocumentFragment from '../document-fragment/DocumentFragment.js';
import Document from '../document/Document.js';
import Element from '../element/Element.js';
import Node from '../node/Node.js';
import HTMLCollection from '../element/HTMLCollection.js';
/**
 * Parent node utility.
 */
export default class ParentNodeUtility {
    /**
     * Inserts a set of Node objects or DOMString objects after the last child of the ParentNode. DOMString objects are inserted as equivalent Text nodes.
     *
     * @param parentNode Parent node.
     * @param nodes List of Node or DOMString.
     */
    static append(parentNode: Element | Document | DocumentFragment, ...nodes: any[]): void;
    /**
     * Inserts a set of Node objects or DOMString objects before the first child of the ParentNode. DOMString objects are inserted as equivalent Text nodes.
     *
     * @param parentNode Parent node.
     * @param nodes List of Node or DOMString.
     */
    static prepend(parentNode: Element | Document | DocumentFragment, ...nodes: (string | Node)[]): void;
    /**
     * Replaces the existing children of a ParentNode with a specified new set of children.
     *
     * @param parentNode Parent node.
     * @param nodes List of Node or DOMString.
     */
    static replaceChildren(parentNode: Element | Document | DocumentFragment, ...nodes: (string | Node)[]): void;
    /**
     * Returns an elements by class name.
     *
     * @param parentNode Parent node.
     * @param className Tag name.
     * @returns Matching element.
     */
    static getElementsByClassName(parentNode: Element | DocumentFragment | Document, className: string): HTMLCollection<Element>;
    /**
     * Returns an elements by tag name.
     *
     * @param parentNode Parent node.
     * @param tagName Tag name.
     * @returns Matching element.
     */
    static getElementsByTagName<T extends Element = Element>(parentNode: Element | DocumentFragment | Document, tagName: string): HTMLCollection<T>;
    /**
     * Returns an elements by tag name and namespace.
     *
     * @param parentNode Parent node.
     * @param namespaceURI Namespace URI.
     * @param tagName Tag name.
     * @returns Matching element.
     */
    static getElementsByTagNameNS(parentNode: Element | DocumentFragment | Document, namespaceURI: string, tagName: string): HTMLCollection<Element>;
    /**
     * Returns the first element matching a tag name.
     * This is not part of the browser standard and is only used internally (used in Document).
     *
     * @param parentNode Parent node.
     * @param tagName Tag name.
     * @returns Matching element.
     */
    static getElementByTagName(parentNode: Element | DocumentFragment | Document, tagName: string): Element;
    /**
     * Returns an element by ID.
     *
     * @param parentNode Parent node.
     * @param id ID.
     * @returns Matching element.
     */
    static getElementById(parentNode: Element | DocumentFragment | Document, id: string): Element | null;
}
//# sourceMappingURL=ParentNodeUtility.d.ts.map