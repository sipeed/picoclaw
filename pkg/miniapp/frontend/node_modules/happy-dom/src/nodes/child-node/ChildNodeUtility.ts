import DOMException from '../../exception/DOMException.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import Node from '../node/Node.js';
import IParentNode from '../parent-node/IParentNode.js';
import IChildNode from './IChildNode.js';

/**
 * Child node utility.
 */
export default class ChildNodeUtility {
	/**
	 * Removes the node from its parent.
	 *
	 * @param childNode Child node.
	 */
	public static remove(childNode: IChildNode): void {
		if (childNode[PropertySymbol.parentNode]) {
			childNode[PropertySymbol.parentNode].removeChild(childNode);
		}
	}

	/**
	 * The Node.replaceWith() method replaces this Node in the children list of its parent with a set of Node or DOMString objects.
	 *
	 * @param childNode Child node.
	 * @param nodes List of Node or DOMString.
	 */
	public static replaceWith(childNode: IChildNode, ...nodes: (Node | string)[]): void {
		const parent = <IParentNode>childNode[PropertySymbol.parentNode];

		if (!parent) {
			throw new DOMException('This element has no parent node.');
		}

		for (const node of nodes) {
			if (node instanceof Node) {
				parent.insertBefore(node, childNode);
			} else {
				parent.insertBefore(
					parent[PropertySymbol.ownerDocument].createTextNode(String(node)),
					childNode
				);
			}
		}

		parent.removeChild(childNode);
	}

	/**
	 * Inserts a set of Node or DOMString objects in the children list of this ChildNode's parent, just before this ChildNode. DOMString objects are inserted as equivalent Text nodes.
	 *
	 * @param childNode Child node.
	 * @param nodes List of Node or DOMString.
	 */
	public static before(childNode: IChildNode, ...nodes: (string | Node)[]): void {
		const parent = <IParentNode>childNode[PropertySymbol.parentNode];

		if (!parent) {
			return;
		}

		for (const node of nodes) {
			if (node instanceof Node) {
				parent.insertBefore(node, childNode);
			} else {
				parent.insertBefore(
					parent[PropertySymbol.ownerDocument].createTextNode(String(node)),
					childNode
				);
			}
		}
	}

	/**
	 * Inserts a set of Node or DOMString objects in the children list of this ChildNode's parent, just after this ChildNode. DOMString objects are inserted as equivalent Text nodes.
	 *
	 * @param childNode Child node.
	 * @param nodes List of Node or DOMString.
	 */
	public static after(childNode: IChildNode, ...nodes: (string | Node)[]): void {
		const parent = <IParentNode>childNode[PropertySymbol.parentNode];

		if (!parent) {
			return;
		}

		const nextSibling = childNode.nextSibling;

		for (const node of nodes) {
			const insertedNode =
				node instanceof Node
					? node
					: parent[PropertySymbol.ownerDocument].createTextNode(String(node));
			if (!nextSibling) {
				parent.appendChild(insertedNode);
			} else {
				parent.insertBefore(insertedNode, nextSibling);
			}
		}
	}
}
