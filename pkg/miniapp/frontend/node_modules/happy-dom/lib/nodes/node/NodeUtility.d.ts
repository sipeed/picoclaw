import Text from '../text/Text.js';
import Node from './Node.js';
import Element from '../element/Element.js';
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
    static isTextNode(node: Node | null): node is Text;
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
    static isInclusiveAncestor(ancestorNode: Node | null, referenceNode: Node | null, includeShadowRoots?: boolean): boolean;
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
    static isFollowing(nodeA: Node, nodeB: Node): boolean;
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
    static getNodeLength(node: Node): number;
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
    static following(node: Node, root?: Node): Node;
    /**
     * Returns the next sibling or parents sibling.
     *
     * @param node Node.
     * @returns Next descendant node.
     */
    static nextDescendantNode(node: Node): Node;
    /**
     * Needed by https://dom.spec.whatwg.org/#concept-node-equals
     *
     * @param elementA
     * @param elementB
     */
    static attributeListsEqual(elementA: Element, elementB: Element): boolean;
    /**
     * Check if node nodeA equals node nodeB.
     * Reference: https://dom.spec.whatwg.org/#concept-node-equals
     *
     * @param nodeA Node A.
     * @param nodeB Node B.
     */
    static isEqualNode(nodeA: Node, nodeB: Node): boolean;
}
//# sourceMappingURL=NodeUtility.d.ts.map