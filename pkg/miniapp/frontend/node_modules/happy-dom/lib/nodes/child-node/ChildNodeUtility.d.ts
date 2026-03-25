import Node from '../node/Node.js';
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
    static remove(childNode: IChildNode): void;
    /**
     * The Node.replaceWith() method replaces this Node in the children list of its parent with a set of Node or DOMString objects.
     *
     * @param childNode Child node.
     * @param nodes List of Node or DOMString.
     */
    static replaceWith(childNode: IChildNode, ...nodes: (Node | string)[]): void;
    /**
     * Inserts a set of Node or DOMString objects in the children list of this ChildNode's parent, just before this ChildNode. DOMString objects are inserted as equivalent Text nodes.
     *
     * @param childNode Child node.
     * @param nodes List of Node or DOMString.
     */
    static before(childNode: IChildNode, ...nodes: (string | Node)[]): void;
    /**
     * Inserts a set of Node or DOMString objects in the children list of this ChildNode's parent, just after this ChildNode. DOMString objects are inserted as equivalent Text nodes.
     *
     * @param childNode Child node.
     * @param nodes List of Node or DOMString.
     */
    static after(childNode: IChildNode, ...nodes: (string | Node)[]): void;
}
//# sourceMappingURL=ChildNodeUtility.d.ts.map