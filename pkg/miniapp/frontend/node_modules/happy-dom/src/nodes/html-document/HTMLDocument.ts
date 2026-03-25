import Document from '../document/Document.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import Node from '../node/Node.js';
import NodeTypeEnum from '../node/NodeTypeEnum.js';

/**
 * Document.
 */
export default class HTMLDocument extends Document {
	/**
	 * Constructor.
	 */
	constructor() {
		super();

		// Default document elements
		const documentElement = this.createElement('html');
		const bodyElement = this.createElement('body');
		const headElement = this.createElement('head');

		this.appendChild(documentElement);

		documentElement.appendChild(headElement);
		documentElement.appendChild(bodyElement);
	}

	/**
	 * @override
	 */
	public override [PropertySymbol.appendChild](node: Node, disableValidations = false): Node {
		if (node[PropertySymbol.nodeType] === NodeTypeEnum.textNode) {
			throw new this[PropertySymbol.window].Error(
				`Failed to execute 'appendChild' on 'Node': Nodes of type '#text' may not be inserted inside nodes of type '#document'.`
			);
		}

		if (node[PropertySymbol.nodeType] === NodeTypeEnum.documentFragmentNode) {
			return;
		}

		if (
			node[PropertySymbol.nodeType] === NodeTypeEnum.elementNode &&
			this[PropertySymbol.elementArray].length !== 0
		) {
			throw new this[PropertySymbol.window].Error(
				`Failed to execute 'appendChild' on 'Node': Only one element on document allowed.`
			);
		}

		return super[PropertySymbol.appendChild](node, disableValidations);
	}

	/**
	 * @override
	 */
	public override [PropertySymbol.insertBefore](
		newNode: Node,
		referenceNode: Node | null,
		disableValidations = false
	): Node {
		if (newNode[PropertySymbol.nodeType] === NodeTypeEnum.textNode) {
			throw new this[PropertySymbol.window].Error(
				`Failed to execute 'insertBefore' on 'Node': Nodes of type '#text' may not be inserted inside nodes of type '#document'.`
			);
		}

		if (newNode[PropertySymbol.nodeType] === NodeTypeEnum.documentFragmentNode) {
			return;
		}

		if (
			newNode[PropertySymbol.nodeType] === NodeTypeEnum.elementNode &&
			this[PropertySymbol.elementArray].length !== 0
		) {
			throw new this[PropertySymbol.window].Error(
				`Failed to execute 'insertBefore' on 'Node': Only one element on document allowed.`
			);
		}

		return super[PropertySymbol.insertBefore](newNode, referenceNode, disableValidations);
	}
}
