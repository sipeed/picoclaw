import Node from '../node/Node.js';
export default interface IChildNode extends Node {
    /**
     * Removes the node from its parent.
     */
    remove(): void;
    /**
     * Inserts a set of Node or DOMString objects in the children list of this ChildNode's parent, just before this ChildNode. DOMString objects are inserted as equivalent Text nodes.
     *
     * @param nodes List of Node or DOMString.
     */
    before(...nodes: (Node | string)[]): void;
    /**
     * Inserts a set of Node or DOMString objects in the children list of this ChildNode's parent, just after this ChildNode. DOMString objects are inserted as equivalent Text nodes.
     *
     * @param nodes List of Node or DOMString.
     */
    after(...nodes: (Node | string)[]): void;
    /**
     * The Node.replaceWith() method replaces this Node in the children list of its parent with a set of Node or DOMString objects.
     *
     * @param nodes List of Node or DOMString.
     */
    replaceWith(...nodes: (Node | string)[]): void;
}
//# sourceMappingURL=IChildNode.d.ts.map