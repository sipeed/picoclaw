import TextTrack from './TextTrack.js';
import EventTarget from '../../event/EventTarget.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import ClassMethodBinder from '../../utilities/ClassMethodBinder.js';

/**
 * TextTrackList.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/TextTrackList
 */
export default class TextTrackList extends EventTarget {
	// Internal properties
	public [PropertySymbol.items]: TextTrack[] = [];

	// Events
	public onaddtrack: (event: Event) => void = null;
	public onchange: (event: Event) => void = null;
	public onremovetrack: (event: Event) => void = null;

	/**
	 * Constructor.
	 *
	 * @param illegalConstructorSymbol Illegal constructor symbol.
	 * @param items Items.
	 */
	constructor(illegalConstructorSymbol: symbol, items: TextTrack[]) {
		super();

		if (illegalConstructorSymbol !== PropertySymbol.illegalConstructor) {
			throw new TypeError('Illegal constructor');
		}

		this[PropertySymbol.items] = items;

		const methodBinder = new ClassMethodBinder(this, [TextTrackList, EventTarget]);

		return new Proxy(this, {
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
	}

	/**
	 * Returns the number of TextTrack objects in the TextTrackList.
	 *
	 * @returns Number of TextTrack objects.
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
		return 'TextTrackList';
	}

	/**
	 * Returns `[object NodeList]`.
	 *
	 * @returns `[object NodeList]`.
	 */
	public toLocaleString(): string {
		return '[object TextTrackList]';
	}

	/**
	 * Returns `[object NodeList]`.
	 *
	 * @returns `[object NodeList]`.
	 */
	public toString(): string {
		return '[object TextTrackList]';
	}

	/**
	 * Returns an iterator, allowing you to go through all values of the key/value pairs contained in this object.
	 *
	 * @returns Iterator.
	 */
	public [Symbol.iterator](): IterableIterator<TextTrack> {
		const items = <TextTrack[]>this[PropertySymbol.items];
		return items[Symbol.iterator]();
	}

	/**
	 * Returns the TextTrack found within the TextTrackList whose id matches the specified string. If no match is found, null is returned.
	 *
	 * @param id Text track cue identifier.
	 * @returns TextTrack.
	 */
	public getTrackById(id: string): TextTrack | null {
		for (const cue of this[PropertySymbol.items]) {
			if (cue.id === id) {
				return cue;
			}
		}
		return null;
	}
}
