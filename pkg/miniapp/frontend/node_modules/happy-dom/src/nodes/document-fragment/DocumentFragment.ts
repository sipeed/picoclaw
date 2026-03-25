import Node from '../node/Node.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import Element from '../element/Element.js';
import QuerySelector from '../../query-selector/QuerySelector.js';
import ParentNodeUtility from '../parent-node/ParentNodeUtility.js';
import HTMLCollection from '../element/HTMLCollection.js';
import NodeTypeEnum from '../node/NodeTypeEnum.js';
import IHTMLElementTagNameMap from '../../config/IHTMLElementTagNameMap.js';
import ISVGElementTagNameMap from '../../config/ISVGElementTagNameMap.js';
import NodeList from '../node/NodeList.js';

/**
 * DocumentFragment.
 */
export default class DocumentFragment extends Node {
	public [PropertySymbol.children]: HTMLCollection<Element> | null = null;
	public [PropertySymbol.rootNode]: Node = this;
	public [PropertySymbol.nodeType] = NodeTypeEnum.documentFragmentNode;
	public declare cloneNode: (deep?: boolean) => DocumentFragment;

	/**
	 * Returns the document fragment children.
	 */
	public get children(): HTMLCollection<Element> {
		if (!this[PropertySymbol.children]) {
			const elements = this[PropertySymbol.elementArray];
			this[PropertySymbol.children] = new HTMLCollection<Element>(
				PropertySymbol.illegalConstructor,
				() => elements
			);
		}
		return this[PropertySymbol.children];
	}

	/**
	 * Last element child.
	 *
	 * @returns Element.
	 */
	public get childElementCount(): number {
		return this[PropertySymbol.elementArray].length;
	}

	/**
	 * First element child.
	 *
	 * @returns Element.
	 */
	public get firstElementChild(): Element {
		return this[PropertySymbol.elementArray][0] ?? null;
	}

	/**
	 * Last element child.
	 *
	 * @returns Element.
	 */
	public get lastElementChild(): Element {
		const children = this[PropertySymbol.elementArray];
		return children[children.length - 1] ?? null;
	}

	/**
	 * Get text value of children.
	 *
	 * @returns Text content.
	 */
	public get textContent(): string {
		let result = '';
		for (const childNode of this[PropertySymbol.nodeArray]) {
			if (
				childNode[PropertySymbol.nodeType] === NodeTypeEnum.elementNode ||
				childNode[PropertySymbol.nodeType] === NodeTypeEnum.textNode
			) {
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
	public set textContent(textContent: string) {
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
	public append(...nodes: (Node | string)[]): void {
		ParentNodeUtility.append(this, ...nodes);
	}

	/**
	 * Inserts a set of Node objects or DOMString objects before the first child of the ParentNode. DOMString objects are inserted as equivalent Text nodes.
	 *
	 * @param nodes List of Node or DOMString.
	 */
	public prepend(...nodes: (Node | string)[]): void {
		ParentNodeUtility.prepend(this, ...nodes);
	}

	/**
	 * Replaces the existing children of a node with a specified new set of children.
	 *
	 * @param nodes List of Node or DOMString.
	 */
	public replaceChildren(...nodes: (Node | string)[]): void {
		ParentNodeUtility.replaceChildren(this, ...nodes);
	}

	/**
	 * Query CSS selector to find matching nodes.
	 *
	 * @param selector CSS selector.
	 * @returns Matching elements.
	 */
	public querySelectorAll<K extends keyof IHTMLElementTagNameMap>(
		selector: K
	): NodeList<IHTMLElementTagNameMap[K]>;

	/**
	 * Query CSS selector to find matching elments.
	 *
	 * @param selector CSS selector.
	 * @returns Matching elements.
	 */
	public querySelectorAll<K extends keyof ISVGElementTagNameMap>(
		selector: K
	): NodeList<ISVGElementTagNameMap[K]>;

	/**
	 * Query CSS selector to find matching elments.
	 *
	 * @param selector CSS selector.
	 * @returns Matching elements.
	 */
	public querySelectorAll(selector: string): NodeList<Element>;

	/**
	 * Query CSS selector to find matching elments.
	 *
	 * @param selector CSS selector.
	 * @returns Matching elements.
	 */
	public querySelectorAll(selector: string): NodeList<Element> {
		return QuerySelector.querySelectorAll(this, selector);
	}

	/**
	 * Query CSS Selector to find matching node.
	 *
	 * @param selector CSS selector.
	 * @returns Matching element.
	 */
	public querySelector<K extends keyof IHTMLElementTagNameMap>(
		selector: K
	): IHTMLElementTagNameMap[K] | null;

	/**
	 * Query CSS Selector to find a matching element.
	 *
	 * @param selector CSS selector.
	 * @returns Matching element.
	 */
	public querySelector<K extends keyof ISVGElementTagNameMap>(
		selector: K
	): ISVGElementTagNameMap[K] | null;

	/**
	 * Query CSS Selector to find a matching element.
	 *
	 * @param selector CSS selector.
	 * @returns Matching element.
	 */
	public querySelector(selector: string): Element | null;

	/**
	 * Query CSS Selector to find a matching element.
	 *
	 * @param selector CSS selector.
	 * @returns Matching element.
	 */
	public querySelector(selector: string): Element | null {
		return QuerySelector.querySelector(this, selector);
	}

	/**
	 * Returns an element by ID.
	 *
	 * @param id ID.
	 * @returns Matching element.
	 */
	public getElementById(id: string): Element | null {
		return ParentNodeUtility.getElementById(this, id);
	}
}
