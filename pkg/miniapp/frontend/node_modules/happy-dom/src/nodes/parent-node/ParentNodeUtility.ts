import * as PropertySymbol from '../../PropertySymbol.js';
import DocumentFragment from '../document-fragment/DocumentFragment.js';
import Document from '../document/Document.js';
import Element from '../element/Element.js';
import Node from '../node/Node.js';
import NamespaceURI from '../../config/NamespaceURI.js';
import HTMLCollection from '../element/HTMLCollection.js';
import QuerySelector from '../../query-selector/QuerySelector.js';
import ICachedResult from '../node/ICachedResult.js';

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
	public static append(parentNode: Element | Document | DocumentFragment, ...nodes: any[]): void {
		for (const node of nodes) {
			if (node instanceof Node) {
				parentNode.appendChild(node);
			} else {
				parentNode.appendChild(
					parentNode[PropertySymbol.ownerDocument].createTextNode(String(node))
				);
			}
		}
	}

	/**
	 * Inserts a set of Node objects or DOMString objects before the first child of the ParentNode. DOMString objects are inserted as equivalent Text nodes.
	 *
	 * @param parentNode Parent node.
	 * @param nodes List of Node or DOMString.
	 */
	public static prepend(
		parentNode: Element | Document | DocumentFragment,
		...nodes: (string | Node)[]
	): void {
		const firstChild = parentNode.firstChild;
		for (const node of nodes) {
			if (node instanceof Node) {
				parentNode.insertBefore(node, firstChild);
			} else {
				parentNode.insertBefore(
					parentNode[PropertySymbol.ownerDocument].createTextNode(String(node)),
					firstChild
				);
			}
		}
	}

	/**
	 * Replaces the existing children of a ParentNode with a specified new set of children.
	 *
	 * @param parentNode Parent node.
	 * @param nodes List of Node or DOMString.
	 */
	public static replaceChildren(
		parentNode: Element | Document | DocumentFragment,
		...nodes: (string | Node)[]
	): void {
		const childNodes = (<DocumentFragment>parentNode)[PropertySymbol.nodeArray];

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
	public static getElementsByClassName(
		parentNode: Element | DocumentFragment | Document,
		className: string
	): HTMLCollection<Element> {
		return new HTMLCollection(
			PropertySymbol.illegalConstructor,
			() =>
				QuerySelector.querySelectorAll(parentNode, `.${className.replace(/\s+/gm, '.')}`)[
					PropertySymbol.items
				]
		);
	}

	/**
	 * Returns an elements by tag name.
	 *
	 * @param parentNode Parent node.
	 * @param tagName Tag name.
	 * @returns Matching element.
	 */
	public static getElementsByTagName<T extends Element = Element>(
		parentNode: Element | DocumentFragment | Document,
		tagName: string
	): HTMLCollection<T> {
		const upperTagName = tagName.toUpperCase();
		const includeAll = tagName === '*';

		const find = (
			parent: Element | DocumentFragment | Document,
			cachedResult: ICachedResult
		): T[] => {
			const elements: T[] = [];

			for (const element of (<DocumentFragment>parent)[PropertySymbol.elementArray]) {
				if (includeAll || element[PropertySymbol.tagName].toUpperCase() === upperTagName) {
					elements.push(<T>element);
				}

				element[PropertySymbol.affectsCache].push(cachedResult);

				for (const foundElement of find(<Element>element, cachedResult)) {
					elements.push(foundElement);
				}
			}

			return elements;
		};

		const query = (): T[] => {
			const cache = parentNode[PropertySymbol.cache].elementsByTagName;
			const cachedItems = cache.get(tagName);

			if (cachedItems?.result) {
				const items = cachedItems.result.deref();
				if (items) {
					return <T[]>items;
				}
			}

			const cachedResult = { result: null };
			const items = find(parentNode, cachedResult);

			cachedResult.result = new WeakRef(items);
			cache.set(tagName, cachedResult);

			return items;
		};

		return new HTMLCollection<T>(PropertySymbol.illegalConstructor, query);
	}

	/**
	 * Returns an elements by tag name and namespace.
	 *
	 * @param parentNode Parent node.
	 * @param namespaceURI Namespace URI.
	 * @param tagName Tag name.
	 * @returns Matching element.
	 */
	public static getElementsByTagNameNS(
		parentNode: Element | DocumentFragment | Document,
		namespaceURI: string,
		tagName: string
	): HTMLCollection<Element> {
		// When the namespace is HTML, the tag name is case-insensitive.
		const formattedTagName =
			namespaceURI === NamespaceURI.html &&
			parentNode[PropertySymbol.ownerDocument][PropertySymbol.contentType] === 'text/html'
				? tagName.toUpperCase()
				: tagName;
		const includeAll = tagName === '*';

		const find = (
			parent: Element | DocumentFragment | Document,
			cachedResult: ICachedResult
		): Element[] => {
			const elements: Element[] = [];

			for (const element of (<DocumentFragment>parent)[PropertySymbol.elementArray]) {
				if (
					(includeAll || element[PropertySymbol.tagName] === formattedTagName) &&
					element[PropertySymbol.namespaceURI] === namespaceURI
				) {
					elements.push(<Element>element);
				}

				element[PropertySymbol.affectsCache].push(cachedResult);

				for (const foundElement of find(<Element>element, cachedResult)) {
					elements.push(foundElement);
				}
			}

			return elements;
		};

		const query = (): Element[] => {
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
	public static getElementByTagName(
		parentNode: Element | DocumentFragment | Document,
		tagName: string
	): Element {
		const upperTagName = tagName.toUpperCase();

		const find = (
			parent: Element | DocumentFragment | Document,
			cachedResult: ICachedResult
		): Element | null => {
			for (const element of (<DocumentFragment>parent)[PropertySymbol.elementArray]) {
				element[PropertySymbol.affectsCache].push(cachedResult);

				if (element[PropertySymbol.tagName] === upperTagName) {
					return <Element>element;
				}

				const foundElement = find(<Element>element, cachedResult);
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
	public static getElementById(
		parentNode: Element | DocumentFragment | Document,
		id: string
	): Element | null {
		id = String(id);

		if (parentNode instanceof Document) {
			const entry = parentNode[PropertySymbol.elementIdMap].get(id);
			if (entry?.elements.length > 0) {
				return entry.elements[0];
			}
			return null;
		}

		const find = (
			parent: Element | DocumentFragment | Document,
			cachedResult: ICachedResult
		): Element | null => {
			for (const element of (<DocumentFragment>parent)[PropertySymbol.elementArray]) {
				element[PropertySymbol.affectsCache].push(cachedResult);

				if (element.getAttribute('id') === id) {
					return <Element>element;
				}

				const foundElement = find(<Element>element, cachedResult);

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
