import INodeFilter from './INodeFilter.js';
import Node from '../nodes/node/Node.js';
/**
 * The NodeIterator object represents the nodes of a document subtree and a position within them.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/NodeIterator
 */
export default class NodeIterator {
    #private;
    /**
     * Constructor.
     *
     * @param root Root.
     * @param [whatToShow] What to show.
     * @param [filter] Filter.
     */
    constructor(root: Node, whatToShow?: number, filter?: INodeFilter);
    /**
     * Returns root.
     *
     * @returns Root.
     */
    get root(): Node | null;
    /**
     * Returns what to show.
     *
     * @returns What to show.
     */
    get whatToShow(): number;
    /**
     * Returns filter.
     *
     * @returns Filter.
     */
    get filter(): INodeFilter;
    /**
     * Moves the current Node to the next visible node in the document order.
     *
     * @returns Current node.
     */
    nextNode(): Node;
    /**
     * Moves the current Node to the previous visible node in the document order, and returns the found node. It also moves the current node to this one. If no such node exists, or if it is before that the root node defined at the object construction, returns null and the current node is not changed.
     *
     * @returns Current node.
     */
    previousNode(): Node;
}
//# sourceMappingURL=NodeIterator.d.ts.map