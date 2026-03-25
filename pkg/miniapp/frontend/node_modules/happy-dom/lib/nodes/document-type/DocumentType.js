import Node from '../node/Node.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import NodeTypeEnum from '../node/NodeTypeEnum.js';
/**
 * DocumentType.
 */
export default class DocumentType extends Node {
    [PropertySymbol.nodeType] = NodeTypeEnum.documentTypeNode;
    [PropertySymbol.name] = '';
    [PropertySymbol.publicId] = '';
    [PropertySymbol.systemId] = '';
    /**
     * Returns name.
     *
     * @returns Name.
     */
    get name() {
        return this[PropertySymbol.name];
    }
    /**
     * Returns public ID.
     *
     * @returns Public ID.
     */
    get publicId() {
        return this[PropertySymbol.publicId];
    }
    /**
     * Returns system ID.
     *
     * @returns System ID.
     */
    get systemId() {
        return this[PropertySymbol.systemId];
    }
    /**
     * Node name.
     *
     * @returns Node name.
     */
    get nodeName() {
        return this.name;
    }
    /**
     * Converts to string.
     *
     * @returns String.
     */
    toString() {
        return '[object DocumentType]';
    }
    /**
     * @override
     */
    [PropertySymbol.cloneNode](deep = false) {
        const clone = super[PropertySymbol.cloneNode](deep);
        clone[PropertySymbol.name] = this[PropertySymbol.name];
        clone[PropertySymbol.publicId] = this[PropertySymbol.publicId];
        clone[PropertySymbol.systemId] = this[PropertySymbol.systemId];
        return clone;
    }
}
//# sourceMappingURL=DocumentType.js.map