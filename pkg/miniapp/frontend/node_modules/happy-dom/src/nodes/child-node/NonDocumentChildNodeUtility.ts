import Element from '../element/Element.js';
import NodeTypeEnum from '../node/NodeTypeEnum.js';
import INonDocumentTypeChildNode from './INonDocumentTypeChildNode.js';
import * as PropertySymbol from '../../PropertySymbol.js';

/**
 * Non Document Child node utility.
 */
export default class NonDocumentChildNodeUtility {
	/**
	 * Previous element sibling.
	 *
	 * @param childNode Child node.
	 * @returns Element.
	 */
	public static previousElementSibling(childNode: INonDocumentTypeChildNode): Element {
		let sibling = childNode.previousSibling;
		while (sibling && sibling[PropertySymbol.nodeType] !== NodeTypeEnum.elementNode) {
			sibling = sibling.previousSibling;
		}
		return <Element>sibling;
	}

	/**
	 * Next element sibling.
	 *
	 * @param childNode Child node.
	 * @returns Element.
	 */
	public static nextElementSibling(childNode: INonDocumentTypeChildNode): Element {
		let sibling = childNode.nextSibling;
		while (sibling && sibling[PropertySymbol.nodeType] !== NodeTypeEnum.elementNode) {
			sibling = sibling.nextSibling;
		}
		return <Element>sibling;
	}
}
