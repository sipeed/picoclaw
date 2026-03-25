import Node from '../node/Node.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import NodeTypeEnum from '../node/NodeTypeEnum.js';
/**
 * Attribute node interface.
 *
 * Reference: https://developer.mozilla.org/en-US/docs/Web/API/Attr.
 */
export default class Attr extends Node {
    [PropertySymbol.nodeType] = NodeTypeEnum.attributeNode;
    [PropertySymbol.namespaceURI] = null;
    [PropertySymbol.name] = null;
    [PropertySymbol.localName] = null;
    [PropertySymbol.prefix] = null;
    [PropertySymbol.value] = null;
    [PropertySymbol.specified] = true;
    [PropertySymbol.ownerElement] = null;
    /**
     * Returns specified.
     *
     * @returns Specified.
     */
    get specified() {
        return this[PropertySymbol.specified];
    }
    /**
     * Returns owner element.
     *
     * @returns Owner element.
     */
    get ownerElement() {
        return this[PropertySymbol.ownerElement];
    }
    /**
     * Returns value.
     *
     * @returns Value.
     */
    get value() {
        return this[PropertySymbol.value];
    }
    /**
     * Sets value.
     *
     * @param value Value.
     */
    set value(value) {
        this[PropertySymbol.value] = value;
    }
    /**
     * Returns name.
     *
     * @returns Name.
     */
    get name() {
        return this[PropertySymbol.name];
    }
    /**
     * Returns local name.
     *
     * @returns Local name.
     */
    get localName() {
        return this[PropertySymbol.localName];
    }
    /**
     * Returns prefix.
     *
     * @returns Prefix.
     */
    get prefix() {
        return this[PropertySymbol.prefix];
    }
    /**
     * @override
     */
    get textContent() {
        return this[PropertySymbol.value];
    }
    /**
     * Returns namespace URI.
     *
     * @returns Namespace URI.
     */
    get namespaceURI() {
        return this[PropertySymbol.namespaceURI];
    }
    /**
     * @override
     */
    [PropertySymbol.cloneNode](deep = false) {
        const clone = super[PropertySymbol.cloneNode](deep);
        clone[PropertySymbol.namespaceURI] = this[PropertySymbol.namespaceURI];
        clone[PropertySymbol.name] = this[PropertySymbol.name];
        clone[PropertySymbol.localName] = this[PropertySymbol.localName];
        clone[PropertySymbol.prefix] = this[PropertySymbol.prefix];
        clone[PropertySymbol.value] = this[PropertySymbol.value];
        clone[PropertySymbol.specified] = this[PropertySymbol.specified];
        return clone;
    }
}
//# sourceMappingURL=Attr.js.map