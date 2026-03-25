import Node from '../node/Node.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import ShadowRoot from '../shadow-root/ShadowRoot.js';
import DOMRect from '../../dom/DOMRect.js';
import DOMTokenList from '../../dom/DOMTokenList.js';
import QuerySelector from '../../query-selector/QuerySelector.js';
import ChildNodeUtility from '../child-node/ChildNodeUtility.js';
import ParentNodeUtility from '../parent-node/ParentNodeUtility.js';
import NonDocumentChildNodeUtility from '../child-node/NonDocumentChildNodeUtility.js';
import HTMLCollection from './HTMLCollection.js';
import Text from '../text/Text.js';
import DOMRectList from '../../dom/DOMRectList.js';
import Attr from '../attr/Attr.js';
import NamedNodeMap from './NamedNodeMap.js';
import Event from '../../event/Event.js';
import EventPhaseEnum from '../../event/EventPhaseEnum.js';
import WindowErrorUtility from '../../window/WindowErrorUtility.js';
import WindowBrowserContext from '../../window/WindowBrowserContext.js';
import BrowserErrorCaptureEnum from '../../browser/enums/BrowserErrorCaptureEnum.js';
import NodeTypeEnum from '../node/NodeTypeEnum.js';
import IHTMLElementTagNameMap from '../../config/IHTMLElementTagNameMap.js';
import ISVGElementTagNameMap from '../../config/ISVGElementTagNameMap.js';
import IChildNode from '../child-node/IChildNode.js';
import INonDocumentTypeChildNode from '../child-node/INonDocumentTypeChildNode.js';
import IParentNode from '../parent-node/IParentNode.js';
import MutationRecord from '../../mutation-observer/MutationRecord.js';
import MutationTypeEnum from '../../mutation-observer/MutationTypeEnum.js';
import NamespaceURI from '../../config/NamespaceURI.js';
import NodeList from '../node/NodeList.js';
import CSSStyleDeclaration from '../../css/declaration/CSSStyleDeclaration.js';
import NamedNodeMapProxyFactory from './NamedNodeMapProxyFactory.js';
import NodeFactory from '../NodeFactory.js';
import HTMLSerializer from '../../html-serializer/HTMLSerializer.js';
import HTMLParser from '../../html-parser/HTMLParser.js';
import IScrollToOptions from '../../window/IScrollToOptions.js';

type InsertAdjacentPosition = 'beforebegin' | 'afterbegin' | 'beforeend' | 'afterend';

/**
 * Element.
 */
export default class Element
	extends Node
	implements IChildNode, INonDocumentTypeChildNode, IParentNode
{
	public static [PropertySymbol.tagName]: string | null = null;
	public static [PropertySymbol.localName]: string | null = null;
	public static [PropertySymbol.namespaceURI]: string | null = null;
	public declare cloneNode: (deep?: boolean) => Element;

	// Events
	public oncancel: ((event: Event) => void) | null = null;
	public onerror: ((event: Event) => void) | null = null;
	public onscroll: ((event: Event) => void) | null = null;
	public onselect: ((event: Event) => void) | null = null;
	public onwheel: ((event: Event) => void) | null = null;
	public oncopy: ((event: Event) => void) | null = null;
	public oncut: ((event: Event) => void) | null = null;
	public onpaste: ((event: Event) => void) | null = null;
	public oncompositionend: ((event: Event) => void) | null = null;
	public oncompositionstart: ((event: Event) => void) | null = null;
	public oncompositionupdate: ((event: Event) => void) | null = null;
	public onblur: ((event: Event) => void) | null = null;
	public onfocus: ((event: Event) => void) | null = null;
	public onfocusin: ((event: Event) => void) | null = null;
	public onfocusout: ((event: Event) => void) | null = null;
	public onfullscreenchange: ((event: Event) => void) | null = null;
	public onfullscreenerror: ((event: Event) => void) | null = null;
	public onkeydown: ((event: Event) => void) | null = null;
	public onkeyup: ((event: Event) => void) | null = null;
	public onauxclick: ((event: Event) => void) | null = null;
	public onclick: ((event: Event) => void) | null = null;
	public oncontextmenu: ((event: Event) => void) | null = null;
	public ondblclick: ((event: Event) => void) | null = null;
	public onmousedown: ((event: Event) => void) | null = null;
	public onmouseenter: ((event: Event) => void) | null = null;
	public onmouseleave: ((event: Event) => void) | null = null;
	public onmousemove: ((event: Event) => void) | null = null;
	public onmouseout: ((event: Event) => void) | null = null;
	public onmouseover: ((event: Event) => void) | null = null;
	public onmouseup: ((event: Event) => void) | null = null;
	public ontouchcancel: ((event: Event) => void) | null = null;
	public ontouchend: ((event: Event) => void) | null = null;
	public ontouchmove: ((event: Event) => void) | null = null;
	public ontouchstart: ((event: Event) => void) | null = null;

	// Internal properties
	public [PropertySymbol.classList]: DOMTokenList | null = null;
	public [PropertySymbol.isValue]: string | null = null;
	public [PropertySymbol.nodeType] = NodeTypeEnum.elementNode;
	public [PropertySymbol.prefix]: string | null = null;
	public [PropertySymbol.shadowRoot]: ShadowRoot | null = null;
	public [PropertySymbol.scrollHeight] = 0;
	public [PropertySymbol.scrollWidth] = 0;
	public [PropertySymbol.scrollTop] = 0;
	public [PropertySymbol.scrollLeft] = 0;
	public [PropertySymbol.attributes] = new NamedNodeMap(this);
	public [PropertySymbol.attributesProxy]: NamedNodeMap | null = null;
	public [PropertySymbol.children]: HTMLCollection<Element> | null = null;
	public [PropertySymbol.computedStyle]: CSSStyleDeclaration | null = null;
	public declare [PropertySymbol.tagName]: string | null;
	public declare [PropertySymbol.localName]: string | null;
	public declare [PropertySymbol.namespaceURI]: string | null;

	/**
	 * Constructor.
	 */
	constructor() {
		super();

		// CustomElementRegistry will populate the properties upon calling "CustomElementRegistry.define()".
		// Elements that can be constructed with the "new" keyword (without using "Document.createElement()") will also populate the properties.

		if (!this[PropertySymbol.tagName]) {
			this[PropertySymbol.tagName] = null;
		}

		if (!this[PropertySymbol.localName]) {
			this[PropertySymbol.localName] = null;
		}

		if (!this[PropertySymbol.namespaceURI]) {
			this[PropertySymbol.namespaceURI] = null;
		}
	}

	/**
	 * Returns tag name.
	 *
	 * @returns Tag name.
	 */
	public get tagName(): string {
		return <string>this[PropertySymbol.tagName];
	}

	/**
	 * Returns prefix.
	 *
	 * @returns Prefix.
	 */
	public get prefix(): string | null {
		return <string>this[PropertySymbol.prefix];
	}

	/**
	 * Returns shadow root.
	 *
	 * @returns Shadow root.
	 */
	public get shadowRoot(): ShadowRoot | null {
		const shadowRoot = this[PropertySymbol.shadowRoot];
		return shadowRoot && shadowRoot[PropertySymbol.mode] === 'open' ? shadowRoot : null;
	}

	/**
	 * Returns scroll height.
	 *
	 * @returns Scroll height.
	 */
	public get scrollHeight(): number {
		return this[PropertySymbol.scrollHeight];
	}

	/**
	 * Returns scroll width.
	 *
	 * @returns Scroll width.
	 */
	public get scrollWidth(): number {
		return this[PropertySymbol.scrollWidth];
	}

	/**
	 * Returns scroll top.
	 *
	 * @returns Scroll top.
	 */
	public get scrollTop(): number {
		return this[PropertySymbol.scrollTop];
	}

	/**
	 * Sets scroll top.
	 *
	 * @param value Scroll top.
	 */
	public set scrollTop(value: number) {
		this[PropertySymbol.scrollTop] = value;
	}

	/**
	 * Returns scroll left.
	 *
	 * @returns Scroll left.
	 */
	public get scrollLeft(): number {
		return this[PropertySymbol.scrollLeft];
	}

	/**
	 * Sets scroll left.
	 *
	 * @param value Scroll left.
	 */
	public set scrollLeft(value: number) {
		this[PropertySymbol.scrollLeft] = value;
	}

	/**
	 * Returns attributes.
	 *
	 * @returns Attributes.
	 */
	public get attributes(): NamedNodeMap {
		if (!this[PropertySymbol.attributesProxy]) {
			this[PropertySymbol.attributesProxy] = NamedNodeMapProxyFactory.createProxy(
				this[PropertySymbol.attributes]
			);
		}
		return this[PropertySymbol.attributesProxy];
	}

	/**
	 * Returns namespace URI.
	 *
	 * @returns Namespace URI.
	 */
	public get namespaceURI(): string | null {
		return this[PropertySymbol.namespaceURI];
	}

	/**
	 * Returns element children.
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
	 * Returns class list.
	 *
	 * @returns Class list.
	 */
	public get classList(): DOMTokenList {
		if (!this[PropertySymbol.classList]) {
			this[PropertySymbol.classList] = new DOMTokenList(
				PropertySymbol.illegalConstructor,
				this,
				'class'
			);
		}
		return <DOMTokenList>this[PropertySymbol.classList];
	}

	/**
	 * Returns ID.
	 *
	 * @returns ID.
	 */
	public get id(): string {
		return this.getAttribute('id') || '';
	}

	/**
	 * Sets ID.
	 *
	 * @param id ID.
	 */
	public set id(id: string) {
		this.setAttribute('id', id);
	}

	/**
	 * Returns class name.
	 *
	 * @returns Class name.
	 */
	public get className(): string {
		return this.getAttribute('class') || '';
	}

	/**
	 * Sets class name.
	 *
	 * @param className Class name.
	 */
	public set className(className: string) {
		this.setAttribute('class', className);
	}

	/**
	 * Node name.
	 *
	 * @returns Node name.
	 */
	public get nodeName(): string {
		return this[PropertySymbol.tagName];
	}

	/**
	 * Local name.
	 *
	 * @returns Local name.
	 */
	public get localName(): string {
		return this[PropertySymbol.localName];
	}

	/**
	 * Returns role.
	 *
	 * @returns Role.
	 */
	public get role(): string {
		return this.getAttribute('role') || '';
	}

	/**
	 * Sets role.
	 *
	 * @param role Role.
	 */
	public set role(role: string) {
		this.setAttribute('role', role);
	}

	/**
	 * Previous element sibling.
	 *
	 * @returns Element.
	 */
	public get previousElementSibling(): Element {
		return NonDocumentChildNodeUtility.previousElementSibling(this);
	}

	/**
	 * Next element sibling.
	 *
	 * @returns Element.
	 */
	public get nextElementSibling(): Element {
		return NonDocumentChildNodeUtility.nextElementSibling(this);
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
	 * Returns inner HTML.
	 *
	 * @returns HTML.
	 */
	public get innerHTML(): string {
		return this.getHTML();
	}

	/**
	 * Sets inner HTML.
	 *
	 * @param html HTML.
	 */
	public set innerHTML(html: string) {
		const childNodes = this[PropertySymbol.nodeArray];

		while (childNodes.length) {
			this.removeChild(childNodes[0]);
		}

		new HTMLParser(this[PropertySymbol.window]).parse(html, this);
	}

	/**
	 * Returns outer HTML.
	 *
	 * @returns HTML.
	 */
	public get outerHTML(): string {
		return new HTMLSerializer().serializeToString(this);
	}

	/**
	 * Returns outer HTML.
	 *
	 * @param html HTML.
	 */
	public set outerHTML(html: string) {
		const childNodes = new HTMLParser(this[PropertySymbol.window]).parse(html)[
			PropertySymbol.nodeArray
		];
		this.replaceWith(...childNodes);
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
	 * Returns slot.
	 *
	 * @returns Slot.
	 */
	public get slot(): string {
		return this.getAttributeNS(null, 'slot') || '';
	}

	/**
	 * Returns slot.
	 *
	 * @param slot Slot.
	 */
	public set slot(title: string) {
		this.setAttribute('slot', title);
	}

	/**
	 * Returns inner HTML and optionally the content of shadow roots.
	 *
	 * @deprecated
	 * @param [options] Options.
	 * @param [options.includeShadowRoots] Set to "true" to include shadow roots.
	 * @returns HTML.
	 */
	public getInnerHTML(options?: { includeShadowRoots?: boolean }): string {
		const serializer = new HTMLSerializer({
			allShadowRoots: !!options?.includeShadowRoots
		});

		let html = '';

		for (const node of this[PropertySymbol.nodeArray]) {
			html += serializer.serializeToString(node);
		}

		return html;
	}

	/**
	 * Returns inner HTML and optionally the content of shadow roots.
	 *
	 * @param [options] Options.
	 * @param [options.serializableShadowRoots] A boolean value that specifies whether to include serializable shadow roots. The default value is false.
	 * @param [options.shadowRoots] An array of ShadowRoot objects to serialize. These are included regardless of whether they are marked as serializable, or if they are open or closed. The default value is an empty array.
	 * @returns HTML.
	 */
	public getHTML(options?: {
		serializableShadowRoots?: boolean;
		shadowRoots?: ShadowRoot[];
	}): string {
		const serializer = new HTMLSerializer({
			serializableShadowRoots: !!options?.serializableShadowRoots,
			shadowRoots: options?.shadowRoots ?? null
		});

		let html = '';

		for (const node of this[PropertySymbol.nodeArray]) {
			html += serializer.serializeToString(node);
		}

		return html;
	}

	/**
	 * @override
	 */
	public override [PropertySymbol.cloneNode](deep = false): Element {
		const clone = <Element>super[PropertySymbol.cloneNode](deep);

		clone[PropertySymbol.tagName] = this[PropertySymbol.tagName];
		clone[PropertySymbol.localName] = this[PropertySymbol.localName];
		clone[PropertySymbol.namespaceURI] = this[PropertySymbol.namespaceURI];

		if (this[PropertySymbol.shadowRoot]?.[PropertySymbol.clonable]) {
			clone[PropertySymbol.shadowRoot] = this[PropertySymbol.shadowRoot].cloneNode(deep);
			clone[PropertySymbol.shadowRoot][PropertySymbol.host] = clone;
		}

		for (const item of this[PropertySymbol.attributes][PropertySymbol.namespaceItems].values()) {
			clone[PropertySymbol.attributes].setNamedItem(item.cloneNode());
		}

		return <Element>clone;
	}

	/**
	 * Removes the node from its parent.
	 */
	public remove(): void {
		ChildNodeUtility.remove(this);
	}

	/**
	 * The Node.replaceWith() method replaces this Node in the children list of its parent with a set of Node or DOMString objects.
	 *
	 * @param nodes List of Node or DOMString.
	 */
	public replaceWith(...nodes: (Node | string)[]): void {
		ChildNodeUtility.replaceWith(this, ...nodes);
	}

	/**
	 * Inserts a set of Node or DOMString objects in the children list of this ChildNode's parent, just before this ChildNode. DOMString objects are inserted as equivalent Text nodes.
	 *
	 * @param nodes List of Node or DOMString.
	 */
	public before(...nodes: (string | Node)[]): void {
		ChildNodeUtility.before(this, ...nodes);
	}

	/**
	 * Inserts a set of Node or DOMString objects in the children list of this ChildNode's parent, just after this ChildNode. DOMString objects are inserted as equivalent Text nodes.
	 *
	 * @param nodes List of Node or DOMString.
	 */
	public after(...nodes: (string | Node)[]): void {
		ChildNodeUtility.after(this, ...nodes);
	}

	/**
	 * Inserts a set of Node objects or DOMString objects after the last child of the ParentNode. DOMString objects are inserted as equivalent Text nodes.
	 *
	 * @param nodes List of Node or DOMString.
	 */
	public append(...nodes: (string | Node)[]): void {
		ParentNodeUtility.append(this, ...nodes);
	}

	/**
	 * Inserts a set of Node objects or DOMString objects before the first child of the ParentNode. DOMString objects are inserted as equivalent Text nodes.
	 *
	 * @param nodes List of Node or DOMString.
	 */
	public prepend(...nodes: (string | Node)[]): void {
		ParentNodeUtility.prepend(this, ...nodes);
	}

	/**
	 * Replaces the existing children of a node with a specified new set of children.
	 *
	 * @param nodes List of Node or DOMString.
	 */
	public replaceChildren(...nodes: (string | Node)[]): void {
		ParentNodeUtility.replaceChildren(this, ...nodes);
	}

	/**
	 * Inserts a node to the given position.
	 *
	 * @param position Position to insert element.
	 * @param element Node to insert.
	 * @returns Inserted node or null if couldn't insert.
	 */
	public insertAdjacentElement(position: InsertAdjacentPosition, element: Node): Node | null {
		if (position === 'beforebegin') {
			if (!this.parentElement) {
				return null;
			}

			this.parentElement.insertBefore(element, this);
		} else if (position === 'afterbegin') {
			this.insertBefore(element, this.firstChild);
		} else if (position === 'beforeend') {
			this.appendChild(element);
		} else if (position === 'afterend') {
			if (!this.parentElement) {
				return null;
			}

			this.parentElement.insertBefore(element, this.nextSibling);
		}

		return element;
	}

	/**
	 * Inserts an HTML string to the given position.
	 *
	 * @param position Position to insert text.
	 * @param text HTML string to insert.
	 */
	public insertAdjacentHTML(position: InsertAdjacentPosition, text: string): void {
		const childNodes = new HTMLParser(this[PropertySymbol.window]).parse(text)[
			PropertySymbol.nodeArray
		];
		while (childNodes.length) {
			this.insertAdjacentElement(position, childNodes[0]);
		}
	}

	/**
	 * Inserts text to the given position.
	 *
	 * @param position Position to insert text.
	 * @param text String to insert.
	 */
	public insertAdjacentText(position: InsertAdjacentPosition, text: string): void {
		if (!text) {
			return;
		}
		const textNode = <Text>this[PropertySymbol.ownerDocument].createTextNode(text);
		this.insertAdjacentElement(position, textNode);
	}

	/**
	 * Sets an attribute.
	 *
	 * @param name Name.
	 * @param value Value.
	 */
	public setAttribute(name: string, value: string): void {
		const namespaceURI = this[PropertySymbol.namespaceURI];
		// TODO: Is it correct to check for namespaceURI === NamespaceURI.svg?
		const attribute =
			namespaceURI === NamespaceURI.html &&
			this[PropertySymbol.ownerDocument][PropertySymbol.contentType] === 'text/html'
				? this[PropertySymbol.ownerDocument].createAttribute(name)
				: this[PropertySymbol.ownerDocument].createAttributeNS(null, name);
		attribute[PropertySymbol.value] = String(value);
		this[PropertySymbol.attributes].setNamedItem(attribute);
	}

	/**
	 * Sets a namespace attribute.
	 *
	 * @param namespaceURI Namespace URI.
	 * @param name Name.
	 * @param value Value.
	 */
	public setAttributeNS(namespaceURI: string, name: string, value: string): void {
		const attribute = this[PropertySymbol.ownerDocument].createAttributeNS(namespaceURI, name);
		attribute[PropertySymbol.value] = String(value);
		this[PropertySymbol.attributes].setNamedItemNS(attribute);
	}

	/**
	 * Returns attribute names.
	 *
	 * @returns Attribute names.
	 */
	public getAttributeNames(): string[] {
		const names = [];
		for (const item of this[PropertySymbol.attributes][PropertySymbol.namespaceItems].values()) {
			names.push(item[PropertySymbol.name]);
		}
		return names;
	}

	/**
	 * Returns attribute value.
	 *
	 * @param name Name.
	 */
	public getAttribute(name: string): string {
		const attribute = this[PropertySymbol.attributes].getNamedItem(name);
		if (attribute) {
			return attribute[PropertySymbol.value];
		}
		return null;
	}

	/**
	 * Toggle an attribute.
	 * Returns `true` if attribute name is eventually present, and `false` otherwise.
	 *
	 * @param name A DOMString specifying the name of the attribute to be toggled.
	 * @param force A boolean value to determine whether the attribute should be added or removed, no matter whether the attribute is present or not at the moment.
	 */
	public toggleAttribute(name: string, force?: boolean): boolean {
		name = name.toLowerCase();
		const attribute = this[PropertySymbol.attributes].getNamedItem(name);
		if (attribute) {
			if (force === true) {
				return true;
			}
			this[PropertySymbol.attributes][PropertySymbol.removeNamedItem](attribute);
			return false;
		}
		if (force === false) {
			return false;
		}
		this.setAttribute(name, '');
		return true;
	}

	/**
	 * Returns namespace attribute value.
	 *
	 * @param namespace Namespace URI.
	 * @param localName Local name.
	 */
	public getAttributeNS(namespace: string | null, localName: string): string {
		const attribute = this.getAttributeNodeNS(namespace, localName);
		if (attribute) {
			return attribute[PropertySymbol.value];
		}
		return null;
	}

	/**
	 * Returns a boolean value indicating whether the specified element has the attribute or not.
	 *
	 * @param name Attribute name.
	 * @returns True if attribute exists, false otherwise.
	 */
	public hasAttribute(name: string): boolean {
		return !!this.getAttributeNode(name);
	}

	/**
	 * Returns a boolean value indicating whether the specified element has the namespace attribute or not.
	 *
	 * @param namespace Namespace URI.
	 * @param localName Local name.
	 * @returns True if attribute exists, false otherwise.
	 */
	public hasAttributeNS(namespace: string | null, localName: string): boolean {
		return this[PropertySymbol.attributes].getNamedItemNS(namespace, localName) !== null;
	}

	/**
	 * Returns "true" if the element has attributes.
	 *
	 * @returns "true" if the element has attributes.
	 */
	public hasAttributes(): boolean {
		return this[PropertySymbol.attributes][PropertySymbol.namespaceItems].size > 0;
	}

	/**
	 * Removes an attribute.
	 *
	 * @param name Name.
	 */
	public removeAttribute(name: string): void {
		const item = this[PropertySymbol.attributes].getNamedItem(name);
		if (item) {
			this[PropertySymbol.attributes][PropertySymbol.removeNamedItem](item);
		}
	}

	/**
	 * Removes a namespace attribute.
	 *
	 * @param namespace Namespace URI.
	 * @param localName Local name.
	 */
	public removeAttributeNS(namespace: string | null, localName: string): void {
		const item = this[PropertySymbol.attributes].getNamedItemNS(namespace, localName);
		if (item) {
			this[PropertySymbol.attributes][PropertySymbol.removeNamedItem](item);
		}
	}

	/**
	 * Attaches a shadow root.
	 *
	 * @param init Shadow root init.
	 * @param init.mode Shadow root mode.
	 * @param [init.clonable] Clonable.
	 * @param [init.delegateFocus] Delegate focus.
	 * @param [init.serializable] Serializable.
	 * @param [init.slotAssignment] Slot assignment.
	 * @returns Shadow root.
	 */
	public attachShadow(init: {
		mode: 'open' | 'closed';
		clonable?: boolean;
		delegateFocus?: boolean;
		serializable?: boolean;
		slotAssignment?: 'named' | 'manual';
	}): ShadowRoot {
		const window = this[PropertySymbol.window];

		if (!init) {
			throw new window.TypeError(
				"Failed to execute 'attachShadow' on 'Element': 1 argument required, but only 0 present."
			);
		}

		if (!init.mode) {
			throw new window.TypeError(
				"Failed to execute 'attachShadow' on 'Element': Failed to read the 'mode' property from 'ShadowRootInit': Required member is undefined."
			);
		}

		if (init.mode !== 'open' && init.mode !== 'closed') {
			throw new window.TypeError(
				`Failed to execute 'attachShadow' on 'Element': Failed to read the 'mode' property from 'ShadowRootInit': The provided value '${init.mode}' is not a valid enum value of type ShadowRootMode.`
			);
		}

		if (this[PropertySymbol.shadowRoot]) {
			throw new window.DOMException(
				"Failed to execute 'attachShadow' on 'Element': Shadow root cannot be created on a host which already hosts a shadow tree."
			);
		}

		const shadowRoot = NodeFactory.createNode(
			this[PropertySymbol.ownerDocument],
			this[PropertySymbol.window].ShadowRoot
		);

		this[PropertySymbol.shadowRoot] = shadowRoot;

		shadowRoot[PropertySymbol.host] = this;
		shadowRoot[PropertySymbol.mode] = init.mode;
		shadowRoot[PropertySymbol.clonable] = !!init.clonable;
		shadowRoot[PropertySymbol.delegatesFocus] = !!init.delegateFocus;
		shadowRoot[PropertySymbol.serializable] = !!init.serializable;
		shadowRoot[PropertySymbol.slotAssignment] =
			init.slotAssignment === 'manual' ? 'manual' : 'named';

		shadowRoot[PropertySymbol.connectedToNode]();

		return this[PropertySymbol.shadowRoot];
	}

	/**
	 * Converts to string.
	 *
	 * @returns String.
	 */
	public toString(): string {
		return this.outerHTML;
	}

	/**
	 * Returns the size of an element and its position relative to the viewport.
	 *
	 * @returns DOM rect.
	 */
	public getBoundingClientRect(): DOMRect {
		// TODO: Not full implementation
		return new DOMRect();
	}

	/**
	 * Returns a collection of DOMRect objects that indicate the bounding rectangles for each CSS border box in a client.
	 *
	 * @see https://developer.mozilla.org/en-US/docs/Web/API/Element/getClientRects
	 * @returns DOM rect list.
	 */
	public getClientRects(): DOMRectList {
		// TODO: Not full implementation
		const domRectList = new DOMRectList(PropertySymbol.illegalConstructor);
		domRectList.push(this.getBoundingClientRect());
		return domRectList;
	}

	/**
	 * The matches() method checks to see if the Element would be selected by the provided selectorString.
	 *
	 * @param selector Selector.
	 * @returns "true" if matching.
	 */
	public matches(selector: string): boolean {
		return !!QuerySelector.matches(this, selector);
	}

	/**
	 * Traverses the Element and its parents (heading toward the document root) until it finds a node that matches the provided selector string.
	 *
	 * @param selector Selector.
	 * @returns Closest matching element.
	 */
	public closest(selector: string): Element {
		// eslint-disable-next-line
		let parent: Element = this;

		while (parent) {
			if (QuerySelector.matches(parent, selector)) {
				return parent;
			}
			parent = parent.parentElement;
		}

		return null;
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
	 * Query CSS Selector to find matching node.
	 *
	 * @param selector CSS selector.
	 * @returns Matching element.
	 */
	public querySelector<K extends keyof ISVGElementTagNameMap>(
		selector: K
	): ISVGElementTagNameMap[K] | null;

	/**
	 * Query CSS Selector to find matching node.
	 *
	 * @param selector CSS selector.
	 * @returns Matching element.
	 */
	public querySelector(selector: string): Element | null;

	/**
	 * Query CSS Selector to find matching node.
	 *
	 * @param selector CSS selector.
	 * @returns Matching element.
	 */
	public querySelector(selector: string): Element | null {
		return QuerySelector.querySelector(this, selector);
	}

	/**
	 * Returns an elements by class name.
	 *
	 * @param className Tag name.
	 * @returns Matching element.
	 */
	public getElementsByClassName(className: string): HTMLCollection<Element> {
		return ParentNodeUtility.getElementsByClassName(this, className);
	}

	/**
	 * Returns an elements by tag name.
	 *
	 * @param tagName Tag name.
	 * @returns Matching element.
	 */
	public getElementsByTagName<K extends keyof IHTMLElementTagNameMap>(
		tagName: K
	): HTMLCollection<IHTMLElementTagNameMap[K]>;

	/**
	 * Returns an elements by tag name.
	 *
	 * @param tagName Tag name.
	 * @returns Matching element.
	 */
	public getElementsByTagName<K extends keyof ISVGElementTagNameMap>(
		tagName: K
	): HTMLCollection<ISVGElementTagNameMap[K]>;

	/**
	 * Returns an elements by tag name.
	 *
	 * @param tagName Tag name.
	 * @returns Matching element.
	 */
	public getElementsByTagName(tagName: string): HTMLCollection<Element>;

	/**
	 * Returns an elements by tag name.
	 *
	 * @param tagName Tag name.
	 * @returns Matching element.
	 */
	public getElementsByTagName(tagName: string): HTMLCollection<Element> {
		return ParentNodeUtility.getElementsByTagName(this, tagName);
	}

	/**
	 * Returns an elements by tag name and namespace.
	 *
	 * @param namespaceURI Namespace URI.
	 * @param tagName Tag name.
	 * @returns Matching element.
	 */
	public getElementsByTagNameNS<K extends keyof IHTMLElementTagNameMap>(
		namespaceURI: 'http://www.w3.org/1999/xhtml',
		tagName: K
	): HTMLCollection<IHTMLElementTagNameMap[K]>;

	/**
	 * Returns an elements by tag name and namespace.
	 *
	 * @param namespaceURI Namespace URI.
	 * @param tagName Tag name.
	 * @returns Matching element.
	 */
	public getElementsByTagNameNS<K extends keyof ISVGElementTagNameMap>(
		namespaceURI: 'http://www.w3.org/2000/svg',
		tagName: K
	): HTMLCollection<ISVGElementTagNameMap[K]>;

	/**
	 * Returns an elements by tag name and namespace.
	 *
	 * @param namespaceURI Namespace URI.
	 * @param tagName Tag name.
	 * @returns Matching element.
	 */
	public getElementsByTagNameNS(namespaceURI: string, tagName: string): HTMLCollection<Element>;

	/**
	 * Returns an elements by tag name and namespace.
	 *
	 * @param namespaceURI Namespace URI.
	 * @param tagName Tag name.
	 * @returns Matching element.
	 */
	public getElementsByTagNameNS(namespaceURI: string, tagName: string): HTMLCollection<Element> {
		return ParentNodeUtility.getElementsByTagNameNS(this, namespaceURI, tagName);
	}

	/**
	 * The setAttributeNode() method adds a new Attr node to the specified element.
	 *
	 * @param attribute Attribute.
	 * @returns Replaced attribute.
	 */
	public setAttributeNode(attribute: Attr): Attr | null {
		return this[PropertySymbol.attributes].setNamedItem(attribute);
	}

	/**
	 * The setAttributeNodeNS() method adds a new Attr node to the specified element.
	 *
	 * @param attribute Attribute.
	 * @returns Replaced attribute.
	 */
	public setAttributeNodeNS(attribute: Attr): Attr | null {
		return this[PropertySymbol.attributes].setNamedItemNS(attribute);
	}

	/**
	 * Returns an Attr node.
	 *
	 * @param name Name.
	 * @returns Replaced attribute.
	 */
	public getAttributeNode(name: string): Attr | null {
		return this[PropertySymbol.attributes].getNamedItem(name);
	}

	/**
	 * Returns a namespaced Attr node.
	 *
	 * @param namespace Namespace.
	 * @param localName Name.
	 * @returns Replaced attribute.
	 */
	public getAttributeNodeNS(namespace: string | null, localName: string): Attr | null {
		return this[PropertySymbol.attributes].getNamedItemNS(namespace, localName);
	}

	/**
	 * Removes an Attr node.
	 *
	 * @param attribute Attribute.
	 * @returns Removed attribute.
	 */
	public removeAttributeNode(attribute: Attr): Attr | null {
		if (attribute[PropertySymbol.ownerElement] !== this) {
			throw new this[PropertySymbol.window].DOMException(
				"Failed to execute 'removeAttributeNode' on 'Element': The node provided is owned by another element."
			);
		}
		this[PropertySymbol.attributes][PropertySymbol.removeNamedItem](attribute);
		return attribute;
	}

	/**
	 * Scrolls to a particular set of coordinates.
	 *
	 * @param x X position or options object.
	 * @param y Y position.
	 */
	public scroll(x: IScrollToOptions | number, y?: number): void {
		if (typeof x !== 'object' && arguments.length === 1) {
			throw new this[PropertySymbol.window].TypeError(
				"Failed to execute 'scroll' on 'Element': The provided value is not of type 'ScrollToOptions'."
			);
		}

		const options = typeof x === 'object' ? x : { left: x, top: y };

		if (options.behavior === 'smooth') {
			this[PropertySymbol.window].setTimeout(() => {
				if (options.top !== undefined) {
					const top = Number(options.top);
					(<number>this.scrollTop) = isNaN(top) ? 0 : top;
				}
				if (options.left !== undefined) {
					const left = Number(options.left);
					(<number>this.scrollLeft) = isNaN(left) ? 0 : left;
				}
			});
		} else {
			if (options.top !== undefined) {
				const top = Number(options.top);
				(<number>this.scrollTop) = isNaN(top) ? 0 : top;
			}
			if (options.left !== undefined) {
				const left = Number(options.left);
				(<number>this.scrollLeft) = isNaN(left) ? 0 : left;
			}
		}
	}

	/**
	 * Scrolls to a particular set of coordinates.
	 *
	 * @param x X position or options object.
	 * @param y Y position.
	 */
	public scrollTo(x: IScrollToOptions | number, y?: number): void {
		if (typeof x !== 'object' && arguments.length === 1) {
			throw new this[PropertySymbol.window].TypeError(
				"Failed to execute 'scrollTo' on 'Element': The provided value is not of type 'ScrollToOptions'."
			);
		}
		this.scroll(x, y);
	}

	/**
	 * Scrolls by a relative amount from the current position.
	 *
	 * @param x Pixels to scroll by from top or scroll options object.
	 * @param y Pixels to scroll by from left.
	 */
	public scrollBy(x: IScrollToOptions | number, y?: number): void {
		if (typeof x !== 'object' && arguments.length === 1) {
			throw new this[PropertySymbol.window].TypeError(
				"Failed to execute 'scrollBy' on 'Element': The provided value is not of type 'ScrollToOptions'."
			);
		}
		const options = typeof x === 'object' ? x : { left: x, top: y };
		this.scroll({
			left: this.scrollLeft + (options.left ?? 0),
			top: this.scrollTop + (options.top ?? 0),
			behavior: options.behavior
		});
	}

	/**
	 * Scrolls the element's ancestor containers such that the element on which scrollIntoView() is called is visible to the user.
	 *
	 * @param [_options] Options.
	 */
	public scrollIntoView(
		_options?:
			| boolean
			| {
					behavior?: 'smooth' | 'instant' | 'auto';
					block?: 'start' | 'center' | 'end' | 'nearest';
					inline?: 'start' | 'center' | 'end' | 'nearest';
			  }
	): void {
		// Do nothing
	}

	/**
	 * @override
	 */
	public override dispatchEvent(event: Event): boolean {
		const returnValue = super.dispatchEvent(event);
		const browserSettings = new WindowBrowserContext(this[PropertySymbol.window]).getSettings();

		if (
			browserSettings &&
			!browserSettings.disableJavaScriptEvaluation &&
			event.eventPhase === EventPhaseEnum.none &&
			!event[PropertySymbol.immediatePropagationStopped]
		) {
			const attribute = this.getAttribute('on' + event.type);

			if (attribute && !event[PropertySymbol.immediatePropagationStopped]) {
				const code = `//# sourceURL=${this[PropertySymbol.window].location.href}\n${attribute}`;

				if (
					browserSettings.disableErrorCapturing ||
					browserSettings.errorCapture !== BrowserErrorCaptureEnum.tryAndCatch
				) {
					this[PropertySymbol.window].eval(code);
				} else {
					WindowErrorUtility.captureError(this[PropertySymbol.window], () =>
						this[PropertySymbol.window].eval(code)
					);
				}
			}
		}

		return returnValue;
	}

	/**
	 * @override
	 */
	public override [PropertySymbol.appendChild](node: Node, disableValidations = false): Node {
		const returnValue = super[PropertySymbol.appendChild](node, disableValidations);
		this.#onSlotChange(node);
		return returnValue;
	}

	/**
	 * @override
	 */
	public override [PropertySymbol.removeChild](node: Node): Node {
		const returnValue = super[PropertySymbol.removeChild](node);
		this.#onSlotChange(node);
		return returnValue;
	}

	/**
	 * @override
	 */
	public override [PropertySymbol.insertBefore](
		newNode: Node,
		referenceNode: Node | null,
		disableValidations = false
	): Node {
		const returnValue = super[PropertySymbol.insertBefore](
			newNode,
			referenceNode,
			disableValidations
		);
		this.#onSlotChange(newNode);
		return returnValue;
	}

	/**
	 * Triggered when an attribute is set.
	 *
	 * @param attribute Attribute.
	 * @param replacedAttribute Replaced attribute.
	 */
	public [PropertySymbol.onSetAttribute](attribute: Attr, replacedAttribute: Attr | null): void {
		if (!attribute[PropertySymbol.name]) {
			return null;
		}

		const oldValue = replacedAttribute ? replacedAttribute[PropertySymbol.value] : null;

		if (
			attribute[PropertySymbol.name] === 'slot' &&
			this[PropertySymbol.parentNode] &&
			this[PropertySymbol.parentNode][PropertySymbol.shadowRoot]
		) {
			const shadowRoot = this[PropertySymbol.parentNode][PropertySymbol.shadowRoot];

			if (attribute[PropertySymbol.value] !== oldValue) {
				// Previous slot
				if (oldValue !== null) {
					const slot = shadowRoot.querySelector(
						`slot[name="${replacedAttribute[PropertySymbol.value]}"]`
					);
					if (slot) {
						slot.dispatchEvent(new Event('slotchange', { bubbles: true }));
					}
				} else {
					const slot = shadowRoot.querySelector('slot:not([name])');
					if (slot) {
						slot.dispatchEvent(new Event('slotchange', { bubbles: true }));
					}
				}

				// New slot
				const slot = shadowRoot.querySelector(`slot[name="${attribute[PropertySymbol.value]}"]`);
				if (slot) {
					slot.dispatchEvent(new Event('slotchange', { bubbles: true }));
				}
			}
		}

		if (attribute[PropertySymbol.name] === 'id' && this[PropertySymbol.isConnected]) {
			if (replacedAttribute?.[PropertySymbol.value]) {
				this.#removeIdentifierFromWindow(replacedAttribute[PropertySymbol.value]);
			}
			this.#addIdentifierToWindow(attribute[PropertySymbol.value]);
		}

		this[PropertySymbol.reportMutation](
			new MutationRecord({
				target: this,
				type: MutationTypeEnum.attributes,
				attributeName: attribute[PropertySymbol.name],
				oldValue
			})
		);
	}

	/**
	 * Triggered when an attribute is set.
	 *
	 * @param removedAttribute Attribute.
	 */
	public [PropertySymbol.onRemoveAttribute](removedAttribute: Attr): void {
		if (
			removedAttribute[PropertySymbol.name] === 'slot' &&
			this[PropertySymbol.parentNode] &&
			this[PropertySymbol.parentNode][PropertySymbol.shadowRoot]
		) {
			const shadowRoot = this[PropertySymbol.parentNode][PropertySymbol.shadowRoot];
			const namedSlot = shadowRoot.querySelector(
				`slot[name="${removedAttribute[PropertySymbol.value]}"]`
			);
			const defaultSlot = shadowRoot.querySelector('slot:not([name])');
			if (namedSlot) {
				namedSlot.dispatchEvent(new Event('slotchange', { bubbles: true }));
			}
			if (defaultSlot) {
				defaultSlot.dispatchEvent(new Event('slotchange', { bubbles: true }));
			}
		}

		if (removedAttribute[PropertySymbol.name] === 'id' && this[PropertySymbol.isConnected]) {
			this.#removeIdentifierFromWindow(removedAttribute[PropertySymbol.value]);
		}

		this[PropertySymbol.reportMutation](
			new MutationRecord({
				type: MutationTypeEnum.attributes,
				target: this,
				attributeName: removedAttribute[PropertySymbol.name],
				oldValue: removedAttribute[PropertySymbol.value]
			})
		);
	}

	/**
	 * @override
	 */
	public override [PropertySymbol.connectedToDocument](): void {
		const id = this.getAttribute('id');
		if (id) {
			this.#addIdentifierToWindow(id);
		}

		super[PropertySymbol.connectedToDocument]();
	}

	/**
	 * @override
	 */
	public override [PropertySymbol.disconnectedFromDocument](): void {
		super[PropertySymbol.disconnectedFromDocument]();

		const id = this.getAttribute('id');
		if (id) {
			this.#removeIdentifierFromWindow(id);
		}
	}

	/**
	 * Adds identifier to the window object.
	 *
	 * @param id Identifier.
	 */
	#addIdentifierToWindow(id: string): void {
		if (!id) {
			return;
		}

		const document = this[PropertySymbol.ownerDocument];
		const window = this[PropertySymbol.window];

		// We should not add the identifier when inside a shadow root
		if (this[PropertySymbol.rootNode] && this[PropertySymbol.rootNode] !== document) {
			return;
		}

		if (!document[PropertySymbol.elementIdMap].has(id)) {
			document[PropertySymbol.elementIdMap].set(id, { elements: [], htmlCollection: null });
		}

		const entry = document[PropertySymbol.elementIdMap].get(id);

		// HTMLFormElement and HTMLSelectElement can be a proxy, but the scope can be the target and not the actual proxy
		// To make sure we use the proxy we can check for the proxy property
		const element = this[PropertySymbol.proxy] || this;

		entry.elements.push(element);

		if (entry.elements.length > 1) {
			if (!entry.htmlCollection) {
				entry.htmlCollection = new HTMLCollection<Element>(
					PropertySymbol.illegalConstructor,
					() => entry.elements
				);
			}

			if (!(id in window) || window[id] === entry.elements[0]) {
				window[id] = entry.htmlCollection;
			}
		} else if (!(id in window) || window[id] === entry.htmlCollection) {
			window[id] = element;
		}
	}

	/**
	 * Removes identifier from the window object.
	 *
	 * @param id Identifier.
	 */
	#removeIdentifierFromWindow(id: string): void {
		if (!id) {
			return;
		}

		const document = this[PropertySymbol.ownerDocument];
		const window = this[PropertySymbol.window];

		// We should not add the identifier when inside a shadow root
		if (this[PropertySymbol.rootNode] && this[PropertySymbol.rootNode] !== document) {
			return;
		}

		const entry = document[PropertySymbol.elementIdMap].get(id);

		if (entry) {
			// HTMLFormElement and HTMLSelectElement can be a proxy, but the scope can be the target and not the actual proxy
			// To make sure we use the proxy we can check for the proxy property
			const element = this[PropertySymbol.proxy] || this;
			const index = entry.elements.indexOf(element);

			if (index !== -1) {
				entry.elements.splice(index, 1);
			}

			if (entry.elements.length === 1) {
				if (window[id] === entry.htmlCollection) {
					window[id] = entry.elements[0];
				}

				entry.htmlCollection = null;
			} else if (!entry.elements.length) {
				document[PropertySymbol.elementIdMap].delete(id);

				if (window[id] === element || window[id] === entry.htmlCollection) {
					delete window[id];
				}
			}
		}
	}

	/**
	 * Triggered when child nodes are changed.
	 *
	 * @param addedOrRemovedNode Changed node.
	 */
	#onSlotChange(addedOrRemovedNode: Node): void {
		const shadowRoot = this[PropertySymbol.shadowRoot];

		if (!shadowRoot) {
			return;
		}

		const slotName = addedOrRemovedNode['getAttribute']
			? (<Element>addedOrRemovedNode).getAttribute('slot')
			: null;
		if (slotName) {
			const slot = shadowRoot.querySelector(`slot[name="${slotName}"]`);
			if (slot) {
				slot.dispatchEvent(new Event('slotchange', { bubbles: true }));
			}
		} else if (addedOrRemovedNode[PropertySymbol.nodeType] !== NodeTypeEnum.commentNode) {
			const slot = shadowRoot.querySelector('slot:not([name])');
			if (slot) {
				slot.dispatchEvent(new Event('slotchange', { bubbles: true }));
			}
		}
	}
}
