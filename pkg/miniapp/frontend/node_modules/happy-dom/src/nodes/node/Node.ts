import EventTarget from '../../event/EventTarget.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import Document from '../document/Document.js';
import Element from '../element/Element.js';
import NodeTypeEnum from './NodeTypeEnum.js';
import NodeDocumentPositionEnum from './NodeDocumentPositionEnum.js';
import NodeUtility from './NodeUtility.js';
import Attr from '../attr/Attr.js';
import NodeList from './NodeList.js';
import MutationRecord from '../../mutation-observer/MutationRecord.js';
import MutationTypeEnum from '../../mutation-observer/MutationTypeEnum.js';
import DOMExceptionNameEnum from '../../exception/DOMExceptionNameEnum.js';
import IMutationListener from '../../mutation-observer/IMutationListener.js';
import ICachedQuerySelectorAllResult from './ICachedQuerySelectorAllResult.js';
import ICachedQuerySelectorResult from './ICachedQuerySelectorResult.js';
import ICachedMatchesResult from './ICachedMatchesResult.js';
import ICachedElementsByTagNameResult from './ICachedElementsByTagNameResult.js';
import ICachedElementByTagNameResult from './ICachedElementByTagNameResult.js';
import ICachedComputedStyleResult from './ICachedComputedStyleResult.js';
import ICachedResult from './ICachedResult.js';
import ICachedElementByIdResult from './ICachedElementByIdResult.js';
import HTMLStyleElement from '../html-style-element/HTMLStyleElement.js';
import HTMLFormElement from '../html-form-element/HTMLFormElement.js';
import HTMLSelectElement from '../html-select-element/HTMLSelectElement.js';
import HTMLTextAreaElement from '../html-text-area-element/HTMLTextAreaElement.js';
import HTMLSlotElement from '../html-slot-element/HTMLSlotElement.js';
import NodeFactory from '../NodeFactory.js';
import SVGStyleElement from '../svg-style-element/SVGStyleElement.js';

/**
 * Node.
 */
export default class Node extends EventTarget {
	// Can be injected by CustomElementRegistry or a sub-class
	public declare [PropertySymbol.ownerDocument]: Document;

	// Public properties
	public static readonly ELEMENT_NODE = NodeTypeEnum.elementNode;
	public static readonly ATTRIBUTE_NODE = NodeTypeEnum.attributeNode;
	public static readonly TEXT_NODE = NodeTypeEnum.textNode;
	public static readonly CDATA_SECTION_NODE = NodeTypeEnum.cdataSectionNode;
	public static readonly COMMENT_NODE = NodeTypeEnum.commentNode;
	public static readonly DOCUMENT_NODE = NodeTypeEnum.documentNode;
	public static readonly DOCUMENT_TYPE_NODE = NodeTypeEnum.documentTypeNode;
	public static readonly DOCUMENT_FRAGMENT_NODE = NodeTypeEnum.documentFragmentNode;
	public static readonly PROCESSING_INSTRUCTION_NODE = NodeTypeEnum.processingInstructionNode;
	public static readonly DOCUMENT_POSITION_CONTAINED_BY = NodeDocumentPositionEnum.containedBy;
	public static readonly DOCUMENT_POSITION_CONTAINS = NodeDocumentPositionEnum.contains;
	public static readonly DOCUMENT_POSITION_DISCONNECTED = NodeDocumentPositionEnum.disconnect;
	public static readonly DOCUMENT_POSITION_FOLLOWING = NodeDocumentPositionEnum.following;
	public static readonly DOCUMENT_POSITION_IMPLEMENTATION_SPECIFIC =
		NodeDocumentPositionEnum.implementationSpecific;
	public static readonly DOCUMENT_POSITION_PRECEDING = NodeDocumentPositionEnum.preceding;

	// Defined on the prototype
	public declare readonly ELEMENT_NODE;
	public declare readonly ATTRIBUTE_NODE;
	public declare readonly TEXT_NODE;
	public declare readonly CDATA_SECTION_NODE;
	public declare readonly COMMENT_NODE;
	public declare readonly DOCUMENT_NODE;
	public declare readonly DOCUMENT_TYPE_NODE;
	public declare readonly DOCUMENT_FRAGMENT_NODE;
	public declare readonly PROCESSING_INSTRUCTION_NODE;
	public declare readonly DOCUMENT_POSITION_CONTAINED_BY;
	public declare readonly DOCUMENT_POSITION_CONTAINS;
	public declare readonly DOCUMENT_POSITION_DISCONNECTED;
	public declare readonly DOCUMENT_POSITION_FOLLOWING;
	public declare readonly DOCUMENT_POSITION_IMPLEMENTATION_SPECIFIC;
	public declare readonly DOCUMENT_POSITION_PRECEDING;

	// Internal properties
	public [PropertySymbol.isConnected] = false;
	public [PropertySymbol.parentNode]: Node | null = null;
	public [PropertySymbol.nodeType]: NodeTypeEnum;
	public [PropertySymbol.rootNode]: Node = null;
	public [PropertySymbol.styleNode]: HTMLStyleElement | SVGStyleElement | null = null;
	public [PropertySymbol.textAreaNode]: HTMLTextAreaElement | null = null;
	public [PropertySymbol.formNode]: HTMLFormElement | null = null;
	public [PropertySymbol.selectNode]: HTMLSelectElement | null = null;
	public [PropertySymbol.mutationListeners]: IMutationListener[] = [];
	public [PropertySymbol.nodeArray]: Node[] = [];
	public [PropertySymbol.elementArray]: Element[] = [];
	public [PropertySymbol.childNodes]: NodeList<Node> | null = null;
	public [PropertySymbol.assignedToSlot]: HTMLSlotElement | null = null;
	public [PropertySymbol.cache]: {
		querySelector: Map<string, ICachedQuerySelectorResult>;
		querySelectorAll: Map<string, ICachedQuerySelectorAllResult>;
		matches: Map<string, ICachedMatchesResult>;
		elementsByTagName: Map<string, ICachedElementsByTagNameResult>;
		elementsByTagNameNS: Map<string, ICachedElementsByTagNameResult>;
		elementByTagName: Map<string, ICachedElementByTagNameResult>;
		elementById: Map<string, ICachedElementByIdResult>;
		computedStyle: ICachedComputedStyleResult | null;
	} = {
		querySelector: new Map(),
		querySelectorAll: new Map(),
		matches: new Map(),
		elementsByTagName: new Map(),
		elementsByTagNameNS: new Map(),
		elementByTagName: new Map(),
		elementById: new Map(),
		computedStyle: null
	};
	public [PropertySymbol.affectsCache]: ICachedResult[] = [];

	/**
	 * Constructor.
	 */
	constructor() {
		super();

		// Window injected by WindowContextClassExtender (used when the Node can be created using "new" keyword)
		if (this[PropertySymbol.window]) {
			this[PropertySymbol.ownerDocument] = this[PropertySymbol.window].document;
		} else {
			const ownerDocument = NodeFactory.pullOwnerDocument();

			if (!ownerDocument) {
				throw new TypeError('Illegal constructor');
			}

			this[PropertySymbol.ownerDocument] = ownerDocument;
			this[PropertySymbol.window] = ownerDocument[PropertySymbol.window];
		}
	}

	/**
	 * Returns `Symbol.toStringTag`.
	 *
	 * @returns `Symbol.toStringTag`.
	 */
	public get [Symbol.toStringTag](): string {
		return this.constructor.name;
	}

	/**
	 * Returns connected state.
	 *
	 * @returns Connected state.
	 */
	public get isConnected(): boolean {
		return this[PropertySymbol.isConnected];
	}

	/**
	 * Returns owner document.
	 *
	 * @returns Owner document.
	 */
	public get ownerDocument(): Document {
		return this[PropertySymbol.ownerDocument];
	}

	/**
	 * Returns parent node.
	 *
	 * @returns Parent node.
	 */
	public get parentNode(): Node | null {
		return this[PropertySymbol.parentNode];
	}

	/**
	 * Returns node type.
	 *
	 * @returns Node type.
	 */
	public get nodeType(): number {
		return this[PropertySymbol.nodeType];
	}

	/**
	 * Get child nodes.
	 *
	 * @returns Child nodes list.
	 */
	public get childNodes(): NodeList<Node> {
		if (!this[PropertySymbol.childNodes]) {
			this[PropertySymbol.childNodes] = new NodeList<Node>(
				PropertySymbol.illegalConstructor,
				this[PropertySymbol.nodeArray]
			);
		}
		return this[PropertySymbol.childNodes];
	}

	/**
	 * Get text value of children.
	 *
	 * @returns Text content.
	 */
	public get textContent(): string {
		// Sub-classes should implement this method.
		return null;
	}

	/**
	 * Sets text content.
	 *
	 * @param _textContent Text content.
	 */
	public set textContent(_textContent) {
		// Do nothing.
		// Sub-classes should implement this method.
	}

	/**
	 * Node value.
	 *
	 * @returns Node value.
	 */
	public get nodeValue(): string {
		return null;
	}

	/**
	 * Sets node value.
	 */
	public set nodeValue(_nodeValue: string) {
		// Do nothing
	}

	/**
	 * Node name.
	 *
	 * @returns Node name.
	 */
	public get nodeName(): string {
		return '';
	}

	/**
	 * Previous sibling.
	 *
	 * @returns Node.
	 */
	public get previousSibling(): Node {
		if (this[PropertySymbol.parentNode]) {
			const nodeArray = this[PropertySymbol.parentNode][PropertySymbol.nodeArray];
			const index = nodeArray.indexOf(this);
			if (index > 0) {
				return nodeArray[index - 1];
			}
		}
		return null;
	}

	/**
	 * Next sibling.
	 *
	 * @returns Node.
	 */
	public get nextSibling(): Node {
		if (this[PropertySymbol.parentNode]) {
			const nodeArray = this[PropertySymbol.parentNode][PropertySymbol.nodeArray];
			const index = nodeArray.indexOf(this);
			if (index > -1 && index + 1 < nodeArray.length) {
				return nodeArray[index + 1];
			}
		}
		return null;
	}

	/**
	 * First child.
	 *
	 * @returns Node.
	 */
	public get firstChild(): Node {
		const nodeArray = this[PropertySymbol.nodeArray];
		if (nodeArray.length > 0) {
			return nodeArray[0];
		}
		return null;
	}

	/**
	 * Last child.
	 *
	 * @returns Node.
	 */
	public get lastChild(): Node {
		const nodeArray = this[PropertySymbol.nodeArray];
		if (nodeArray.length > 0) {
			return nodeArray[nodeArray.length - 1];
		}
		return null;
	}

	/**
	 * Returns parent element.
	 *
	 * @returns Element.
	 */
	public get parentElement(): Element | null {
		let parent = this[PropertySymbol.parentNode];
		while (parent && parent[PropertySymbol.nodeType] !== NodeTypeEnum.elementNode) {
			parent = parent[PropertySymbol.parentNode];
		}
		return <Element>parent;
	}

	/**
	 * Returns base URI.
	 *
	 * @returns Base URI.
	 */
	public get baseURI(): string {
		const base = this[PropertySymbol.ownerDocument].querySelector('base');
		if (base) {
			return base.href;
		}
		return this[PropertySymbol.window].location.href;
	}

	/**
	 * Returns "true" if the node has child nodes.
	 *
	 * @returns "true" if the node has child nodes.
	 */
	public hasChildNodes(): boolean {
		return this[PropertySymbol.nodeArray].length > 0;
	}

	/**
	 * Returns "true" if this node contains the other node.
	 *
	 * @param otherNode Node to test with.
	 * @returns "true" if this node contains the other node.
	 */
	public contains(otherNode: Node): boolean {
		if (otherNode === undefined) {
			return false;
		}
		return NodeUtility.isInclusiveAncestor(this, otherNode);
	}

	/**
	 * Returns closest root node (Document or ShadowRoot).
	 *
	 * @param options Options.
	 * @param options.composed A Boolean that indicates whether the shadow root should be returned (false, the default), or a root node beyond shadow root (true).
	 * @returns Node.
	 */
	public getRootNode(options?: { composed: boolean }): Node {
		if (!this[PropertySymbol.isConnected]) {
			return this;
		}

		if (this[PropertySymbol.rootNode] && !options?.composed) {
			return this[PropertySymbol.rootNode];
		}

		return this[PropertySymbol.ownerDocument];
	}

	/**
	 * Clones a node.
	 *
	 * @param [deep=false] "true" to clone deep.
	 * @returns Cloned node.
	 */
	public cloneNode(deep = false): Node {
		return this[PropertySymbol.cloneNode](deep);
	}

	/**
	 * Append a child node to childNodes.
	 *
	 * @param  node Node to append.
	 * @returns Appended node.
	 */
	public appendChild(node: Node): Node {
		if (arguments.length < 1) {
			throw new this[PropertySymbol.window].TypeError(
				`Failed to execute 'appendChild' on 'Node': 1 argument required, but only 0 present`
			);
		}
		return this[PropertySymbol.appendChild](node);
	}

	/**
	 * Remove Child element from childNodes array.
	 *
	 * @param node Node to remove.
	 * @returns Removed node.
	 */
	public removeChild(node: Node): Node {
		if (arguments.length < 1) {
			throw new this[PropertySymbol.window].TypeError(
				`Failed to execute 'removeChild' on 'Node': 1 argument required, but only 0 present`
			);
		}
		return this[PropertySymbol.removeChild](node);
	}

	/**
	 * Inserts a node before another.
	 *
	 * @param newNode Node to insert.
	 * @param referenceNode Node to insert before.
	 * @returns Inserted node.
	 */
	public insertBefore(newNode: Node, referenceNode: Node | null): Node {
		if (arguments.length < 2) {
			throw new this[PropertySymbol.window].TypeError(
				`Failed to execute 'insertBefore' on 'Node': 2 arguments required, but only ${arguments.length} present.`
			);
		}
		return this[PropertySymbol.insertBefore](newNode, referenceNode);
	}

	/**
	 * Replaces a node with another.
	 *
	 * @param newChild New child.
	 * @param oldChild Old child.
	 * @returns Replaced node.
	 */
	public replaceChild(newChild: Node, oldChild: Node): Node {
		if (arguments.length < 2) {
			throw new this[PropertySymbol.window].TypeError(
				`Failed to execute 'replaceChild' on 'Node': 2 arguments required, but only ${arguments.length} present.`
			);
		}
		return this[PropertySymbol.replaceChild](newChild, oldChild);
	}

	/**
	 * Clones a node.
	 *
	 * @param [deep=false] "true" to clone deep.
	 * @returns Cloned node.
	 */
	public [PropertySymbol.cloneNode](deep = false): Node {
		const clone = NodeFactory.createNode(
			this[PropertySymbol.ownerDocument],
			<typeof Node>this.constructor
		);

		// Document has childNodes directly when it is created
		// We need to remove them
		if (clone[PropertySymbol.nodeArray].length) {
			const childNodes = clone[PropertySymbol.nodeArray];
			while (childNodes.length) {
				clone.removeChild(childNodes[0]);
			}
		}

		if (deep) {
			for (const childNode of this[PropertySymbol.nodeArray]) {
				const childClone = childNode.cloneNode(true);
				childClone[PropertySymbol.parentNode] = clone;
				clone[PropertySymbol.nodeArray].push(childClone);

				if (childClone[PropertySymbol.nodeType] === NodeTypeEnum.elementNode) {
					clone[PropertySymbol.elementArray].push(<Element>childClone);
				}
			}
		}

		return clone;
	}

	/**
	 * Append a child node to childNodes.
	 *
	 * @param  node Node to append.
	 * @param [disableValidations=false] "true" to disable validations.
	 * @returns Appended node.
	 */
	public [PropertySymbol.appendChild](node: Node, disableValidations = false): Node {
		if (node[PropertySymbol.proxy]) {
			node = node[PropertySymbol.proxy];
		}

		const self = this[PropertySymbol.proxy] || this;

		if (!disableValidations) {
			if (node === self) {
				throw new this[PropertySymbol.window].DOMException(
					"Failed to execute 'appendChild' on 'Node': Not possible to append a node as a child of itself."
				);
			}

			if (NodeUtility.isInclusiveAncestor(node, self, true)) {
				throw new this[PropertySymbol.window].DOMException(
					"Failed to execute 'appendChild' on 'Node': The new node is a parent of the node to insert to.",
					DOMExceptionNameEnum.domException
				);
			}
		}

		// If the type is DocumentFragment, then the child nodes of if it should be moved instead of the actual node.
		// See: https://developer.mozilla.org/en-US/docs/Web/API/DocumentFragment
		if (node[PropertySymbol.nodeType] === NodeTypeEnum.documentFragmentNode) {
			const childNodes = node[PropertySymbol.nodeArray];
			while (childNodes.length) {
				this.appendChild(childNodes[0]);
			}
			return node;
		}

		// Remove the node from its previous parent if it has any.
		if (node[PropertySymbol.parentNode]) {
			node[PropertySymbol.parentNode][PropertySymbol.removeChild](node);
		}

		node[PropertySymbol.parentNode] = self;

		node[PropertySymbol.clearCache]();

		this[PropertySymbol.nodeArray].push(node);

		if (node[PropertySymbol.nodeType] === NodeTypeEnum.elementNode) {
			this[PropertySymbol.elementArray].push(<Element>node);
		}

		node[PropertySymbol.connectedToNode]();

		// Mutation listeners
		for (const mutationListener of this[PropertySymbol.mutationListeners]) {
			if (mutationListener.options?.subtree && mutationListener.callback.deref()) {
				(<Node>node)[PropertySymbol.observeMutations](mutationListener);
			}
		}

		this[PropertySymbol.reportMutation](
			new MutationRecord({
				target: self,
				type: MutationTypeEnum.childList,
				addedNodes: [node]
			})
		);

		return node;
	}

	/**
	 * Remove Child element from childNodes array.
	 *
	 * @param node Node to remove.
	 * @returns Removed node.
	 */
	public [PropertySymbol.removeChild](node: Node): Node {
		if (node[PropertySymbol.proxy]) {
			node = node[PropertySymbol.proxy];
		}

		node[PropertySymbol.parentNode] = null;

		node[PropertySymbol.clearCache]();

		const index = this[PropertySymbol.nodeArray].indexOf(node);

		if (index === -1) {
			throw new this[PropertySymbol.window].DOMException(
				`Failed to execute 'removeChild' on 'Node': The node to be removed is not a child of this node.`
			);
		}

		this[PropertySymbol.nodeArray].splice(index, 1);

		if (node[PropertySymbol.nodeType] === NodeTypeEnum.elementNode) {
			const index = this[PropertySymbol.elementArray].indexOf(<Element>node);
			if (index !== -1) {
				this[PropertySymbol.elementArray].splice(index, 1);
			}
		}

		if (node[PropertySymbol.assignedToSlot]) {
			const index = node[PropertySymbol.assignedToSlot][PropertySymbol.assignedNodes].indexOf(node);
			if (index !== -1) {
				node[PropertySymbol.assignedToSlot][PropertySymbol.assignedNodes].splice(index, 1);
			}
			node[PropertySymbol.assignedToSlot] = null;
		}

		node[PropertySymbol.disconnectedFromNode]();

		// Mutation listeners
		for (const mutationListener of this[PropertySymbol.mutationListeners]) {
			if (mutationListener.options?.subtree && mutationListener.callback.deref()) {
				(<Node>node)[PropertySymbol.unobserveMutations](mutationListener);
			}
		}

		this[PropertySymbol.reportMutation](
			new MutationRecord({
				target: this[PropertySymbol.proxy] || this,
				type: MutationTypeEnum.childList,
				removedNodes: [node]
			})
		);

		return node;
	}

	/**
	 * Inserts a node before another.
	 *
	 * @param newNode Node to insert.
	 * @param referenceNode Node to insert before.
	 * @param [disableValidations=false] "true" to disable validations.
	 * @returns Inserted node.
	 */
	public [PropertySymbol.insertBefore](
		newNode: Node,
		referenceNode: Node | null,
		disableValidations = false
	): Node {
		if (newNode[PropertySymbol.proxy]) {
			newNode = newNode[PropertySymbol.proxy];
		}

		if (referenceNode && referenceNode[PropertySymbol.proxy]) {
			referenceNode = referenceNode[PropertySymbol.proxy];
		}

		if (newNode === referenceNode) {
			return newNode;
		}

		const self = this[PropertySymbol.proxy] || this;

		if (!disableValidations) {
			if (newNode === self) {
				throw new this[PropertySymbol.window].DOMException(
					"Failed to execute 'insertBefore' on 'Node': Not possible to insert a node as a child of itself."
				);
			}

			if (NodeUtility.isInclusiveAncestor(newNode, self, true)) {
				throw new this[PropertySymbol.window].DOMException(
					"Failed to execute 'insertBefore' on 'Node': The new node is a parent of the node to insert to.",
					DOMExceptionNameEnum.domException
				);
			}
		}

		// If the type is DocumentFragment, then the child nodes of if it should be moved instead of the actual node.
		// See: https://developer.mozilla.org/en-US/docs/Web/API/DocumentFragment
		if (newNode[PropertySymbol.nodeType] === NodeTypeEnum.documentFragmentNode) {
			const childNodes = (<Node>newNode)[PropertySymbol.nodeArray];
			while (childNodes.length > 0) {
				this[PropertySymbol.insertBefore](childNodes[0], referenceNode, true);
			}
			return newNode;
		}

		// If the referenceNode is null or undefined, then the newNode should be appended to the ancestorNode.
		// According to spec only null is valid, but browsers support undefined as well.
		if (!referenceNode) {
			this[PropertySymbol.appendChild](newNode, true);
			return newNode;
		}

		const nodeArray = this[PropertySymbol.nodeArray];

		// We need to check if the referenceNode is a child of this node before removing it from its parent, as the parent may be the same node and the index would be wrong.
		if (!nodeArray.includes(referenceNode)) {
			throw new this[PropertySymbol.window].DOMException(
				"Failed to execute 'insertBefore' on 'Node': The node before which the new node is to be inserted is not a child of this node."
			);
		}

		if (newNode[PropertySymbol.parentNode]) {
			newNode[PropertySymbol.parentNode][PropertySymbol.removeChild](newNode);
		}

		newNode[PropertySymbol.parentNode] = self;

		newNode[PropertySymbol.clearCache]();

		const index = nodeArray.indexOf(referenceNode);

		if (newNode[PropertySymbol.nodeType] === NodeTypeEnum.elementNode) {
			const elementArray = this[PropertySymbol.elementArray];
			if (referenceNode[PropertySymbol.nodeType] === NodeTypeEnum.elementNode) {
				elementArray.splice(elementArray.indexOf(<Element>referenceNode), 0, <Element>newNode);
			} else {
				let isInserted = false;
				for (let i = index, max = nodeArray.length; i < max; i++) {
					if (nodeArray[i][PropertySymbol.nodeType] === NodeTypeEnum.elementNode) {
						elementArray.splice(elementArray.indexOf(<Element>nodeArray[i]), 0, <Element>newNode);
						isInserted = true;
						break;
					}
				}
				if (!isInserted) {
					elementArray.push(<Element>newNode);
				}
			}
		}

		nodeArray.splice(index, 0, newNode);

		newNode[PropertySymbol.connectedToNode]();

		// Mutation listeners
		for (const mutationListener of this[PropertySymbol.mutationListeners]) {
			if (mutationListener.options?.subtree && mutationListener.callback.deref()) {
				newNode[PropertySymbol.observeMutations](mutationListener);
			}
		}

		this[PropertySymbol.reportMutation](
			new MutationRecord({
				target: self,
				type: MutationTypeEnum.childList,
				addedNodes: [newNode]
			})
		);

		return newNode;
	}

	/**
	 * Replaces a node with another.
	 *
	 * @param newChild New child.
	 * @param oldChild Old child.
	 * @returns Replaced node.
	 */
	public [PropertySymbol.replaceChild](newChild: Node, oldChild: Node): Node {
		this.insertBefore(newChild, oldChild);
		this.removeChild(oldChild);

		return oldChild;
	}

	/**
	 * Compares two nodes.
	 * Two nodes are equal if they have the same type, defining the same attributes, and so on.
	 *
	 * @param node  Node to compare.
	 * @returns boolean - `true` if two nodes are equal.
	 */
	public isEqualNode(node: Node): boolean {
		return NodeUtility.isEqualNode(this, node);
	}

	/**
	 * Converts the node to a string.
	 *
	 * @param listener Listener.
	 */
	public toString(): string {
		return `[object ${this.constructor.name}]`;
	}

	/**
	 * Observeres mutations on the node.
	 *
	 * Used by MutationObserver and internal logic.
	 *
	 * @param listener Listener.
	 */
	public [PropertySymbol.observeMutations](listener: IMutationListener): void {
		this[PropertySymbol.mutationListeners].push(listener);
		if (listener.options.subtree) {
			for (const node of this[PropertySymbol.nodeArray]) {
				(<Node>node)[PropertySymbol.observeMutations](listener);
			}
		}
	}

	/**
	 * Stops observing mutations on the node.
	 *
	 * Used by MutationObserver and internal logic.
	 *
	 * @param listener Listener.
	 */
	public [PropertySymbol.unobserveMutations](listener: IMutationListener): void {
		const index = this[PropertySymbol.mutationListeners].indexOf(listener);
		if (index !== -1) {
			this[PropertySymbol.mutationListeners].splice(index, 1);
		}
		if (listener.options.subtree) {
			for (const node of this[PropertySymbol.nodeArray]) {
				node[PropertySymbol.unobserveMutations](listener);
			}
		}
	}

	/**
	 * Reports a mutation on the node.
	 *
	 * Used by MutationObserver and internal logic.
	 *
	 * @param record Mutation record.
	 */
	public [PropertySymbol.reportMutation](record: MutationRecord): void {
		this[PropertySymbol.clearCache]();

		const mutationListeners = this[PropertySymbol.mutationListeners];

		if (!mutationListeners.length) {
			return;
		}

		for (let i = 0, max = mutationListeners.length; i < max; i++) {
			const mutationListener = mutationListeners[i];
			const callback = mutationListener.callback.deref();
			if (callback) {
				switch (record.type) {
					case MutationTypeEnum.childList:
						if (mutationListener.options.childList) {
							callback(record);
						}
						break;
					case MutationTypeEnum.attributes:
						if (
							mutationListener.options.attributes &&
							(!mutationListener.options.attributeFilter ||
								mutationListener.options.attributeFilter.includes(record.attributeName))
						) {
							callback(record);
						}
						break;
					case MutationTypeEnum.characterData:
						if (mutationListener.options?.characterData) {
							callback(record);
						}
						break;
				}
			} else {
				mutationListeners.splice(i, 1);
				i--;
				max--;
			}
		}
	}

	/**
	 * Clears query selector cache.
	 */
	public [PropertySymbol.clearCache](): void {
		const cache = this[PropertySymbol.cache];

		if (cache.querySelector.size) {
			for (const item of cache.querySelector.values()) {
				if (item.result) {
					item.result = null;
				}
			}
			cache.querySelector = new Map();
		}

		if (cache.querySelectorAll.size) {
			for (const item of cache.querySelectorAll.values()) {
				if (item.result) {
					item.result = null;
				}
			}
			cache.querySelectorAll = new Map();
		}

		if (cache.matches.size) {
			for (const item of cache.matches.values()) {
				if (item.result) {
					item.result = null;
				}
			}
			cache.matches = new Map();
		}

		if (cache.elementsByTagName.size) {
			for (const item of cache.elementsByTagName.values()) {
				if (item.result) {
					item.result = null;
				}
			}
			cache.elementsByTagName = new Map();
		}

		if (cache.elementsByTagNameNS.size) {
			for (const item of cache.elementsByTagNameNS.values()) {
				if (item.result) {
					item.result = null;
				}
			}
			cache.elementsByTagNameNS = new Map();
		}

		if (cache.elementByTagName.size) {
			for (const item of cache.elementByTagName.values()) {
				if (item.result) {
					item.result = null;
				}
			}
			cache.elementByTagName = new Map();
		}

		if (cache.elementById.size) {
			for (const item of cache.elementById.values()) {
				if (item.result) {
					item.result = null;
				}
			}
			cache.elementById = new Map();
		}

		const affectsCache = this[PropertySymbol.affectsCache];

		if (affectsCache.length) {
			for (const item of affectsCache) {
				item.result = null;
			}
			this[PropertySymbol.affectsCache] = [];
		}

		// Computed style cache is affected by all mutations.
		const document = this[PropertySymbol.ownerDocument];

		if (document && document[PropertySymbol.affectsComputedStyleCache].length) {
			for (const item of document[PropertySymbol.affectsComputedStyleCache]) {
				item.result = null;
			}
			document[PropertySymbol.affectsComputedStyleCache] = [];
		}
	}

	/**
	 * Called when connected to a node.
	 */
	public [PropertySymbol.connectedToNode](): void {
		const parentNode = this[PropertySymbol.parentNode];
		const parent = parentNode || this[PropertySymbol.host];
		let isConnected = false;
		let isDisconnected = false;

		if (!this[PropertySymbol.isConnected] && parent[PropertySymbol.isConnected]) {
			this[PropertySymbol.isConnected] = true;
			isConnected = true;
		} else if (this[PropertySymbol.isConnected] && !parent[PropertySymbol.isConnected]) {
			this[PropertySymbol.isConnected] = false;
			isDisconnected = true;
		}

		this[PropertySymbol.ownerDocument] = parent[PropertySymbol.ownerDocument] || <Document>parent;
		this[PropertySymbol.window] =
			parent[PropertySymbol.window] || parent[PropertySymbol.defaultView];

		if (parentNode) {
			if (this[PropertySymbol.nodeType] !== NodeTypeEnum.documentFragmentNode) {
				this[PropertySymbol.rootNode] = parentNode[PropertySymbol.rootNode];
			}

			if (parentNode[PropertySymbol.styleNode] && this[PropertySymbol.tagName] !== 'STYLE') {
				this[PropertySymbol.styleNode] = parentNode[PropertySymbol.styleNode];
			}

			if (parentNode[PropertySymbol.textAreaNode] && this[PropertySymbol.tagName] !== 'TEXTAREA') {
				this[PropertySymbol.textAreaNode] = parentNode[PropertySymbol.textAreaNode];
			}

			if (parentNode[PropertySymbol.formNode] && this[PropertySymbol.tagName] !== 'FORM') {
				this[PropertySymbol.formNode] = parentNode[PropertySymbol.formNode];
			}

			if (parentNode[PropertySymbol.selectNode] && this[PropertySymbol.tagName] !== 'SELECT') {
				this[PropertySymbol.selectNode] = parentNode[PropertySymbol.selectNode];
			}
		}

		const childNodes = this[PropertySymbol.nodeArray];
		for (let i = 0, max = childNodes.length; i < max; i++) {
			childNodes[i][PropertySymbol.parentNode] = this;
			childNodes[i][PropertySymbol.connectedToNode]();
		}

		// eslint-disable-next-line
		if ((<any>this)[PropertySymbol.shadowRoot]) {
			// eslint-disable-next-line
			(<any>this)[PropertySymbol.shadowRoot][PropertySymbol.connectedToNode]();
		}

		if (isConnected) {
			this[PropertySymbol.connectedToDocument]();
		} else if (isDisconnected) {
			this[PropertySymbol.disconnectedFromDocument]();
		}
	}

	/**
	 * Called when disconnected from a node.
	 */
	public [PropertySymbol.disconnectedFromNode](): void {
		if (this[PropertySymbol.nodeType] !== NodeTypeEnum.documentFragmentNode) {
			this[PropertySymbol.rootNode] = null;
		}

		if (this[PropertySymbol.tagName] !== 'STYLE') {
			this[PropertySymbol.styleNode] = null;
		}

		if (this[PropertySymbol.tagName] !== 'TEXTAREA') {
			this[PropertySymbol.textAreaNode] = null;
		}

		if (this[PropertySymbol.tagName] !== 'FORM') {
			this[PropertySymbol.formNode] = null;
		}

		if (this[PropertySymbol.tagName] !== 'SELECT') {
			this[PropertySymbol.selectNode] = null;
		}

		if (this[PropertySymbol.isConnected]) {
			this[PropertySymbol.isConnected] = false;
			this[PropertySymbol.disconnectedFromDocument]();
		}

		const childNodes = this[PropertySymbol.nodeArray];
		for (let i = 0, max = childNodes.length; i < max; i++) {
			childNodes[i][PropertySymbol.connectedToNode]();
		}

		// eslint-disable-next-line
		if ((<any>this)[PropertySymbol.shadowRoot]) {
			// eslint-disable-next-line
			(<any>this)[PropertySymbol.shadowRoot][PropertySymbol.disconnectedFromNode]();
		}
	}

	/**
	 * Called when connected to document.
	 */
	public [PropertySymbol.connectedToDocument](): void {
		// eslint-disable-next-line
		if ((<any>this)[PropertySymbol.shadowRoot]) {
			// eslint-disable-next-line
			(<any>this)[PropertySymbol.shadowRoot][PropertySymbol.connectedToDocument]();
		}
	}

	/**
	 * Called when disconnected from document.
	 */
	public [PropertySymbol.disconnectedFromDocument](): void {
		if (this[PropertySymbol.ownerDocument][PropertySymbol.activeElement] === <unknown>this) {
			this[PropertySymbol.ownerDocument][PropertySymbol.clearCache]();
			this[PropertySymbol.ownerDocument][PropertySymbol.activeElement] = null;
		}

		// eslint-disable-next-line
		if ((<any>this)[PropertySymbol.shadowRoot]) {
			// eslint-disable-next-line
			(<any>this)[PropertySymbol.shadowRoot][PropertySymbol.disconnectedFromDocument]();
		}
	}

	/**
	 * Reports the position of its argument node relative to the node on which it is called.
	 *
	 * @see https://dom.spec.whatwg.org/#dom-node-comparedocumentposition
	 * @param otherNode Other node.
	 */
	public compareDocumentPosition(otherNode: Node): number {
		/**
		 * 1. If this is other, then return zero.
		 */
		if (this === otherNode) {
			return 0;
		}

		/**
		 * 2. Let node1 be other and node2 be this.
		 */
		let node1: Node = otherNode;
		let node2: Node = this[PropertySymbol.proxy] || this;

		/**
		 * 3. Let attr1 and attr2 be null.
		 */
		let attr1 = null;
		let attr2 = null;

		/**
		 * 4. If node1 is an attribute, then set attr1 to node1 and node1 to attr1’s element.
		 */
		if (node1[PropertySymbol.nodeType] === NodeTypeEnum.attributeNode) {
			attr1 = node1;
			node1 = (<Attr>attr1)[PropertySymbol.ownerElement];
		}

		/**
		 * 5. If node2 is an attribute, then:
		 * 5.1. Set attr2 to node2 and node2 to attr2’s element.
		 */
		if (node2[PropertySymbol.nodeType] === NodeTypeEnum.attributeNode) {
			attr2 = node2;
			node2 = (<Attr>attr2)[PropertySymbol.ownerElement];

			/**
			 * 5.2. If attr1 and node1 are non-null, and node2 is node1, then:
			 */
			if (attr1 !== null && node1 !== null && node2 === node1) {
				/**
				 * 5.2.1. For each attr in node2’s attribute list:
				 */
				for (const attributes of Array.from(
					(<Element>node2)[PropertySymbol.attributes][PropertySymbol.namedItems].values()
				)) {
					/**
					 * 5.2.1.1. If attr equals attr1, then return the result of adding DOCUMENT_POSITION_IMPLEMENTATION_SPECIFIC and DOCUMENT_POSITION_PRECEDING.
					 */
					if (NodeUtility.isEqualNode(attributes[0], attr1)) {
						return (
							Node.DOCUMENT_POSITION_IMPLEMENTATION_SPECIFIC | Node.DOCUMENT_POSITION_PRECEDING
						);
					}

					/**
					 * 5.2.1.2. If attr equals attr2, then return the result of adding DOCUMENT_POSITION_IMPLEMENTATION_SPECIFIC and DOCUMENT_POSITION_FOLLOWING.
					 */
					if (NodeUtility.isEqualNode(attributes[0], attr2)) {
						return (
							Node.DOCUMENT_POSITION_IMPLEMENTATION_SPECIFIC | Node.DOCUMENT_POSITION_FOLLOWING
						);
					}
				}
			}
		}

		const node2Ancestors: Node[] = [];
		let node2Ancestor: Node = node2;

		while (node2Ancestor) {
			/**
			 * 7. If node1 is an ancestor of node2 […] then return the result of adding DOCUMENT_POSITION_CONTAINS to DOCUMENT_POSITION_PRECEDING.
			 */
			if (node2Ancestor === node1) {
				return Node.DOCUMENT_POSITION_CONTAINS | Node.DOCUMENT_POSITION_PRECEDING;
			}

			node2Ancestors.push(node2Ancestor);
			node2Ancestor = node2Ancestor[PropertySymbol.parentNode];
		}

		const node1Ancestors: Node[] = [];
		let node1Ancestor: Node = node1;

		while (node1Ancestor) {
			/**
			 * 8. If node1 is a descendant of node2 […] then return the result of adding DOCUMENT_POSITION_CONTAINED_BY to DOCUMENT_POSITION_FOLLOWING.
			 */
			if (node1Ancestor === node2) {
				return Node.DOCUMENT_POSITION_CONTAINED_BY | Node.DOCUMENT_POSITION_FOLLOWING;
			}

			node1Ancestors.push(node1Ancestor);
			node1Ancestor = node1Ancestor[PropertySymbol.parentNode];
		}

		const reverseArrayIndex = (array: Node[], reverseIndex: number): Node => {
			return array[array.length - 1 - reverseIndex];
		};

		const root = reverseArrayIndex(node2Ancestors, 0);

		/**
		 * 6. If node1 or node2 is null, or node1’s root is not node2’s root, then return the result of adding
		 * DOCUMENT_POSITION_DISCONNECTED, DOCUMENT_POSITION_IMPLEMENTATION_SPECIFIC, and either
		 * DOCUMENT_POSITION_PRECEDING or DOCUMENT_POSITION_FOLLOWING, with the constraint that this is to be consistent, together.
		 */
		if (!root || root !== reverseArrayIndex(node1Ancestors, 0)) {
			return (
				Node.DOCUMENT_POSITION_DISCONNECTED |
				Node.DOCUMENT_POSITION_IMPLEMENTATION_SPECIFIC |
				Node.DOCUMENT_POSITION_FOLLOWING
			);
		}

		// Find the lowest common ancestor
		let commonAncestorIndex = 0;
		const ancestorsMinLength = Math.min(node2Ancestors.length, node1Ancestors.length);

		for (let i = 0; i < ancestorsMinLength; ++i) {
			const node2Ancestor = reverseArrayIndex(node2Ancestors, i);
			const node1Ancestor = reverseArrayIndex(node1Ancestors, i);

			if (node2Ancestor !== node1Ancestor) {
				break;
			}

			commonAncestorIndex = i;
		}

		const commonAncestor = reverseArrayIndex(node2Ancestors, commonAncestorIndex);

		// Indexes within the common ancestor
		let indexes = 0;
		let node2Index = -1;
		let node1Index = -1;
		const node2Node = reverseArrayIndex(node2Ancestors, commonAncestorIndex + 1);
		const node1Node = reverseArrayIndex(node1Ancestors, commonAncestorIndex + 1);

		const computeNodeIndexes = (nodes: Node[]): void => {
			for (const childNode of nodes) {
				computeNodeIndexes(childNode[PropertySymbol.nodeArray]);

				if (childNode === node2Node) {
					node2Index = indexes;
				} else if (childNode === node1Node) {
					node1Index = indexes;
				}

				if (node2Index !== -1 && node1Index !== -1) {
					break;
				}

				indexes++;
			}
		};

		computeNodeIndexes(commonAncestor[PropertySymbol.nodeArray]);

		/**
		 * 9. If node1 is preceding node2, then return DOCUMENT_POSITION_PRECEDING.
		 * 10. Return DOCUMENT_POSITION_FOLLOWING.
		 */
		return node1Index < node2Index
			? Node.DOCUMENT_POSITION_PRECEDING
			: Node.DOCUMENT_POSITION_FOLLOWING;
	}

	/**
	 * Normalizes the sub-tree of the node, i.e. joins adjacent text nodes, and
	 * removes all empty text nodes.
	 *
	 * @see https://developer.mozilla.org/en-US/docs/Web/API/Node/normalize
	 */
	public normalize(): void {
		let child = this.firstChild;
		while (child) {
			if (NodeUtility.isTextNode(child)) {
				// Append text of all following text nodes, and remove them.
				while (NodeUtility.isTextNode(child.nextSibling)) {
					child.data += child.nextSibling.data;
					child.nextSibling.remove();
				}
				// Remove text node if it is still empty.
				if (!child.data.length) {
					const node = child;
					child = child.nextSibling;
					node.remove();
					continue;
				}
			} else {
				// Normalize child nodes recursively.
				child.normalize();
			}
			child = child.nextSibling;
		}
	}

	/**
	 * Determines whether the given node is equal to the current node.
	 *
	 * @see https://developer.mozilla.org/en-US/docs/Web/API/Node/isSameNode
	 * @param node Node to check.
	 * @returns True if the given node is equal to the current node, otherwise false.
	 */
	public isSameNode(node: Node): boolean {
		return this === node;
	}
}

// According to the spec, these properties should be on the prototype.
(<NodeTypeEnum>Node.prototype.ELEMENT_NODE) = NodeTypeEnum.elementNode;
(<NodeTypeEnum>Node.prototype.ATTRIBUTE_NODE) = NodeTypeEnum.attributeNode;

(<NodeTypeEnum>Node.prototype.TEXT_NODE) = NodeTypeEnum.textNode;
(<NodeTypeEnum>Node.prototype.CDATA_SECTION_NODE) = NodeTypeEnum.cdataSectionNode;
(<NodeTypeEnum>Node.prototype.COMMENT_NODE) = NodeTypeEnum.commentNode;
(<NodeTypeEnum>Node.prototype.DOCUMENT_NODE) = NodeTypeEnum.documentNode;
(<NodeTypeEnum>Node.prototype.DOCUMENT_TYPE_NODE) = NodeTypeEnum.documentTypeNode;
(<NodeTypeEnum>Node.prototype.DOCUMENT_FRAGMENT_NODE) = NodeTypeEnum.documentFragmentNode;
(<NodeTypeEnum>Node.prototype.PROCESSING_INSTRUCTION_NODE) = NodeTypeEnum.processingInstructionNode;
(<NodeDocumentPositionEnum>Node.prototype.DOCUMENT_POSITION_CONTAINED_BY) =
	NodeDocumentPositionEnum.containedBy;
(<NodeDocumentPositionEnum>Node.prototype.DOCUMENT_POSITION_CONTAINS) =
	NodeDocumentPositionEnum.contains;
(<NodeDocumentPositionEnum>Node.prototype.DOCUMENT_POSITION_DISCONNECTED) =
	NodeDocumentPositionEnum.disconnect;
(<NodeDocumentPositionEnum>Node.prototype.DOCUMENT_POSITION_FOLLOWING) =
	NodeDocumentPositionEnum.following;
(<NodeDocumentPositionEnum>Node.prototype.DOCUMENT_POSITION_IMPLEMENTATION_SPECIFIC) =
	NodeDocumentPositionEnum.implementationSpecific;
(<NodeDocumentPositionEnum>Node.prototype.DOCUMENT_POSITION_PRECEDING) =
	NodeDocumentPositionEnum.preceding;
