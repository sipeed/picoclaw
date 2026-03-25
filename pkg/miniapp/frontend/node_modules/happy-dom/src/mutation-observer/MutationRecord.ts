import Node from '../nodes/node/Node.js';

/**
 * MutationRecord is a model for a mutation.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/MutationRecord
 */
export default class MutationRecord {
	public type: string = null;
	public target: Node = null;
	public addedNodes: Node[] = [];
	public removedNodes: Node[] = [];
	public previousSibling: Node = null;
	public nextSibling: Node = null;
	public attributeName: string = null;
	public attributeNamespace: string = null;
	public oldValue: string = null;

	/**
	 * Constructor.
	 *
	 * @param init Options to initialize the mutation record.
	 */
	constructor(init?: Partial<MutationRecord>) {
		Object.assign(this, init);
	}
}
