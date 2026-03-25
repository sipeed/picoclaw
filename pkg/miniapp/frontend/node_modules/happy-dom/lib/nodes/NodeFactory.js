import * as PropertySymbol from '../PropertySymbol.js';
/**
 * Node factory used for setting the owner document to nodes.
 */
export default class NodeFactory {
    static ownerDocuments = [];
    /**
     * Creates a node instance with the given owner document.
     *
     * @param ownerDocument Owner document.
     * @param nodeClass Node class.
     * @param [args] Node arguments.
     * @returns Node instance.
     */
    static createNode(ownerDocument, nodeClass, ...args) {
        if (!nodeClass.prototype[PropertySymbol.window]) {
            this.ownerDocuments.push(ownerDocument);
        }
        return new nodeClass(...args);
    }
    /**
     * Pulls an owner document from the queue.
     *
     * @returns Document.
     */
    static pullOwnerDocument() {
        return this.ownerDocuments.pop();
    }
}
//# sourceMappingURL=NodeFactory.js.map