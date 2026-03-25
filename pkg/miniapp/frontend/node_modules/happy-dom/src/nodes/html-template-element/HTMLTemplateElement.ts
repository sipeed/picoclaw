import HTMLElement from '../html-element/HTMLElement.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import DocumentFragment from '../document-fragment/DocumentFragment.js';
import Node from '../node/Node.js';
import ShadowRoot from '../shadow-root/ShadowRoot.js';
import HTMLSerializer from '../../html-serializer/HTMLSerializer.js';
import HTMLParser from '../../html-parser/HTMLParser.js';

/**
 * HTML Template Element.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/HTMLTemplateElement.
 */
export default class HTMLTemplateElement extends HTMLElement {
	// Public properties
	public declare cloneNode: (deep?: boolean) => HTMLTemplateElement;

	// Internal properties
	public [PropertySymbol.content]: DocumentFragment =
		this[PropertySymbol.ownerDocument].createDocumentFragment();

	/**
	 * Returns content.
	 *
	 * @returns Content.
	 */
	public get content(): DocumentFragment {
		return this[PropertySymbol.content];
	}

	/**
	 * @override
	 */
	public override get innerHTML(): string {
		return this.getHTML();
	}

	/**
	 * @override
	 */
	public override set innerHTML(html: string) {
		const content = <DocumentFragment>this[PropertySymbol.content];
		const childNodes = content[PropertySymbol.nodeArray];

		while (childNodes.length) {
			content.removeChild(childNodes[0]);
		}

		new HTMLParser(this[PropertySymbol.window]).parse(html, this[PropertySymbol.content]);
	}

	/**
	 * @override
	 */
	public override get firstChild(): Node {
		return this[PropertySymbol.content].firstChild;
	}

	/**
	 * @override
	 */
	public override get lastChild(): Node {
		return this[PropertySymbol.content].lastChild;
	}

	/**
	 * @deprecated
	 * @override
	 */
	public override getInnerHTML(_options?: { includeShadowRoots?: boolean }): string {
		const serializer = new HTMLSerializer();

		// Options should be ignored as shadow roots should not be serialized for HTMLTemplateElement.

		const content = <DocumentFragment>this[PropertySymbol.content];
		let html = '';

		for (const node of content[PropertySymbol.nodeArray]) {
			html += serializer.serializeToString(node);
		}

		return html;
	}

	/**
	 * @override
	 */
	public override getHTML(_options?: {
		serializableShadowRoots?: boolean;
		shadowRoots?: ShadowRoot[];
	}): string {
		const serializer = new HTMLSerializer();

		// Options should be ignored as shadow roots should not be serialized for HTMLTemplateElement.

		const content = <DocumentFragment>this[PropertySymbol.content];
		let html = '';

		for (const node of content[PropertySymbol.nodeArray]) {
			html += serializer.serializeToString(node);
		}

		return html;
	}

	/**
	 * @override
	 */
	public override [PropertySymbol.appendChild](node: Node, disableValidations = false): Node {
		return this[PropertySymbol.content][PropertySymbol.appendChild](node, disableValidations);
	}

	/**
	 * @override
	 */
	public override [PropertySymbol.removeChild](node: Node): Node {
		return this[PropertySymbol.content][PropertySymbol.removeChild](node);
	}

	/**
	 * @override
	 */
	public override [PropertySymbol.insertBefore](
		newNode: Node,
		referenceNode: Node,
		disableValidations = false
	): Node {
		return this[PropertySymbol.content][PropertySymbol.insertBefore](
			newNode,
			referenceNode,
			disableValidations
		);
	}

	/**
	 * @override
	 */
	public override [PropertySymbol.replaceChild](newChild: Node, oldChild: Node): Node {
		return this[PropertySymbol.content][PropertySymbol.replaceChild](newChild, oldChild);
	}

	/**
	 * @override
	 */
	public override [PropertySymbol.cloneNode](deep = false): HTMLTemplateElement {
		const clone = <HTMLTemplateElement>super[PropertySymbol.cloneNode](deep);
		clone[PropertySymbol.content] = <DocumentFragment>this[PropertySymbol.content].cloneNode(deep);
		return clone;
	}
}
