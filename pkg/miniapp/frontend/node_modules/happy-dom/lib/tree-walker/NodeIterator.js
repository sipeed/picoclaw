import TreeWalker from './TreeWalker.js';
import * as PropertySymbol from '../PropertySymbol.js';
import NodeFilter from './NodeFilter.js';
/**
 * The NodeIterator object represents the nodes of a document subtree and a position within them.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/NodeIterator
 */
export default class NodeIterator {
    #root = null;
    #whatToShow = -1;
    #filter = null;
    #walker;
    #atRoot = true;
    /**
     * Constructor.
     *
     * @param root Root.
     * @param [whatToShow] What to show.
     * @param [filter] Filter.
     */
    constructor(root, whatToShow = -1, filter = null) {
        this.#root = root;
        this.#whatToShow = whatToShow;
        this.#filter = filter;
        this.#walker = new TreeWalker(root, whatToShow, filter);
    }
    /**
     * Returns root.
     *
     * @returns Root.
     */
    get root() {
        return this.#root;
    }
    /**
     * Returns what to show.
     *
     * @returns What to show.
     */
    get whatToShow() {
        return this.#whatToShow;
    }
    /**
     * Returns filter.
     *
     * @returns Filter.
     */
    get filter() {
        return this.#filter;
    }
    /**
     * Moves the current Node to the next visible node in the document order.
     *
     * @returns Current node.
     */
    nextNode() {
        if (this.#atRoot) {
            this.#atRoot = false;
            if (this.#walker[PropertySymbol.filterNode](this.#root) !== NodeFilter.FILTER_ACCEPT) {
                return this.#walker.nextNode();
            }
            return this.#root;
        }
        return this.#walker.nextNode();
    }
    /**
     * Moves the current Node to the previous visible node in the document order, and returns the found node. It also moves the current node to this one. If no such node exists, or if it is before that the root node defined at the object construction, returns null and the current node is not changed.
     *
     * @returns Current node.
     */
    previousNode() {
        return this.#walker.previousNode();
    }
}
//# sourceMappingURL=NodeIterator.js.map