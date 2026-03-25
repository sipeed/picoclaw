import Node from '../node/Node.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import CharacterDataUtility from './CharacterDataUtility.js';
import NonDocumentChildNodeUtility from '../child-node/NonDocumentChildNodeUtility.js';
import ChildNodeUtility from '../child-node/ChildNodeUtility.js';
import MutationRecord from '../../mutation-observer/MutationRecord.js';
import MutationTypeEnum from '../../mutation-observer/MutationTypeEnum.js';
/**
 * Character data base class.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/CharacterData.
 */
export default class CharacterData extends Node {
    [PropertySymbol.data] = '';
    /**
     * Constructor.
     *
     * @param [data] Data.
     */
    constructor(data) {
        super();
        this[PropertySymbol.data] = data !== undefined ? String(data) : '';
    }
    /**
     * Returns text content.
     *
     * @returns Text content.
     */
    get length() {
        return this[PropertySymbol.data].length;
    }
    /**
     * Returns text content.
     *
     * @returns Text content.
     */
    get data() {
        return this[PropertySymbol.data];
    }
    /**
     * Sets text content.
     *
     * @param textContent Text content.
     */
    set data(data) {
        const oldValue = this[PropertySymbol.data];
        this[PropertySymbol.data] = String(data);
        this[PropertySymbol.reportMutation](new MutationRecord({
            target: this,
            type: MutationTypeEnum.characterData,
            oldValue
        }));
    }
    /**
     * Returns text content.
     *
     * @returns Text content.
     */
    get textContent() {
        return this[PropertySymbol.data];
    }
    /**
     * Sets text content.
     *
     * @param textContent Text content.
     */
    set textContent(textContent) {
        this.data = textContent;
    }
    /**
     * Returns node value.
     *
     * @returns Node value.
     */
    get nodeValue() {
        return this[PropertySymbol.data];
    }
    /**
     * Sets node value.
     *
     * @param nodeValue Node value.
     */
    set nodeValue(nodeValue) {
        this.textContent = nodeValue;
    }
    /**
     * Previous element sibling.
     *
     * @returns Element.
     */
    get previousElementSibling() {
        return NonDocumentChildNodeUtility.previousElementSibling(this);
    }
    /**
     * Next element sibling.
     *
     * @returns Element.
     */
    get nextElementSibling() {
        return NonDocumentChildNodeUtility.nextElementSibling(this);
    }
    /**
     * Appends the given DOMString to the CharacterData.data string; when this method returns, data contains the concatenated DOMString.
     *
     * @param data Data.
     */
    appendData(data) {
        CharacterDataUtility.appendData(this, data);
    }
    /**
     * Removes the specified amount of characters, starting at the specified offset, from the CharacterData.data string; when this method returns, data contains the shortened DOMString.
     *
     * @param offset Offset.
     * @param count Count.
     */
    deleteData(offset, count) {
        CharacterDataUtility.deleteData(this, offset, count);
    }
    /**
     * Inserts the specified characters, at the specified offset, in the CharacterData.data string; when this method returns, data contains the modified DOMString.
     *
     * @param offset Offset.
     * @param data Data.
     */
    insertData(offset, data) {
        CharacterDataUtility.insertData(this, offset, data);
    }
    /**
     * Replaces the specified amount of characters, starting at the specified offset, with the specified DOMString; when this method returns, data contains the modified DOMString.
     *
     * @param offset Offset.
     * @param count Count.
     * @param data Data.
     */
    replaceData(offset, count, data) {
        CharacterDataUtility.replaceData(this, offset, count, data);
    }
    /**
     * Returns a DOMString containing the part of CharacterData.data of the specified length and starting at the specified offset.
     *
     * @param offset Offset.
     * @param count Count.
     */
    substringData(offset, count) {
        return CharacterDataUtility.substringData(this, offset, count);
    }
    /**
     * Removes the object from its parent children list.
     */
    remove() {
        ChildNodeUtility.remove(this);
    }
    /**
     * The Node.replaceWith() method replaces this Node in the children list of its parent with a set of Node or DOMString objects.
     *
     * @param nodes List of Node or DOMString.
     */
    replaceWith(...nodes) {
        ChildNodeUtility.replaceWith(this, ...nodes);
    }
    /**
     * Inserts a set of Node or DOMString objects in the children list of this ChildNode's parent, just before this ChildNode. DOMString objects are inserted as equivalent Text nodes.
     *
     * @param nodes List of Node or DOMString.
     */
    before(...nodes) {
        ChildNodeUtility.before(this, ...nodes);
    }
    /**
     * Inserts a set of Node or DOMString objects in the children list of this ChildNode's parent, just after this ChildNode. DOMString objects are inserted as equivalent Text nodes.
     *
     * @param nodes List of Node or DOMString.
     */
    after(...nodes) {
        ChildNodeUtility.after(this, ...nodes);
    }
    /**
     * @override
     */
    [PropertySymbol.cloneNode](deep = false) {
        const clone = super[PropertySymbol.cloneNode](deep);
        clone[PropertySymbol.data] = this[PropertySymbol.data];
        return clone;
    }
}
//# sourceMappingURL=CharacterData.js.map