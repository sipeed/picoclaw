import Element from '../element/Element.js';
import Node from '../node/Node.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import NodeTypeEnum from '../node/NodeTypeEnum.js';

/**
 * Attribute node interface.
 *
 * Reference: https://developer.mozilla.org/en-US/docs/Web/API/Attr.
 */
export default class Attr extends Node implements Attr {
	// Public properties
	public declare cloneNode: (deep?: boolean) => Attr;

	public [PropertySymbol.nodeType] = NodeTypeEnum.attributeNode;
	public [PropertySymbol.namespaceURI]: string | null = null;
	public [PropertySymbol.name]: string | null = null;
	public [PropertySymbol.localName]: string | null = null;
	public [PropertySymbol.prefix]: string | null = null;
	public [PropertySymbol.value]: string | null = null;
	public [PropertySymbol.specified] = true;
	public [PropertySymbol.ownerElement]: Element | null = null;

	/**
	 * Returns specified.
	 *
	 * @returns Specified.
	 */
	public get specified(): boolean {
		return this[PropertySymbol.specified];
	}

	/**
	 * Returns owner element.
	 *
	 * @returns Owner element.
	 */
	public get ownerElement(): Element | null {
		return this[PropertySymbol.ownerElement];
	}

	/**
	 * Returns value.
	 *
	 * @returns Value.
	 */
	public get value(): string {
		return this[PropertySymbol.value];
	}

	/**
	 * Sets value.
	 *
	 * @param value Value.
	 */
	public set value(value: string) {
		this[PropertySymbol.value] = value;
	}

	/**
	 * Returns name.
	 *
	 * @returns Name.
	 */
	public get name(): string {
		return this[PropertySymbol.name];
	}

	/**
	 * Returns local name.
	 *
	 * @returns Local name.
	 */
	public get localName(): string {
		return this[PropertySymbol.localName];
	}

	/**
	 * Returns prefix.
	 *
	 * @returns Prefix.
	 */
	public get prefix(): string {
		return this[PropertySymbol.prefix];
	}

	/**
	 * @override
	 */
	public get textContent(): string {
		return this[PropertySymbol.value];
	}

	/**
	 * Returns namespace URI.
	 *
	 * @returns Namespace URI.
	 */
	public get namespaceURI(): string | null {
		return this[PropertySymbol.namespaceURI];
	}

	/**
	 * @override
	 */
	public override [PropertySymbol.cloneNode](deep = false): Attr {
		const clone = <Attr>super[PropertySymbol.cloneNode](deep);

		clone[PropertySymbol.namespaceURI] = this[PropertySymbol.namespaceURI];
		clone[PropertySymbol.name] = this[PropertySymbol.name];
		clone[PropertySymbol.localName] = this[PropertySymbol.localName];
		clone[PropertySymbol.prefix] = this[PropertySymbol.prefix];
		clone[PropertySymbol.value] = this[PropertySymbol.value];
		clone[PropertySymbol.specified] = this[PropertySymbol.specified];

		return clone;
	}
}
