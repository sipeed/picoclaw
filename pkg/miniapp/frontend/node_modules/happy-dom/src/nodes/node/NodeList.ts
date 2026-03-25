import ClassMethodBinder from '../../utilities/ClassMethodBinder.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import Node from './Node.js';

/**
 * NodeList.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/NodeList
 */
class NodeList<T extends Node> {
	[index: number]: T;
	public [PropertySymbol.items]: T[];

	/**
	 * Constructor.
	 *
	 * @param illegalConstructorSymbol Illegal constructor symbol.
	 * @param items Items.
	 */
	constructor(illegalConstructorSymbol: symbol, items: T[]) {
		if (illegalConstructorSymbol !== PropertySymbol.illegalConstructor) {
			throw new TypeError('Illegal constructor');
		}

		this[PropertySymbol.items] = items;

		const methodBinder = new ClassMethodBinder(
			this,
			this.constructor !== NodeList ? [this.constructor, NodeList] : [NodeList]
		);

		const proxy = new Proxy(this, {
			get: (target, property) => {
				if (property === 'length') {
					return items.length;
				}
				if (property in target || typeof property === 'symbol') {
					methodBinder.bind(property);
					return target[property];
				}
				const index = Number(property);
				if (!isNaN(index)) {
					return items[index];
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
			ownKeys(): string[] {
				return Object.keys(items);
			},
			has(target, property): boolean {
				if (property in target) {
					return true;
				}

				if (typeof property === 'symbol') {
					return false;
				}

				const index = Number(property);
				return !isNaN(index) && index >= 0 && index < items.length;
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

		return proxy;
	}

	/**
	 * Returns length.
	 *
	 * @returns Length.
	 */
	public get length(): number {
		return this[PropertySymbol.items].length;
	}

	/**
	 * Returns `Symbol.toStringTag`.
	 *
	 * @returns `Symbol.toStringTag`.
	 */
	public get [Symbol.toStringTag](): string {
		return 'NodeList';
	}

	/**
	 * Returns `[object NodeList]`.
	 *
	 * @returns `[object NodeList]`.
	 */
	public toLocaleString(): string {
		return '[object NodeList]';
	}

	/**
	 * Returns `[object NodeList]`.
	 *
	 * @returns `[object NodeList]`.
	 */
	public toString(): string {
		return '[object NodeList]';
	}

	/**
	 * Returns item by index.
	 *
	 * @param index Index.
	 */
	public item(index: number): T {
		const nodes = this[PropertySymbol.items];
		return index >= 0 && nodes[index] ? <T>nodes[index] : null;
	}

	/**
	 * Returns an iterator, allowing you to go through all values of the key/value pairs contained in this object.
	 *
	 * @returns Iterator.
	 */
	public [Symbol.iterator](): IterableIterator<T> {
		const items = <T[]>this[PropertySymbol.items];
		return items[Symbol.iterator]();
	}

	/**
	 * Returns an iterator, allowing you to go through all values of the key/value pairs contained in this object.
	 *
	 * @returns Iterator.
	 */
	public values(): IterableIterator<T> {
		return (<T[]>this[PropertySymbol.items]).values();
	}

	/**
	 * Returns an iterator, allowing you to go through all key/value pairs contained in this object.
	 *
	 * @returns Iterator.
	 */
	public entries(): IterableIterator<[number, T]> {
		return (<T[]>this[PropertySymbol.items]).entries();
	}

	/**
	 * Executes a provided callback function once for each DOMTokenList element.
	 *
	 * @param callback Function.
	 * @param thisArg thisArg.
	 */
	public forEach(callback: (currentValue, currentIndex, listObj) => void, thisArg?: this): void {
		return (<T[]>this[PropertySymbol.items]).forEach(callback, thisArg);
	}

	/**
	 * Returns an iterator, allowing you to go through all keys of the key/value pairs contained in this object.
	 *
	 * @returns Iterator.
	 */
	public keys(): IterableIterator<number> {
		return (<T[]>this[PropertySymbol.items]).keys();
	}
}

export default NodeList;
