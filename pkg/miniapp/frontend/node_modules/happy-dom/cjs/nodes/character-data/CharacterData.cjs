"use strict";
var __createBinding = (this && this.__createBinding) || (Object.create ? (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    var desc = Object.getOwnPropertyDescriptor(m, k);
    if (!desc || ("get" in desc ? !m.__esModule : desc.writable || desc.configurable)) {
      desc = { enumerable: true, get: function() { return m[k]; } };
    }
    Object.defineProperty(o, k2, desc);
}) : (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    o[k2] = m[k];
}));
var __setModuleDefault = (this && this.__setModuleDefault) || (Object.create ? (function(o, v) {
    Object.defineProperty(o, "default", { enumerable: true, value: v });
}) : function(o, v) {
    o["default"] = v;
});
var __importStar = (this && this.__importStar) || function (mod) {
    if (mod && mod.__esModule) return mod;
    var result = {};
    if (mod != null) for (var k in mod) if (k !== "default" && Object.prototype.hasOwnProperty.call(mod, k)) __createBinding(result, mod, k);
    __setModuleDefault(result, mod);
    return result;
};
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const Node_js_1 = __importDefault(require("../node/Node.cjs"));
const PropertySymbol = __importStar(require("../../PropertySymbol.cjs"));
const CharacterDataUtility_js_1 = __importDefault(require("./CharacterDataUtility.cjs"));
const NonDocumentChildNodeUtility_js_1 = __importDefault(require("../child-node/NonDocumentChildNodeUtility.cjs"));
const ChildNodeUtility_js_1 = __importDefault(require("../child-node/ChildNodeUtility.cjs"));
const MutationRecord_js_1 = __importDefault(require("../../mutation-observer/MutationRecord.cjs"));
const MutationTypeEnum_js_1 = __importDefault(require("../../mutation-observer/MutationTypeEnum.cjs"));
/**
 * Character data base class.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/CharacterData.
 */
class CharacterData extends Node_js_1.default {
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
        this[PropertySymbol.reportMutation](new MutationRecord_js_1.default({
            target: this,
            type: MutationTypeEnum_js_1.default.characterData,
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
        return NonDocumentChildNodeUtility_js_1.default.previousElementSibling(this);
    }
    /**
     * Next element sibling.
     *
     * @returns Element.
     */
    get nextElementSibling() {
        return NonDocumentChildNodeUtility_js_1.default.nextElementSibling(this);
    }
    /**
     * Appends the given DOMString to the CharacterData.data string; when this method returns, data contains the concatenated DOMString.
     *
     * @param data Data.
     */
    appendData(data) {
        CharacterDataUtility_js_1.default.appendData(this, data);
    }
    /**
     * Removes the specified amount of characters, starting at the specified offset, from the CharacterData.data string; when this method returns, data contains the shortened DOMString.
     *
     * @param offset Offset.
     * @param count Count.
     */
    deleteData(offset, count) {
        CharacterDataUtility_js_1.default.deleteData(this, offset, count);
    }
    /**
     * Inserts the specified characters, at the specified offset, in the CharacterData.data string; when this method returns, data contains the modified DOMString.
     *
     * @param offset Offset.
     * @param data Data.
     */
    insertData(offset, data) {
        CharacterDataUtility_js_1.default.insertData(this, offset, data);
    }
    /**
     * Replaces the specified amount of characters, starting at the specified offset, with the specified DOMString; when this method returns, data contains the modified DOMString.
     *
     * @param offset Offset.
     * @param count Count.
     * @param data Data.
     */
    replaceData(offset, count, data) {
        CharacterDataUtility_js_1.default.replaceData(this, offset, count, data);
    }
    /**
     * Returns a DOMString containing the part of CharacterData.data of the specified length and starting at the specified offset.
     *
     * @param offset Offset.
     * @param count Count.
     */
    substringData(offset, count) {
        return CharacterDataUtility_js_1.default.substringData(this, offset, count);
    }
    /**
     * Removes the object from its parent children list.
     */
    remove() {
        ChildNodeUtility_js_1.default.remove(this);
    }
    /**
     * The Node.replaceWith() method replaces this Node in the children list of its parent with a set of Node or DOMString objects.
     *
     * @param nodes List of Node or DOMString.
     */
    replaceWith(...nodes) {
        ChildNodeUtility_js_1.default.replaceWith(this, ...nodes);
    }
    /**
     * Inserts a set of Node or DOMString objects in the children list of this ChildNode's parent, just before this ChildNode. DOMString objects are inserted as equivalent Text nodes.
     *
     * @param nodes List of Node or DOMString.
     */
    before(...nodes) {
        ChildNodeUtility_js_1.default.before(this, ...nodes);
    }
    /**
     * Inserts a set of Node or DOMString objects in the children list of this ChildNode's parent, just after this ChildNode. DOMString objects are inserted as equivalent Text nodes.
     *
     * @param nodes List of Node or DOMString.
     */
    after(...nodes) {
        ChildNodeUtility_js_1.default.after(this, ...nodes);
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
exports.default = CharacterData;
//# sourceMappingURL=CharacterData.cjs.map