import Node from '../node/Node.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import NodeTypeEnum from '../node/NodeTypeEnum.js';

/**
 * DocumentType.
 */
export default class DocumentType extends Node {
	public [PropertySymbol.nodeType] = NodeTypeEnum.documentTypeNode;
	public [PropertySymbol.name] = '';
	public [PropertySymbol.publicId] = '';
	public [PropertySymbol.systemId] = '';
	public declare cloneNode: (deep?: boolean) => DocumentType;

	/**
	 * Returns name.
	 *
	 * @returns Name.
	 */
	public get name(): string {
		return this[PropertySymbol.name];
	}

	/**
	 * Returns public ID.
	 *
	 * @returns Public ID.
	 */
	public get publicId(): string {
		return this[PropertySymbol.publicId];
	}

	/**
	 * Returns system ID.
	 *
	 * @returns System ID.
	 */
	public get systemId(): string {
		return this[PropertySymbol.systemId];
	}

	/**
	 * Node name.
	 *
	 * @returns Node name.
	 */
	public get nodeName(): string {
		return this.name;
	}

	/**
	 * Converts to string.
	 *
	 * @returns String.
	 */
	public toString(): string {
		return '[object DocumentType]';
	}

	/**
	 * @override
	 */
	public override [PropertySymbol.cloneNode](deep = false): DocumentType {
		const clone = <DocumentType>super[PropertySymbol.cloneNode](deep);
		clone[PropertySymbol.name] = this[PropertySymbol.name];
		clone[PropertySymbol.publicId] = this[PropertySymbol.publicId];
		clone[PropertySymbol.systemId] = this[PropertySymbol.systemId];
		return clone;
	}
}
