import Node from '../node/Node.cjs';
import * as PropertySymbol from '../../PropertySymbol.cjs';
import Element from '../element/Element.cjs';
import IChildNode from '../child-node/IChildNode.cjs';
import INonDocumentTypeChildNode from '../child-node/INonDocumentTypeChildNode.cjs';
/**
 * Character data base class.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/CharacterData.
 */
export default abstract class CharacterData extends Node implements IChildNode, INonDocumentTypeChildNode {
    [PropertySymbol.data]: string;
    cloneNode: (deep?: boolean) => CharacterData;
    /**
     * Constructor.
     *
     * @param [data] Data.
     */
    constructor(data?: string);
    /**
     * Returns text content.
     *
     * @returns Text content.
     */
    get length(): number;
    /**
     * Returns text content.
     *
     * @returns Text content.
     */
    get data(): string;
    /**
     * Sets text content.
     *
     * @param textContent Text content.
     */
    set data(data: string);
    /**
     * Returns text content.
     *
     * @returns Text content.
     */
    get textContent(): string;
    /**
     * Sets text content.
     *
     * @param textContent Text content.
     */
    set textContent(textContent: string);
    /**
     * Returns node value.
     *
     * @returns Node value.
     */
    get nodeValue(): string;
    /**
     * Sets node value.
     *
     * @param nodeValue Node value.
     */
    set nodeValue(nodeValue: string);
    /**
     * Previous element sibling.
     *
     * @returns Element.
     */
    get previousElementSibling(): Element;
    /**
     * Next element sibling.
     *
     * @returns Element.
     */
    get nextElementSibling(): Element;
    /**
     * Appends the given DOMString to the CharacterData.data string; when this method returns, data contains the concatenated DOMString.
     *
     * @param data Data.
     */
    appendData(data: string): void;
    /**
     * Removes the specified amount of characters, starting at the specified offset, from the CharacterData.data string; when this method returns, data contains the shortened DOMString.
     *
     * @param offset Offset.
     * @param count Count.
     */
    deleteData(offset: number, count: number): void;
    /**
     * Inserts the specified characters, at the specified offset, in the CharacterData.data string; when this method returns, data contains the modified DOMString.
     *
     * @param offset Offset.
     * @param data Data.
     */
    insertData(offset: number, data: string): void;
    /**
     * Replaces the specified amount of characters, starting at the specified offset, with the specified DOMString; when this method returns, data contains the modified DOMString.
     *
     * @param offset Offset.
     * @param count Count.
     * @param data Data.
     */
    replaceData(offset: number, count: number, data: string): void;
    /**
     * Returns a DOMString containing the part of CharacterData.data of the specified length and starting at the specified offset.
     *
     * @param offset Offset.
     * @param count Count.
     */
    substringData(offset: number, count: number): string;
    /**
     * Removes the object from its parent children list.
     */
    remove(): void;
    /**
     * The Node.replaceWith() method replaces this Node in the children list of its parent with a set of Node or DOMString objects.
     *
     * @param nodes List of Node or DOMString.
     */
    replaceWith(...nodes: (Node | string)[]): void;
    /**
     * Inserts a set of Node or DOMString objects in the children list of this ChildNode's parent, just before this ChildNode. DOMString objects are inserted as equivalent Text nodes.
     *
     * @param nodes List of Node or DOMString.
     */
    before(...nodes: (string | Node)[]): void;
    /**
     * Inserts a set of Node or DOMString objects in the children list of this ChildNode's parent, just after this ChildNode. DOMString objects are inserted as equivalent Text nodes.
     *
     * @param nodes List of Node or DOMString.
     */
    after(...nodes: (string | Node)[]): void;
    /**
     * @override
     */
    [PropertySymbol.cloneNode](deep?: boolean): CharacterData;
}
//# sourceMappingURL=CharacterData.d.ts.map