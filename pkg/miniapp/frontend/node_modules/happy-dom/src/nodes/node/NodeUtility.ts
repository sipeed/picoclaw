import Text from '../text/Text.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import Comment from '../comment/Comment.js';
import Node from './Node.js';
import NodeTypeEnum from './NodeTypeEnum.js';
import Element from '../element/Element.js';
import DocumentType from '../document-type/DocumentType.js';
import Attr from '../attr/Attr.js';
import ProcessingInstruction from '../processing-instruction/ProcessingInstruction.js';
import ShadowRoot from '../shadow-root/ShadowRoot.js';

/**
 * Node utility.
 */
export default class NodeUtility {
	/**
	 * Returns whether the passed node is a text node, and narrows its type.
	 *
	 * @param node The node to be tested.
	 * @returns "true" if the node is a text node.
	 */
	public static isTextNode(node: Node | null): node is Text {
		return node?.[PropertySymbol.nodeType] === NodeTypeEnum.textNode;
	}

	/**
	 * Returns boolean indicating if "ancestorNode" is an inclusive ancestor of "referenceNode".
	 *
	 * Based on:
	 * https://github.com/jsdom/jsdom/blob/master/lib/jsdom/living/helpers/node.js
	 *
	 * @see https://dom.spec.whatwg.org/#concept-tree-inclusive-ancestor
	 * @param ancestorNode Ancestor node.
	 * @param referenceNode Reference node.
	 * @param [includeShadowRoots = false] Include shadow roots.
	 * @returns "true" if inclusive ancestor.
	 */
	public static isInclusiveAncestor(
		ancestorNode: Node | null,
		referenceNode: Node | null,
		includeShadowRoots = false
	): boolean {
		if (ancestorNode === null || referenceNode === null) {
			return false;
		}

		if (ancestorNode === referenceNode) {
			return true;
		}

		if (!(<Node>ancestorNode)[PropertySymbol.nodeArray].length) {
			return false;
		}

		if (
			includeShadowRoots &&
			referenceNode[PropertySymbol.isConnected] !== ancestorNode[PropertySymbol.isConnected]
		) {
			return false;
		}

		if (
			includeShadowRoots &&
			ancestorNode === referenceNode[PropertySymbol.ownerDocument] &&
			referenceNode[PropertySymbol.isConnected]
		) {
			return true;
		}

		let parent: Node = referenceNode[PropertySymbol.parentNode];

		while (parent) {
			if (ancestorNode === parent) {
				return true;
			}

			parent = parent[PropertySymbol.parentNode]
				? parent[PropertySymbol.parentNode]
				: includeShadowRoots && (<ShadowRoot>parent).host
					? (<ShadowRoot>parent).host
					: null;
		}

		return false;
	}

	/**
	 * Returns boolean indicating if nodeB is following nodeA in the document tree.
	 *
	 * Based on:
	 * https://github.com/jsdom/jsdom/blob/master/lib/jsdom/living/helpers/node.js
	 *
	 * @see https://dom.spec.whatwg.org/#concept-tree-following
	 * @param nodeA Node A.
	 * @param nodeB Node B.
	 * @returns "true" if following.
	 */
	public static isFollowing(nodeA: Node, nodeB: Node): boolean {
		if (nodeA === nodeB) {
			return false;
		}

		let current: Node = nodeB;

		while (current) {
			current = this.following(current);

			if (current === nodeA) {
				return true;
			}
		}

		return false;
	}

	/**
	 * Node length.
	 *
	 * Based on:
	 * https://github.com/jsdom/jsdom/blob/master/lib/jsdom/living/helpers/node.js
	 *
	 * @see https://dom.spec.whatwg.org/#concept-node-length
	 * @param node Node.
	 * @returns Node length.
	 */
	public static getNodeLength(node: Node): number {
		switch (node[PropertySymbol.nodeType]) {
			case NodeTypeEnum.documentTypeNode:
				return 0;

			case NodeTypeEnum.textNode:
			case NodeTypeEnum.processingInstructionNode:
			case NodeTypeEnum.commentNode:
				return (<Text | Comment>node).data.length;

			default:
				return (<Node>node)[PropertySymbol.nodeArray].length;
		}
	}

	/**
	 * Returns boolean indicating if nodeB is following nodeA in the document tree.
	 *
	 * Based on:
	 * https://github.com/jsdom/js-symbol-tree/blob/master/lib/SymbolTree.js#L220
	 *
	 * @param node Node.
	 * @param [root] Root.
	 * @returns Following node.
	 */
	public static following(node: Node, root?: Node): Node {
		const firstChild = node.firstChild;

		if (firstChild) {
			return firstChild;
		}

		let current = node;

		while (current) {
			if (current === root) {
				return null;
			}

			const nextSibling = current.nextSibling;

			if (nextSibling) {
				return nextSibling;
			}

			current = current[PropertySymbol.parentNode];
		}

		return null;
	}

	/**
	 * Returns the next sibling or parents sibling.
	 *
	 * @param node Node.
	 * @returns Next descendant node.
	 */
	public static nextDescendantNode(node: Node): Node {
		while (node && !node.nextSibling) {
			node = node[PropertySymbol.parentNode];
		}

		if (!node) {
			return null;
		}

		return node.nextSibling;
	}

	/**
	 * Needed by https://dom.spec.whatwg.org/#concept-node-equals
	 *
	 * @param elementA
	 * @param elementB
	 */
	public static attributeListsEqual(elementA: Element, elementB: Element): boolean {
		const attributesA = Array.from(
			elementA[PropertySymbol.attributes][PropertySymbol.namedItems].values()
		);
		const attributesB = Array.from(
			elementB[PropertySymbol.attributes][PropertySymbol.namedItems].values()
		);
		for (const attributeA of attributesA) {
			let found = false;
			for (const attributeB of attributesB) {
				if (
					attributeA[0][PropertySymbol.namespaceURI] ===
						attributeB[0][PropertySymbol.namespaceURI] &&
					attributeA[0].localName === attributeB[0].localName &&
					attributeA[0][PropertySymbol.value] === attributeB[0][PropertySymbol.value]
				) {
					found = true;
					break;
				}
			}
			if (!found) {
				return false;
			}
		}
		return true;
	}

	/**
	 * Check if node nodeA equals node nodeB.
	 * Reference: https://dom.spec.whatwg.org/#concept-node-equals
	 *
	 * @param nodeA Node A.
	 * @param nodeB Node B.
	 */
	public static isEqualNode(nodeA: Node, nodeB: Node): boolean {
		if (nodeA[PropertySymbol.nodeType] !== nodeB[PropertySymbol.nodeType]) {
			return false;
		}

		switch (nodeA[PropertySymbol.nodeType]) {
			case NodeTypeEnum.documentTypeNode:
				const documentTypeA = <DocumentType>nodeA;
				const documentTypeB = <DocumentType>nodeB;

				if (
					documentTypeA.name !== documentTypeB.name ||
					documentTypeA.publicId !== documentTypeB.publicId ||
					documentTypeA.systemId !== documentTypeB.systemId
				) {
					return false;
				}
				break;
			case NodeTypeEnum.elementNode:
				const elementA = <Element>nodeA;
				const elementB = <Element>nodeB;

				if (
					elementA[PropertySymbol.namespaceURI] !== elementB[PropertySymbol.namespaceURI] ||
					elementA[PropertySymbol.prefix] !== elementB[PropertySymbol.prefix] ||
					elementA[PropertySymbol.localName] !== elementB[PropertySymbol.localName] ||
					elementA[PropertySymbol.attributes][PropertySymbol.namespaceItems].size !==
						elementB[PropertySymbol.attributes][PropertySymbol.namespaceItems].size
				) {
					return false;
				}
				break;
			case NodeTypeEnum.attributeNode:
				const attributeA = <Attr>nodeA;
				const attributeB = <Attr>nodeB;

				if (
					attributeA[PropertySymbol.namespaceURI] !== attributeB[PropertySymbol.namespaceURI] ||
					attributeA.localName !== attributeB.localName ||
					attributeA[PropertySymbol.value] !== attributeB[PropertySymbol.value]
				) {
					return false;
				}
				break;
			case NodeTypeEnum.processingInstructionNode:
				const processingInstructionA = <ProcessingInstruction>nodeA;
				const processingInstructionB = <ProcessingInstruction>nodeB;

				if (
					processingInstructionA.target !== processingInstructionB.target ||
					processingInstructionA.data !== processingInstructionB.data
				) {
					return false;
				}
				break;
			case NodeTypeEnum.textNode:
			case NodeTypeEnum.commentNode:
				type TextOrComment = Text | Comment;
				const textOrCommentA = <TextOrComment>nodeA;
				const textOrCommentB = <TextOrComment>nodeB;

				if (textOrCommentA.data !== textOrCommentB.data) {
					return false;
				}
				break;
		}

		if (
			nodeA[PropertySymbol.nodeType] === NodeTypeEnum.elementNode &&
			!NodeUtility.attributeListsEqual(<Element>nodeA, <Element>nodeB)
		) {
			return false;
		}

		if (
			(<Node>nodeA)[PropertySymbol.nodeArray].length !==
			(<Node>nodeB)[PropertySymbol.nodeArray].length
		) {
			return false;
		}

		for (let i = 0; i < (<Node>nodeA)[PropertySymbol.nodeArray].length; i++) {
			const childNodeA = (<Node>nodeA)[PropertySymbol.nodeArray][i];
			const childNodeB = (<Node>nodeB)[PropertySymbol.nodeArray][i];

			if (!NodeUtility.isEqualNode(childNodeA, childNodeB)) {
				return false;
			}
		}

		return true;
	}
}
