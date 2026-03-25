import HTMLElement from '../html-element/HTMLElement.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import HTMLFormElement from '../html-form-element/HTMLFormElement.js';
import ValidityState from '../../validity-state/ValidityState.js';
import HTMLLabelElement from '../html-label-element/HTMLLabelElement.js';
import HTMLOptionElement from '../html-option-element/HTMLOptionElement.js';
import HTMLOptionsCollection from './HTMLOptionsCollection.js';
import Event from '../../event/Event.js';
import HTMLLabelElementUtility from '../html-label-element/HTMLLabelElementUtility.js';
import HTMLCollection from '../element/HTMLCollection.js';
import NodeTypeEnum from '../node/NodeTypeEnum.js';
import QuerySelector from '../../query-selector/QuerySelector.js';
import NodeList from '../node/NodeList.js';
import ClassMethodBinder from '../../utilities/ClassMethodBinder.js';
import Node from '../node/Node.js';
import Element from '../element/Element.js';
import EventTarget from '../../event/EventTarget.js';

/**
 * HTML Select Element.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/HTMLSelectElement.
 */
export default class HTMLSelectElement extends HTMLElement {
	// Internal properties.
	public [PropertySymbol.validationMessage] = '';
	public [PropertySymbol.validity] = new ValidityState(this);
	public [PropertySymbol.options]: HTMLOptionsCollection | null = null;
	public [PropertySymbol.selectedOptions]: HTMLCollection<HTMLOptionElement> | null = null;
	public [PropertySymbol.selectedIndex]: number = -1;
	public [PropertySymbol.proxy]: HTMLSelectElement;

	// Events
	public onchange: ((event: Event) => void) | null = null;
	public oninput: ((event: Event) => void) | null = null;

	/**
	 * Constructor.
	 */
	constructor() {
		super();

		const methodBinder = new ClassMethodBinder(this, [
			HTMLSelectElement,
			HTMLElement,
			Element,
			Node,
			EventTarget
		]);

		const proxy = new Proxy(this, {
			get: (target, property) => {
				if (property in target || typeof property === 'symbol') {
					methodBinder.bind(property);
					return target[property];
				}
				const index = Number(property);
				if (!isNaN(index)) {
					return QuerySelector.querySelectorAll(target, 'option')[PropertySymbol.items][index];
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
					return true;
				}

				if (!newValue || !(newValue instanceof HTMLOptionElement)) {
					throw new this[PropertySymbol.window].Error(
						`TypeError: Failed to set an indexed property [${index}] on 'HTMLSelectElement': parameter 2 is not of type 'HTMLOptionElement'.`
					);
				}

				const options = QuerySelector.querySelectorAll(target, 'option')[PropertySymbol.items];
				const childNodes = target[PropertySymbol.nodeArray];

				while (childNodes.length) {
					target[PropertySymbol.removeChild](childNodes[0]);
				}

				for (let i = 0; i <= index; i++) {
					if (i === index) {
						target[PropertySymbol.appendChild](newValue);
					} else if (options[i]) {
						target[PropertySymbol.appendChild](options[i]);
					} else {
						target[PropertySymbol.appendChild](
							target[PropertySymbol.ownerDocument].createElement('option')
						);
					}
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
				return Object.keys(QuerySelector.querySelectorAll(target, 'option')[PropertySymbol.items]);
			},
			has(target, property): boolean {
				if (property in target) {
					return true;
				}

				if (typeof property === 'symbol') {
					return false;
				}

				const index = Number(property);

				if (!isNaN(index)) {
					return !!QuerySelector.querySelectorAll(target, 'option')[PropertySymbol.items][index];
				}

				return false;
			},
			defineProperty(target, property, descriptor): boolean {
				methodBinder.preventBinding(property);

				const index = Number(property);

				if (isNaN(index)) {
					Object.defineProperty(target, property, descriptor);
					return true;
				}

				if (!descriptor.value || !(descriptor.value instanceof HTMLOptionElement)) {
					throw new this[PropertySymbol.window].Error(
						`TypeError: Failed to set an indexed property [${index}] on 'HTMLSelectElement': parameter 2 is not of type 'HTMLOptionElement'.`
					);
				}

				const options = QuerySelector.querySelectorAll(target, 'option')[PropertySymbol.items];
				const childNodes = target[PropertySymbol.nodeArray];

				while (childNodes.length) {
					target[PropertySymbol.removeChild](childNodes[0]);
				}

				for (let i = 0; i <= index; i++) {
					if (i === index) {
						target[PropertySymbol.appendChild](descriptor.value);
					} else if (options[i]) {
						target[PropertySymbol.appendChild](options[i]);
					} else {
						target[PropertySymbol.appendChild](
							target[PropertySymbol.ownerDocument].createElement('option')
						);
					}
				}

				return true;
			},
			getOwnPropertyDescriptor(target, property): PropertyDescriptor {
				if (property in target) {
					return Object.getOwnPropertyDescriptor(target, property);
				}

				const index = Number(property);

				if (isNaN(index)) {
					return;
				}

				const options = QuerySelector.querySelectorAll(target, 'option')[PropertySymbol.items];

				if (!options[index]) {
					return;
				}

				return {
					value: options[index],
					writable: true,
					enumerable: true,
					configurable: true
				};
			}
		});

		this[PropertySymbol.proxy] = proxy;
		this[PropertySymbol.selectNode] = proxy;

		return proxy;
	}

	/**
	 * Returns length.
	 *
	 * @returns Length.
	 */
	public get length(): number {
		return QuerySelector.querySelectorAll(this, 'option')[PropertySymbol.items].length;
	}

	/**
	 * Returns options.
	 *
	 * @returns Options.
	 */
	public get options(): HTMLOptionsCollection {
		if (!this[PropertySymbol.options]) {
			this[PropertySymbol.options] = new HTMLOptionsCollection(
				PropertySymbol.illegalConstructor,
				this
			);
		}

		return this[PropertySymbol.options];
	}

	/**
	 * Returns validation message.
	 *
	 * @returns Validation message.
	 */
	public get validationMessage(): string {
		return this[PropertySymbol.validationMessage];
	}

	/**
	 * Returns validity.
	 *
	 * @returns Validity.
	 */
	public get validity(): ValidityState {
		return this[PropertySymbol.validity];
	}

	/**
	 * Returns name.
	 *
	 * @returns Name.
	 */
	public get name(): string {
		return this.getAttribute('name') || '';
	}

	/**
	 * Sets name.
	 *
	 * @param name Name.
	 */
	public set name(name: string) {
		this.setAttribute('name', name);
	}

	/**
	 * Returns disabled.
	 *
	 * @returns Disabled.
	 */
	public get disabled(): boolean {
		return this.getAttribute('disabled') !== null;
	}

	/**
	 * Sets disabled.
	 *
	 * @param disabled Disabled.
	 */
	public set disabled(disabled: boolean) {
		if (!disabled) {
			this.removeAttribute('disabled');
		} else {
			this.setAttribute('disabled', '');
		}
	}

	/**
	 * Returns multiple.
	 *
	 * @returns Multiple.
	 */
	public get multiple(): boolean {
		return this.getAttribute('multiple') !== null;
	}

	/**
	 * Sets multiple.
	 *
	 * @param multiple Multiple.
	 */
	public set multiple(multiple: boolean) {
		if (!multiple) {
			this.removeAttribute('multiple');
		} else {
			this.setAttribute('multiple', '');
		}
	}

	/**
	 * Returns autofocus.
	 *
	 * @returns Autofocus.
	 */
	public get autofocus(): boolean {
		return this.getAttribute('autofocus') !== null;
	}

	/**
	 * Sets autofocus.
	 *
	 * @param autofocus Autofocus.
	 */
	public set autofocus(autofocus: boolean) {
		if (!autofocus) {
			this.removeAttribute('autofocus');
		} else {
			this.setAttribute('autofocus', '');
		}
	}

	/**
	 * Returns required.
	 *
	 * @returns Required.
	 */
	public get required(): boolean {
		return this.getAttribute('required') !== null;
	}

	/**
	 * Sets required.
	 *
	 * @param required Required.
	 */
	public set required(required: boolean) {
		if (!required) {
			this.removeAttribute('required');
		} else {
			this.setAttribute('required', '');
		}
	}

	/**
	 * Returns type.
	 *
	 * @returns type.
	 */
	public get type(): string {
		return this.hasAttributeNS(null, 'multiple') ? 'select-multiple' : 'select-one';
	}

	/**
	 * Returns value.
	 *
	 * @returns Value.
	 */
	public get value(): string {
		const options = QuerySelector.querySelectorAll(this, 'option')[PropertySymbol.items];

		for (let i = 0, max = options.length; i < max; i++) {
			const option = <HTMLOptionElement>options[i];
			if (option[PropertySymbol.selectedness]) {
				return option.value;
			}
		}

		return '';
	}

	/**
	 * Sets value.
	 *
	 * @param value Value.
	 */
	public set value(value: string) {
		const options = QuerySelector.querySelectorAll(this, 'option')[PropertySymbol.items];

		this[PropertySymbol.selectedIndex] = -1;

		for (let i = 0, max = options.length; i < max; i++) {
			const option = <HTMLOptionElement>options[i];
			if (option.value === value) {
				option[PropertySymbol.selectedness] = true;
				option[PropertySymbol.dirtyness] = true;
				this[PropertySymbol.selectedIndex] = i;
			} else {
				option[PropertySymbol.selectedness] = false;
			}
		}
	}

	/**
	 * Returns value.
	 *
	 * @returns Value.
	 */
	public get selectedIndex(): number {
		return this[PropertySymbol.selectedIndex];
	}

	/**
	 * Sets value.
	 *
	 * @param selectedIndex Selected index.
	 */
	public set selectedIndex(selectedIndex: number) {
		selectedIndex = Number(selectedIndex);

		if (isNaN(selectedIndex)) {
			return;
		}

		const options = QuerySelector.querySelectorAll(this, 'option')[PropertySymbol.items];

		this[PropertySymbol.selectedIndex] = -1;

		if (typeof selectedIndex === 'number' && !isNaN(selectedIndex)) {
			for (let i = 0, max = options.length; i < max; i++) {
				(<HTMLOptionElement>options[i])[PropertySymbol.selectedness] = false;
			}

			const selectedOption = <HTMLOptionElement>options[selectedIndex];
			if (selectedOption) {
				selectedOption[PropertySymbol.selectedness] = true;
				selectedOption[PropertySymbol.dirtyness] = true;
				this[PropertySymbol.selectedIndex] = selectedIndex;
			}
		}
	}

	/**
	 * Returns selected options.
	 *
	 * @returns HTMLCollection.
	 */
	public get selectedOptions(): HTMLCollection<HTMLOptionElement> {
		if (!this[PropertySymbol.selectedOptions]) {
			this[PropertySymbol.selectedOptions] = new HTMLCollection<HTMLOptionElement>(
				PropertySymbol.illegalConstructor,
				() => {
					const options = QuerySelector.querySelectorAll(this, 'option')[PropertySymbol.items];

					// If we hit the cache, we should recieve the same Array instance as before, which will then contain the selected options.
					if (options[PropertySymbol.selectedOptions]) {
						return options[PropertySymbol.selectedOptions];
					}

					const selectedOptions = [];

					for (let i = 0, max = options.length; i < max; i++) {
						const option = <HTMLOptionElement>options[i];
						if (option[PropertySymbol.selectedness]) {
							selectedOptions.push(option);
						}
					}

					options[PropertySymbol.selectedOptions] = selectedOptions;

					return selectedOptions;
				}
			);
		}

		return this[PropertySymbol.selectedOptions];
	}

	/**
	 * Returns the associated label elements.
	 *
	 * @returns Label elements.
	 */
	public get labels(): NodeList<HTMLLabelElement> {
		return HTMLLabelElementUtility.getAssociatedLabelElements(this);
	}

	/**
	 * Returns the parent form element.
	 *
	 * @returns Form.
	 */
	public get form(): HTMLFormElement {
		if (this[PropertySymbol.formNode]) {
			return this[PropertySymbol.formNode];
		}
		const id = this.getAttribute('form');
		if (!id || !this[PropertySymbol.isConnected]) {
			return null;
		}
		return <HTMLFormElement>this[PropertySymbol.ownerDocument].getElementById(id);
	}

	/**
	 * Returns "true" if it will validate.
	 *
	 * @returns "true" if it will validate.
	 */
	public get willValidate(): boolean {
		return (
			this.type !== 'hidden' &&
			this.type !== 'reset' &&
			this.type !== 'button' &&
			!this.disabled &&
			!this['readOnly']
		);
	}

	/**
	 * Returns item from options collection by index.
	 *
	 * @param index Index.
	 */
	public item(index: number): HTMLOptionElement {
		return this[PropertySymbol.options].item(index);
	}

	/**
	 * Adds new option to options collection.
	 *
	 * @param element HTMLOptionElement to add.
	 * @param before HTMLOptionElement or index number.
	 */
	public add(element: HTMLOptionElement, before?: number | HTMLOptionElement): void {
		const options = QuerySelector.querySelectorAll(this, 'option')[PropertySymbol.items];

		if (!before && before !== 0) {
			const childNodes = this[PropertySymbol.nodeArray];

			while (childNodes.length) {
				this[PropertySymbol.removeChild](childNodes[0]);
			}

			for (const option of options) {
				this[PropertySymbol.appendChild](option);
			}

			this[PropertySymbol.appendChild](element);

			return;
		}

		const window = this[PropertySymbol.window];

		if (typeof before !== 'number') {
			if (!(before instanceof HTMLOptionElement)) {
				throw new window.DOMException(
					"Failed to execute 'add' on 'HTMLFormElement': The node before which the new node is to be inserted before is not an 'HTMLOptionElement'."
				);
			}

			before = options.indexOf(<HTMLOptionElement>before);
		}

		const optionsElement = options[before];

		if (!optionsElement) {
			throw new window.DOMException(
				"Failed to execute 'add' on 'HTMLFormElement': The node before which the new node is to be inserted before is not a child of this node."
			);
		}

		const childNodes = this[PropertySymbol.nodeArray];

		while (childNodes.length) {
			this[PropertySymbol.removeChild](childNodes[0]);
		}

		for (let i = 0, max = options.length; i < max; i++) {
			if (i === before) {
				this[PropertySymbol.appendChild](element);
			}
			this[PropertySymbol.appendChild](options[i]);
		}
	}

	/**
	 * Removes indexed element from collection or the select element.
	 *
	 * @param [index] Index.
	 */
	public override remove(index?: number): void {
		if (typeof index === 'number') {
			const options = QuerySelector.querySelectorAll(this, 'option')[PropertySymbol.items];

			if (!options[index]) {
				return;
			}

			const childNodes = this[PropertySymbol.nodeArray];

			while (childNodes.length) {
				this[PropertySymbol.removeChild](childNodes[0]);
			}

			for (let i = 0, max = options.length; i < max; i++) {
				if (i !== index) {
					this[PropertySymbol.appendChild](options[i]);
				}
			}
		} else {
			super.remove();
		}
	}

	/**
	 * Sets validation message.
	 *
	 * @param message Message.
	 */
	public setCustomValidity(message: string): void {
		this[PropertySymbol.validationMessage] = String(message);
	}

	/**
	 * Checks validity.
	 *
	 * @returns "true" if the field is valid.
	 */
	public checkValidity(): boolean {
		const valid = this.disabled || this[PropertySymbol.validity].valid;
		if (!valid) {
			this.dispatchEvent(new Event('invalid', { bubbles: true, cancelable: true }));
		}
		return valid;
	}

	/**
	 * Reports validity.
	 *
	 * @returns "true" if the field is valid.
	 */
	public reportValidity(): boolean {
		return this.checkValidity();
	}

	/**
	 * Updates option item.
	 *
	 * Based on:
	 * https://github.com/jsdom/jsdom/blob/master/lib/jsdom/living/nodes/HTMLSelectElement-impl.js
	 *
	 * @see https://html.spec.whatwg.org/multipage/form-elements.html#selectedness-setting-algorithm
	 * @param [selectedOption] Selected option.
	 */
	public [PropertySymbol.updateSelectedness](selectedOption?: HTMLOptionElement): void {
		const isMultiple = this.hasAttribute('multiple');
		const options = QuerySelector.querySelectorAll(this, 'option')[PropertySymbol.items];
		const selected: HTMLOptionElement[] = [];

		if (selectedOption) {
			this[PropertySymbol.selectedIndex] = -1;
		}

		if (!isMultiple) {
			for (let i = 0, max = options.length; i < max; i++) {
				const option = <HTMLOptionElement>options[i];

				if (selectedOption) {
					option[PropertySymbol.selectedness] = option === selectedOption;

					if (option === selectedOption) {
						this[PropertySymbol.selectedIndex] = i;
					}
				}

				if (option[PropertySymbol.selectedness]) {
					selected.push(option);
				}
			}
		}

		const size = this.#getDisplaySize();

		if (size === 1 && !selected.length) {
			this[PropertySymbol.selectedIndex] = -1;
			for (let i = 0, max = options.length; i < max; i++) {
				const option = <HTMLOptionElement>options[i];
				const parentNode = <HTMLElement>option[PropertySymbol.parentNode];
				let disabled = option.hasAttributeNS(null, 'disabled');

				if (
					parentNode &&
					parentNode[PropertySymbol.nodeType] === NodeTypeEnum.elementNode &&
					parentNode[PropertySymbol.tagName] === 'OPTGROUP' &&
					parentNode.hasAttributeNS(null, 'disabled')
				) {
					disabled = true;
				}

				if (!disabled) {
					option[PropertySymbol.selectedness] = true;
					this[PropertySymbol.selectedIndex] = i;
					break;
				}
			}
		} else if (selected.length >= 2) {
			this[PropertySymbol.selectedIndex] = -1;

			for (let i = 0, max = options.length; i < max; i++) {
				(<HTMLOptionElement>options[i])[PropertySymbol.selectedness] = i === selected.length - 1;

				if (i === selected.length - 1) {
					this[PropertySymbol.selectedIndex] = i;
				}
			}
		}
	}

	/**
	 * Returns display size.
	 *
	 * @returns Display size.
	 */
	#getDisplaySize(): number {
		if (this.hasAttributeNS(null, 'size')) {
			const size = parseInt(this.getAttribute('size'));
			if (!isNaN(size) && size >= 0) {
				return size;
			}
		}
		return this.hasAttributeNS(null, 'multiple') ? 4 : 1;
	}
}
