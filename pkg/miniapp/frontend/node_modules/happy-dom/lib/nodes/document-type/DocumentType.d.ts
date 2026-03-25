import Node from '../node/Node.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import NodeTypeEnum from '../node/NodeTypeEnum.js';
/**
 * DocumentType.
 */
export default class DocumentType extends Node {
    [PropertySymbol.nodeType]: NodeTypeEnum;
    [PropertySymbol.name]: string;
    [PropertySymbol.publicId]: string;
    [PropertySymbol.systemId]: string;
    cloneNode: (deep?: boolean) => DocumentType;
    /**
     * Returns name.
     *
     * @returns Name.
     */
    get name(): string;
    /**
     * Returns public ID.
     *
     * @returns Public ID.
     */
    get publicId(): string;
    /**
     * Returns system ID.
     *
     * @returns System ID.
     */
    get systemId(): string;
    /**
     * Node name.
     *
     * @returns Node name.
     */
    get nodeName(): string;
    /**
     * Converts to string.
     *
     * @returns String.
     */
    toString(): string;
    /**
     * @override
     */
    [PropertySymbol.cloneNode](deep?: boolean): DocumentType;
}
//# sourceMappingURL=DocumentType.d.ts.map