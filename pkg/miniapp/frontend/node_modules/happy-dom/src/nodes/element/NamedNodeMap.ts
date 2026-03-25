import * as PropertySymbol from '../../PropertySymbol.js';
import Attr from '../attr/Attr.js';
import DOMException from '../../exception/DOMException.js';
import DOMExceptionNameEnum from '../../exception/DOMExceptionNameEnum.js';
import Element from './Element.js';
import NamespaceURI from '../../config/NamespaceURI.js';
import StringUtility from '../../utilities/StringUtility.js';

/**
 * Named Node Map.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/NamedNodeMap
 */
export default class NamedNodeMap {
	[index: number]: Attr;

	// All items with the namespaceURI as prefix
	public [PropertySymbol.namespaceItems]: Map<string, Attr> = new Map();

	// Items without namespaceURI as prefix, where the HTML namespace is the default namespace
	public [PropertySymbol.namedItems]: Map<string, Attr[]> = new Map();

	public declare [PropertySymbol.ownerElement]: Element;

	/**
	 * Constructor.
	 *
	 * @param ownerElement Owner element.
	 */
	constructor(ownerElement: Element) {
		this[PropertySymbol.ownerElement] = ownerElement;
	}

	/**
	 * Returns length.
	 *
	 * @returns Length.
	 */
	public get length(): number {
		return this[PropertySymbol.namespaceItems].size;
	}

	/**
	 * Returns string.
	 *
	 * @returns string.
	 */
	public get [Symbol.toStringTag](): string {
		return 'NamedNodeMap';
	}

	/**
	 * Returns string.
	 *
	 * @returns string.
	 */
	public toString(): string {
		return '[object NamedNodeMap]';
	}

	/**
	 * Iterator.
	 *
	 * @returns Iterator.
	 */
	public [Symbol.iterator](): IterableIterator<Attr> {
		return this[PropertySymbol.namespaceItems].values();
	}

	/**
	 * Returns item by index.
	 *
	 * @param index Index.
	 */
	public item(index: number): Attr | null {
		const items = Array.from(this[PropertySymbol.namespaceItems].values());
		return index >= 0 && items[index] ? items[index] : null;
	}

	/**
	 * Returns named item.
	 *
	 * @param name Name.
	 * @returns Item.
	 */
	public getNamedItem(name: string): Attr | null {
		name = String(name);
		if (
			this[PropertySymbol.ownerElement][PropertySymbol.namespaceURI] === NamespaceURI.html &&
			this[PropertySymbol.ownerElement][PropertySymbol.ownerDocument][
				PropertySymbol.contentType
			] === 'text/html'
		) {
			return this[PropertySymbol.namedItems].get(StringUtility.asciiLowerCase(name))?.[0] || null;
		}
		return this[PropertySymbol.namedItems].get(name)?.[0] || null;
	}

	/**
	 * Returns item by name and namespace.
	 *
	 * @param namespace Namespace.
	 * @param localName Local name of the attribute.
	 * @returns Item.
	 */
	public getNamedItemNS(namespace: string, localName: string): Attr | null {
		if (namespace === '') {
			namespace = null;
		}

		return this[PropertySymbol.namespaceItems].get(`${namespace || ''}:${localName}`) || null;
	}

	/**
	 * Sets named item.
	 *
	 * @param item Item.
	 * @returns Replaced item.
	 */
	public setNamedItem(item: Attr): Attr | null {
		return this[PropertySymbol.setNamedItem](item);
	}

	/**
	 * Adds a new namespaced item.
	 *
	 * @alias setNamedItem()
	 * @param item Item.
	 * @returns Replaced item.
	 */
	public setNamedItemNS(item: Attr): Attr | null {
		return this[PropertySymbol.setNamedItem](item);
	}

	/**
	 * Removes an item.
	 *
	 * @throws DOMException
	 * @param name Name of item.
	 * @returns Removed item.
	 */
	public removeNamedItem(name: string): Attr {
		const item = this.getNamedItem(name);

		if (!item) {
			throw new DOMException(
				`Failed to execute 'removeNamedItem' on 'NamedNodeMap': No item with name '${name}' was found.`,
				DOMExceptionNameEnum.notFoundError
			);
		}

		this[PropertySymbol.removeNamedItem](item);

		return item;
	}

	/**
	 * Removes a namespaced item.
	 *
	 * @param namespace Namespace.
	 * @param localName Local name of the item.
	 * @returns Removed item.
	 */
	public removeNamedItemNS(namespace: string, localName: string): Attr | null {
		const item = this.getNamedItemNS(namespace, localName);

		if (!item) {
			throw new DOMException(
				`Failed to execute 'removeNamedItemNS' on 'NamedNodeMap': No item with name '${localName}' in namespace '${namespace}' was found.`,
				DOMExceptionNameEnum.notFoundError
			);
		}

		this[PropertySymbol.removeNamedItem](item);

		return item;
	}

	/**
	 * Sets named item.
	 *
	 * @param item Item.
	 * @param [ignoreListeners] Ignore listeners.
	 * @returns Replaced item.
	 */
	public [PropertySymbol.setNamedItem](item: Attr, ignoreListeners = false): Attr {
		if (
			item[PropertySymbol.ownerElement] !== null &&
			item[PropertySymbol.ownerElement] !== this[PropertySymbol.ownerElement]
		) {
			throw new this[PropertySymbol.ownerElement][PropertySymbol.window].DOMException(
				'The attribute is in use.',
				DOMExceptionNameEnum.inUseAttributeError
			);
		}

		item[PropertySymbol.ownerElement] = this[PropertySymbol.ownerElement];

		const replacedItem =
			this.getNamedItemNS(item[PropertySymbol.namespaceURI], item[PropertySymbol.localName]) ||
			null;

		const namedItems = this[PropertySymbol.namedItems].get(item[PropertySymbol.name]);

		if (replacedItem === item) {
			return item;
		}

		this[PropertySymbol.namespaceItems].set(
			`${item[PropertySymbol.namespaceURI] || ''}:${item[PropertySymbol.localName]}`,
			item
		);

		if (!namedItems?.length) {
			this[PropertySymbol.namedItems].set(item[PropertySymbol.name], [item]);
		} else {
			const index = namedItems.indexOf(replacedItem);
			if (index !== -1) {
				namedItems.splice(index, 1);
			}
			namedItems.push(item);
		}

		if (!ignoreListeners) {
			this[PropertySymbol.ownerElement][PropertySymbol.onSetAttribute](item, replacedItem);
		}

		return replacedItem;
	}

	/**
	 * Removes named item.
	 *
	 * @param item Item.
	 * @param ignoreListeners
	 */
	public [PropertySymbol.removeNamedItem](item: Attr, ignoreListeners = false): void {
		item[PropertySymbol.ownerElement] = null;

		this[PropertySymbol.namespaceItems].delete(
			`${item[PropertySymbol.namespaceURI] || ''}:${item[PropertySymbol.localName]}`
		);

		const namedItems = this[PropertySymbol.namedItems].get(item[PropertySymbol.name]);

		if (namedItems?.length) {
			const index = namedItems.indexOf(item);
			if (index !== -1) {
				namedItems.splice(index, 1);
			}
			if (!namedItems.length) {
				this[PropertySymbol.namedItems].delete(item[PropertySymbol.name]);
			}
		}

		if (!ignoreListeners) {
			this[PropertySymbol.ownerElement][PropertySymbol.onRemoveAttribute](item);
		}
	}
}
