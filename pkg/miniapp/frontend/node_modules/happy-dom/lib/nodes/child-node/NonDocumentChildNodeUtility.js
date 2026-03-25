import NodeTypeEnum from '../node/NodeTypeEnum.js';
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
    static previousElementSibling(childNode) {
        let sibling = childNode.previousSibling;
        while (sibling && sibling[PropertySymbol.nodeType] !== NodeTypeEnum.elementNode) {
            sibling = sibling.previousSibling;
        }
        return sibling;
    }
    /**
     * Next element sibling.
     *
     * @param childNode Child node.
     * @returns Element.
     */
    static nextElementSibling(childNode) {
        let sibling = childNode.nextSibling;
        while (sibling && sibling[PropertySymbol.nodeType] !== NodeTypeEnum.elementNode) {
            sibling = sibling.nextSibling;
        }
        return sibling;
    }
}
//# sourceMappingURL=NonDocumentChildNodeUtility.js.map