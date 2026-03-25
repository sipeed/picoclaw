import ClassMethodBinder from '../utilities/ClassMethodBinder.js';
import Element from '../nodes/element/Element.js';
import * as PropertySymbol from '../PropertySymbol.js';

const ATTRIBUTE_SPLIT_REGEXP = /[\t\f\n\r ]+/;

/**
 * DOM Token List.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/DOMTokenList.
 */
export default class DOMTokenList {
	[index: number]: string;

	private [PropertySymbol.ownerElement]: Element;
	private [PropertySymbol.attributeName]: string;
	private [PropertySymbol.cache]: { items: string[]; attributeValue: string } = {
		items: [],
		attributeValue: ''
	};

	/**
	 * Constructor.
	 *
	 * @param illegalConstructorSymbol Illegal constructor symbol.
	 * @param ownerElement Owner element.
	 * @param attributeName Attribute name.
	 */
	constructor(illegalConstructorSymbol: symbol, ownerElement: Element, attributeName: string) {
		if (illegalConstructorSymbol !== PropertySymbol.illegalConstructor) {
			throw new TypeError('Illegal constructor');
		}

		this[PropertySymbol.ownerElement] = ownerElement;
		this[PropertySymbol.attributeName] = attributeName;

		const methodBinder = new ClassMethodBinder(this, [DOMTokenList]);

		return new Proxy(this, {
			get: (target, property) => {
				if (property === 'length') {
					return target[PropertySymbol.getTokenList]().length;
				}
				if (property in target || typeof property === 'symbol') {
					methodBinder.bind(property);
					return target[property];
				}
				const index = Number(property);
				if (!isNaN(index)) {
					return target[PropertySymbol.getTokenList]()[index];
				}
			},
			set(target, property, newValue): boolean {
				methodBinder.bind(property);
				if (typeof property === 'symbol') {
					target[property] = newValue;
					return true;
				}
				const index = Number(property);
				if (isNaN(index)) {
					target[property] = newValue;
				}
				return true;
			},
			deleteProperty(target, property): boolean {
				if (typeof property === 'symbol') {
					delete target[property];
					return true;
				}
				const index = Number(property);
				if (isNaN(index)) {
					delete target[property];
				}
				return true;
			},
			ownKeys(target): string[] {
				return Object.keys(target[PropertySymbol.getTokenList]());
			},
			has(target, property): boolean {
				if (property in target) {
					return true;
				}

				if (typeof property === 'symbol') {
					return false;
				}

				const index = Number(property);
				return !isNaN(index) && index >= 0 && index < target[PropertySymbol.getTokenList]().length;
			},
			defineProperty(target, property, descriptor): boolean {
				methodBinder.preventBinding(property);

				if (property in target) {
					Object.defineProperty(target, property, descriptor);
					return true;
				}

				return false;
			},
			getOwnPropertyDescriptor(target, property): PropertyDescriptor {
				if (property in target || typeof property === 'symbol') {
					return;
				}

				const index = Number(property);
				const items = target[PropertySymbol.getTokenList]();

				if (!isNaN(index) && items[index]) {
					return {
						value: items[index],
						writable: false,
						enumerable: true,
						configurable: true
					};
				}
			}
		});
	}

	/**
	 * Returns length.
	 *
	 * @returns Length.
	 */
	public get length(): number {
		return this[PropertySymbol.getTokenList]().length;
	}

	/**
	 * Set value.
	 *
	 * @param value Value.
	 */
	public set value(value: string) {
		this[PropertySymbol.ownerElement].setAttribute(this[PropertySymbol.attributeName], value);
	}

	/**
	 * Get value.
	 */
	public get value(): string {
		return this[PropertySymbol.ownerElement].getAttribute(this[PropertySymbol.attributeName]);
	}

	/**
	 * Returns an iterator, allowing you to go through all values of the key/value pairs contained in this object.
	 */
	public [Symbol.iterator](): IterableIterator<string> {
		return this[PropertySymbol.getTokenList]().values();
	}

	/**
	 * Get ClassName.
	 *
	 * @param index Index.
	 * */
	public item(index: number | string): string {
		const items = this[PropertySymbol.getTokenList]();
		if (typeof index === 'number') {
			return items[index] ? items[index] : null;
		}
		index = Number(index);
		index = isNaN(index) ? 0 : index;
		return items[index] ? items[index] : null;
	}

	/**
	 * Replace Token.
	 *
	 * @param token Token.
	 * @param newToken NewToken.
	 */
	public replace(token: string, newToken: string): boolean {
		const list = this[PropertySymbol.getTokenList]();
		const index = list.indexOf(token);
		if (index === -1) {
			return false;
		}
		list[index] = newToken;
		this[PropertySymbol.ownerElement].setAttribute(
			this[PropertySymbol.attributeName],
			list.join(' ')
		);
		return true;
	}

	/**
	 * Supports.
	 *
	 * @param _token Token.
	 */
	public supports(_token: string): boolean {
		return false;
	}

	/**
	 * Returns an iterator, allowing you to go through all values of the key/value pairs contained in this object.
	 */
	public values(): IterableIterator<string> {
		return this[PropertySymbol.getTokenList]().values();
	}

	/**
	 * Returns an iterator, allowing you to go through all key/value pairs contained in this object.
	 */
	public entries(): IterableIterator<[number, string]> {
		return this[PropertySymbol.getTokenList]().entries();
	}

	/**
	 * Executes a provided callback function once for each DOMTokenList element.
	 *
	 * @param callback
	 * @param thisArg
	 */
	public forEach(callback: (currentValue, currentIndex, listObj) => void, thisArg?: this): void {
		return this[PropertySymbol.getTokenList]().forEach(callback, thisArg);
	}

	/**
	 * Returns an iterator, allowing you to go through all keys of the key/value pairs contained in this object.
	 *
	 */
	public keys(): IterableIterator<number> {
		return this[PropertySymbol.getTokenList]().keys();
	}

	/**
	 * Adds tokens.
	 *
	 * @param tokens Tokens.
	 */
	public add(...tokens: string[]): void {
		const list = this[PropertySymbol.getTokenList]();

		for (const token of tokens) {
			const index = list.indexOf(token);
			if (index === -1) {
				list.push(token);
			} else {
				list[index] = token;
			}
		}

		this[PropertySymbol.ownerElement].setAttribute(
			this[PropertySymbol.attributeName],
			list.join(' ')
		);
	}

	/**
	 * Removes tokens.
	 *
	 * @param tokens Tokens.
	 */
	public remove(...tokens: string[]): void {
		const list = this[PropertySymbol.getTokenList]();

		for (const token of tokens) {
			const index = list.indexOf(token);
			if (index !== -1) {
				list.splice(index, 1);
			}
		}

		this[PropertySymbol.ownerElement].setAttribute(
			this[PropertySymbol.attributeName],
			list.join(' ')
		);
	}

	/**
	 * Check if the list contains a class.
	 *
	 * @param className Class name.
	 * @returns TRUE if it contains.
	 */
	public contains(className: string): boolean {
		return this[PropertySymbol.getTokenList]().includes(className);
	}

	/**
	 * Toggle a class name.
	 *
	 * @param token A string representing the class name you want to toggle.
	 * @param [force] If included, turns the toggle into a one way-only operation. If set to `false`, then class name will only be removed, but not added. If set to `true`, then class name will only be added, but not removed.
	 * @returns A boolean value, `true` or `false`, indicating whether class name is in the list after the call or not.
	 */
	public toggle(token: string, force?: boolean): boolean {
		let shouldAdd: boolean;

		if (force !== undefined) {
			shouldAdd = force;
		} else {
			shouldAdd = !this.contains(token);
		}

		if (shouldAdd) {
			this.add(token);
			return true;
		}

		this.remove(token);

		return false;
	}

	/**
	 * Returns token list from attribute value.
	 *
	 * @see https://infra.spec.whatwg.org/#split-on-ascii-whitespace
	 */
	public [PropertySymbol.getTokenList](): string[] {
		const attributeValue =
			this[PropertySymbol.ownerElement].getAttribute(this[PropertySymbol.attributeName]) ?? '';

		const cache = this[PropertySymbol.cache];

		if (cache.attributeValue === attributeValue) {
			return cache.items;
		}

		// It is possible to make this statement shorter by using Array.from() and Set, but this is faster when comparing using a bench test.
		const items = [];
		const trimmed = attributeValue.trim();

		if (trimmed) {
			for (const item of trimmed.split(ATTRIBUTE_SPLIT_REGEXP)) {
				if (!items.includes(item)) {
					items.push(item);
				}
			}
		}

		cache.attributeValue = attributeValue;
		cache.items = items;

		return items;
	}

	/**
	 * Returns DOMTokenList value.
	 */
	public toString(): string {
		return this.value || '';
	}
}
