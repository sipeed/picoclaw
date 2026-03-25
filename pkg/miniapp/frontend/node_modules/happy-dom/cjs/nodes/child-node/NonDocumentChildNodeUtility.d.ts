import Element from '../element/Element.cjs';
import INonDocumentTypeChildNode from './INonDocumentTypeChildNode.cjs';
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