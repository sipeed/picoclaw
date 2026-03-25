import Document from '../nodes/document/Document.js';
import Node from './node/Node.js';
/**
 * Node factory used for setting the owner document to nodes.
 */
export default class NodeFactory {
    static ownerDocuments: Document[];
    /**
     * Creates a node instance with the given owner document.
     *
     * @param ownerDocument Owner document.
     * @param nodeClass Node class.
     * @param [args] Node arguments.
     * @returns Node instance.
     */
    static createNode<T extends Node>(ownerDocument: Document, nodeClass: new (...args: any[]) => T, ...args: any[]): T;
    /**
     * Pulls an owner document from the queue.
     *
     * @returns Document.
     */
    static pullOwnerDocument(): Document;
}
//# sourceMappingURL=NodeFactory.d.ts.map