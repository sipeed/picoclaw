import * as PropertySymbol from '../../PropertySymbol.js';
import Document from '../document/Document.js';
import Node from '../node/Node.js';
import NamespaceURI from '../../config/NamespaceURI.js';
import HTMLCollection from '../element/HTMLCollection.js';
import QuerySelector from '../../query-selector/QuerySelector.js';
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
    static append(parentNode, ...nodes) {
        for (const node of nodes) {
            if (node instanceof Node) {
                parentNode.appendChild(node);
            }
            else {
                parentNode.appendChild(parentNode[PropertySymbol.ownerDocument].createTextNode(String(node)));
            }
        }
    }
    /**
     * Inserts a set of Node objects or DOMString objects before the first child of the ParentNode. DOMString objects are inserted as equivalent Text nodes.
     *
     * @param parentNode Parent node.
     * @param nodes List of Node or DOMString.
     */
    static prepend(parentNode, ...nodes) {
        const firstChild = parentNode.firstChild;
        for (const node of nodes) {
            if (node instanceof Node) {
                parentNode.insertBefore(node, firstChild);
            }
            else {
                parentNode.insertBefore(parentNode[PropertySymbol.ownerDocument].createTextNode(String(node)), firstChild);
            }
        }
    }
    /**
     * Replaces the existing children of a ParentNode with a specified new set of children.
     *
     * @param parentNode Parent node.
     * @param nodes List of Node or DOMString.
     */
    static replaceChildren(parentNode, ...nodes) {
        const childNodes = parentNode[PropertySymbol.nodeArray];
        while (childNodes.length) {
            parentNode.removeChild(childNodes[0]);
        }
        this.append(parentNode, ...nodes);
    }
    /**
     * Returns an elements by class name.
     *
     * @param parentNode Parent node.
     * @param className Tag name.
     * @returns Matching element.
     */
    static getElementsByClassName(parentNode, className) {
        return new HTMLCollection(PropertySymbol.illegalConstructor, () => QuerySelector.querySelectorAll(parentNode, `.${className.replace(/\s+/gm, '.')}`)[PropertySymbol.items]);
    }
    /**
     * Returns an elements by tag name.
     *
     * @param parentNode Parent node.
     * @param tagName Tag name.
     * @returns Matching element.
     */
    static getElementsByTagName(parentNode, tagName) {
        const upperTagName = tagName.toUpperCase();
        const includeAll = tagName === '*';
        const find = (parent, cachedResult) => {
            const elements = [];
            for (const element of parent[PropertySymbol.elementArray]) {
                if (includeAll || element[PropertySymbol.tagName].toUpperCase() === upperTagName) {
                    elements.push(element);
                }
                element[PropertySymbol.affectsCache].push(cachedResult);
                for (const foundElement of find(element, cachedResult)) {
                    elements.push(foundElement);
                }
            }
            return elements;
        };
        const query = () => {
            const cache = parentNode[PropertySymbol.cache].elementsByTagName;
            const cachedItems = cache.get(tagName);
            if (cachedItems?.result) {
                const items = cachedItems.result.deref();
                if (items) {
                    return items;
                }
            }
            const cachedResult = { result: null };
            const items = find(parentNode, cachedResult);
            cachedResult.result = new WeakRef(items);
            cache.set(tagName, cachedResult);
            return items;
        };
        return new HTMLCollection(PropertySymbol.illegalConstructor, query);
    }
    /**
     * Returns an elements by tag name and namespace.
     *
     * @param parentNode Parent node.
     * @param namespaceURI Namespace URI.
     * @param tagName Tag name.
     * @returns Matching element.
     */
    static getElementsByTagNameNS(parentNode, namespaceURI, tagName) {
        // When the namespace is HTML, the tag name is case-insensitive.
        const formattedTagName = namespaceURI === NamespaceURI.html &&
            parentNode[PropertySymbol.ownerDocument][PropertySymbol.contentType] === 'text/html'
            ? tagName.toUpperCase()
            : tagName;
        const includeAll = tagName === '*';
        const find = (parent, cachedResult) => {
            const elements = [];
            for (const element of parent[PropertySymbol.elementArray]) {
                if ((includeAll || element[PropertySymbol.tagName] === formattedTagName) &&
                    element[PropertySymbol.namespaceURI] === namespaceURI) {
                    elements.push(element);
                }
                element[PropertySymbol.affectsCache].push(cachedResult);
                for (const foundElement of find(element, cachedResult)) {
                    elements.push(foundElement);
                }
            }
            return elements;
        };
        const query = () => {
            const cache = parentNode[PropertySymbol.cache].elementsByTagNameNS;
            const cachedItems = cache.get(tagName);
            if (cachedItems?.result) {
                const items = cachedItems.result.deref();
                if (items) {
                    return items;
                }
            }
            const cachedResult = { result: null };
            const items = find(parentNode, cachedResult);
            cachedResult.result = new WeakRef(items);
            cache.set(tagName, cachedResult);
            return items;
        };
        return new HTMLCollection(PropertySymbol.illegalConstructor, query);
    }
    /**
     * Returns the first element matching a tag name.
     * This is not part of the browser standard and is only used internally (used in Document).
     *
     * @param parentNode Parent node.
     * @param tagName Tag name.
     * @returns Matching element.
     */
    static getElementByTagName(parentNode, tagName) {
        const upperTagName = tagName.toUpperCase();
        const find = (parent, cachedResult) => {
            for (const element of parent[PropertySymbol.elementArray]) {
                element[PropertySymbol.affectsCache].push(cachedResult);
                if (element[PropertySymbol.tagName] === upperTagName) {
                    return element;
                }
                const foundElement = find(element, cachedResult);
                if (foundElement) {
                    return foundElement;
                }
            }
            return null;
        };
        const cache = parentNode[PropertySymbol.cache].elementByTagName;
        const cachedItem = cache.get(tagName);
        if (cachedItem?.result) {
            const item = cachedItem.result.deref();
            if (item) {
                return item;
            }
        }
        const cachedResult = { result: null };
        const item = find(parentNode, cachedResult);
        cachedResult.result = item ? new WeakRef(item) : { deref: () => null };
        cache.set(tagName, cachedResult);
        return item;
    }
    /**
     * Returns an element by ID.
     *
     * @param parentNode Parent node.
     * @param id ID.
     * @returns Matching element.
     */
    static getElementById(parentNode, id) {
        id = String(id);
        if (parentNode instanceof Document) {
            const entry = parentNode[PropertySymbol.elementIdMap].get(id);
            if (entry?.elements.length > 0) {
                return entry.elements[0];
            }
            return null;
        }
        const find = (parent, cachedResult) => {
            for (const element of parent[PropertySymbol.elementArray]) {
                element[PropertySymbol.affectsCache].push(cachedResult);
                if (element.getAttribute('id') === id) {
                    return element;
                }
                const foundElement = find(element, cachedResult);
                if (foundElement) {
                    return foundElement;
                }
            }
            return null;
        };
        const cache = parentNode[PropertySymbol.cache].elementById;
        const cachedItem = cache.get(id);
        if (cachedItem?.result) {
            const item = cachedItem.result.deref();
            if (item) {
                return item;
            }
        }
        const cachedResult = { result: null };
        const item = find(parentNode, cachedResult);
        cachedResult.result = item ? new WeakRef(item) : { deref: () => null };
        cache.set(id, cachedResult);
        return item;
    }
}
//# sourceMappingURL=ParentNodeUtility.js.map