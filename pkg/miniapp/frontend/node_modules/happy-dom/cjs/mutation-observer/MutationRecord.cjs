"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
/**
 * MutationRecord is a model for a mutation.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/MutationRecord
 */
class MutationRecord {
    type = null;
    target = null;
    addedNodes = [];
    removedNodes = [];
    previousSibling = null;
    nextSibling = null;
    attributeName = null;
    attributeNamespace = null;
    oldValue = null;
    /**
     * Constructor.
     *
     * @param init Options to initialize the mutation record.
     */
    constructor(init) {
        Object.assign(this, init);
    }
}
exports.default = MutationRecord;
//# sourceMappingURL=MutationRecord.cjs.map