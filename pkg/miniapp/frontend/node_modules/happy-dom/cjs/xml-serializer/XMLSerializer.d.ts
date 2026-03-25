import Node from '../nodes/node/Node.cjs';
/**
 * Serializes a node into XML.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/XMLSerializer
 */
export default class XMLSerializer {
    #private;
    /**
     * Serializes a node into XML.
     *
     * @param root Root node.
     * @returns Result.
     */
    serializeToString(root: Node): string;
}
//# sourceMappingURL=XMLSerializer.d.ts.map