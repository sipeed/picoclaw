import ClassMethodBinder from '../../utilities/ClassMethodBinder.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import Element from './Element.js';

/**
 * HTMLCollection.
 *
 * We are extending Array here to improve performance.
 * However, we should not expose Array methods to the outside.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/HTMLCollection
 */
export default class HTMLCollection<T extends Element, NamedItem = T> {
	[index: number]: T;
	protected [PropertySymbol.query]: () => T[];

	/**
	 * Constructor.
	 *
	 * @param illegalConstructorSymbol Illegal constructor symbol.
	 * @param query Query function.
	 */
	constructor(illegalConstructorSymbol: symbol, query: () => T[]) {
		if (illegalConstructorSymbol !== PropertySymbol.illegalConstructor) {
			throw new TypeError('Illegal constructor');
		}

		this[PropertySymbol.query] = query;

		const methodBinder = new ClassMethodBinder(
			this,
			this.constructor !== HTMLCollection ? [this.constructor, HTMLCollection] : [HTMLCollection]
		);

		return new Proxy(this, {
			get: (target, property) => {
				if (property === 'length') {
					return query().length;
				}
				if (property in target || typeof property === 'symbol') {
					methodBinder.bind(property);
					return target[property];
				}
				const index = Number(property);
				if (!isNaN(index)) {
					return query()[index];
				}
				return target.namedItem(<string>property) || undefined;
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
				const keys: string[] = [];
				const items = query();
				for (let i = 0; i < items.length; i++) {
					const item = items[i];
					const name = item.getAttribute('id') || item.getAttribute('name');
					keys.push(String(i));

					if (name) {
						keys.push(name);
					}
				}
				return keys;
			},
			has(target, property): boolean {
				if (property in target) {
					return true;
				}

				const items = query();
				const index = Number(property);

				if (!isNaN(index) && index >= 0 && index < items.length) {
					return true;
				}

				property = String(property);

				for (let i = 0; i < items.length; i++) {
					const item = items[i];
					const name = item.getAttribute('id') || item.getAttribute('name');

					if (name && name === property) {
						return true;
					}
				}

				return false;
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

				const items = query();
				const index = Number(property);

				if (!isNaN(index) && index >= 0 && index < items.length) {
					return {
						value: items[index],
						writable: false,
						enumerable: true,
						configurable: true
					};
				}

				for (let i = 0; i < items.length; i++) {
					const item = items[i];
					const name = item.getAttribute('id') || item.getAttribute('name');

					if (name && name === property) {
						return {
							value: item,
							writable: false,
							enumerable: true,
							configurable: true
						};
					}
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
		return this[PropertySymbol.query]().length;
	}

	/**
	 * Returns `Symbol.toStringTag`.
	 *
	 * @returns `Symbol.toStringTag`.
	 */
	public get [Symbol.toStringTag](): string {
		return this.constructor.name;
	}

	/**
	 * Returns `[object HTMLCollection]`.
	 *
	 * @returns `[object HTMLCollection]`.
	 */
	public toLocaleString(): string {
		return `[object ${this.constructor.name}]`;
	}

	/**
	 * Returns `[object HTMLCollection]`.
	 *
	 * @returns `[object HTMLCollection]`.
	 */
	public toString(): string {
		return `[object ${this.constructor.name}]`;
	}

	/**
	 * Returns item by index.
	 *
	 * @param index Index.
	 */
	public item(index: number): T {
		const items = this[PropertySymbol.query]();
		return index >= 0 && items[index] ? items[index] : null;
	}

	/**
	 * Returns an iterator, allowing you to go through all values of the key/value pairs contained in this object.
	 *
	 * @returns Iterator.
	 */
	public [Symbol.iterator](): IterableIterator<T> {
		const items = this[PropertySymbol.query]();
		return items[Symbol.iterator]();
	}

	/**
	 * Returns named item.
	 *
	 * @param name Name.
	 * @returns Node.
	 */
	public namedItem(name: string): NamedItem | null {
		const items = this[PropertySymbol.query]();
		name = String(name);
		for (const item of items) {
			if (item.getAttribute('id') === name || item.getAttribute('name') === name) {
				return <NamedItem>(<unknown>item);
			}
		}
		return null;
	}
}
