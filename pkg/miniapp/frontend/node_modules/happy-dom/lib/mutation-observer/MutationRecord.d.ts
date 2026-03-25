import Node from '../nodes/node/Node.js';
/**
 * MutationRecord is a model for a mutation.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/MutationRecord
 */
export default class MutationRecord {
    type: string;
    target: Node;
    addedNodes: Node[];
    removedNodes: Node[];
    previousSibling: Node;
    nextSibling: Node;
    attributeName: string;
    attributeNamespace: string;
    oldValue: string;
    /**
     * Constructor.
     *
     * @param init Options to initialize the mutation record.
     */
    constructor(init?: Partial<MutationRecord>);
}
//# sourceMappingURL=MutationRecord.d.ts.map