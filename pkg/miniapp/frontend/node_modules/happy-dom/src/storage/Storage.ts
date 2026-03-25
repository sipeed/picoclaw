import ClassMethodBinder from '../utilities/ClassMethodBinder.js';
import * as PropertySymbol from '../PropertySymbol.js';

/**
 * Storage.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/Storage
 */
export default class Storage {
	public [PropertySymbol.data]: { [key: string]: string } = {};

	/**
	 * Constructor.
	 */
	constructor() {
		const data = this[PropertySymbol.data];

		const methodBinder = new ClassMethodBinder(this, [Storage]);

		return new Proxy(this, {
			get: (target, property) => {
				if (property in target || typeof property === 'symbol') {
					methodBinder.bind(property);
					return target[property];
				}
				if (property in data) {
					return data[property];
				}
			},
			set(target, property, newValue): boolean {
				methodBinder.bind(property);

				if (property in target || typeof property === 'symbol') {
					return true;
				}

				data[String(property)] = String(newValue);
				return true;
			},
			deleteProperty(_target, property): boolean {
				if (property in data) {
					delete data[String(property)];
					return true;
				}
				return false;
			},
			ownKeys(): string[] {
				return Object.keys(data);
			},
			has(target, property): boolean {
				if (property in target || property in data) {
					return true;
				}

				return false;
			},
			defineProperty(target, property, descriptor): boolean {
				methodBinder.preventBinding(property);

				if (property in target) {
					Object.defineProperty(target, property, descriptor);
					return true;
				}

				if (descriptor.value !== undefined) {
					data[String(property)] = String(descriptor.value);
					return true;
				}

				return false;
			},
			getOwnPropertyDescriptor(target, property): PropertyDescriptor {
				if (property in target) {
					return;
				}

				const value = data[String(property)];

				if (value !== undefined) {
					return {
						value: value,
						writable: true,
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
		return Object.keys(this[PropertySymbol.data]).length;
	}

	/**
	 * Returns name of the nth key.
	 *
	 * @param index Index.
	 * @returns Name.
	 */
	public key(index: number): string | null {
		const name = Object.keys(this[PropertySymbol.data])[index];
		return name !== undefined ? name : null;
	}

	/**
	 * Sets item.
	 *
	 * @param name Name.
	 * @param item Item.
	 */
	public setItem(name: string, item: string): void {
		this[PropertySymbol.data][name] = String(item);
	}

	/**
	 * Returns item.
	 *
	 * @param name Name.
	 * @returns Item.
	 */
	public getItem(name: string): string | null {
		return this[PropertySymbol.data][name] !== undefined ? this[PropertySymbol.data][name] : null;
	}

	/**
	 * Removes item.
	 *
	 * @param name Name.
	 */
	public removeItem(name: string): void {
		delete this[PropertySymbol.data][name];
	}

	/**
	 * Clears storage.
	 */
	public clear(): void {
		const data = this[PropertySymbol.data];
		for (const key of Object.keys(data)) {
			delete data[key];
		}
	}
}
