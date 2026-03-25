import Element from '../element/Element.js';
import INonDocumentTypeChildNode from './INonDocumentTypeChildNode.js';
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
    static previousElementSibling(childNode: INonDocumentTypeChildNode): Element;
    /**
     * Next element sibling.
     *
     * @param childNode Child node.
     * @returns Element.
     */
    static nextElementSibling(childNode: INonDocumentTypeChildNode): Element;
}
//# sourceMappingURL=NonDocumentChildNodeUtility.d.ts.map