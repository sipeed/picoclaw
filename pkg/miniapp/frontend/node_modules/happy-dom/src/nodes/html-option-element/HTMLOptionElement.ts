import * as PropertySymbol from '../../PropertySymbol.js';
import QuerySelector from '../../query-selector/QuerySelector.js';
import Attr from '../attr/Attr.js';
import HTMLElement from '../html-element/HTMLElement.js';
import HTMLFormElement from '../html-form-element/HTMLFormElement.js';
import HTMLSelectElement from '../html-select-element/HTMLSelectElement.js';

/**
 * HTML Option Element.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/HTMLOptionElement.
 */
export default class HTMLOptionElement extends HTMLElement {
	public [PropertySymbol.selectedness] = false;
	public [PropertySymbol.dirtyness] = false;
	public [PropertySymbol.selectNode]: HTMLSelectElement | null = null;

	/**
	 * Returns inner text, which is the rendered appearance of text.
	 *
	 * @returns Inner text.
	 */
	public get text(): string {
		return this.innerText;
	}

	/**
	 * Sets the inner text, which is the rendered appearance of text.
	 *
	 * @param innerText Inner text.
	 */
	public set text(text: string) {
		this.innerText = text;
	}

	/**
	 * Returns index.
	 *
	 * @returns Index.
	 */
	public get index(): number {
		if (!this[PropertySymbol.selectNode]) {
			return 0;
		}
		const options = QuerySelector.querySelectorAll(this[PropertySymbol.selectNode], 'option')[
			PropertySymbol.elements
		];
		return options.indexOf(this);
	}

	/**
	 * Returns the parent form element.
	 *
	 * @returns Form.
	 */
	public get form(): HTMLFormElement {
		return (<HTMLSelectElement>this[PropertySymbol.selectNode])?.form || null;
	}

	/**
	 * Returns selected.
	 *
	 * @returns Selected.
	 */
	public get selected(): boolean {
		return this[PropertySymbol.selectedness];
	}

	/**
	 * Sets selected.
	 *
	 * @param selected Selected.
	 */
	public set selected(selected: boolean) {
		const selectNode = <HTMLSelectElement>this[PropertySymbol.selectNode];

		this[PropertySymbol.dirtyness] = true;
		this[PropertySymbol.selectedness] = Boolean(selected);

		if (selectNode) {
			selectNode[PropertySymbol.updateSelectedness](
				this[PropertySymbol.selectedness] ? this : null
			);
		}
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
	 * Returns value.
	 *
	 * @returns Value.
	 */
	public get value(): string {
		return this.getAttribute('value') ?? this.textContent;
	}

	/**
	 * Sets value.
	 *
	 * @param value Value.
	 */
	public set value(value: string) {
		this.setAttribute('value', value);
	}

	/**
	 * @override
	 */
	public override [PropertySymbol.onSetAttribute](
		attribute: Attr,
		replacedAttribute: Attr | null
	): void {
		super[PropertySymbol.onSetAttribute](attribute, replacedAttribute);
		if (
			!this[PropertySymbol.dirtyness] &&
			attribute[PropertySymbol.name] === 'selected' &&
			replacedAttribute?.[PropertySymbol.value] !== attribute[PropertySymbol.value]
		) {
			const selectNode = <HTMLSelectElement>this[PropertySymbol.selectNode];

			this[PropertySymbol.selectedness] = true;

			if (selectNode) {
				selectNode[PropertySymbol.updateSelectedness](this);
			}
		}
	}

	/**
	 * @override
	 */
	public override [PropertySymbol.onRemoveAttribute](removedAttribute: Attr): void {
		super[PropertySymbol.onRemoveAttribute](removedAttribute);
		if (
			removedAttribute &&
			!this[PropertySymbol.dirtyness] &&
			removedAttribute[PropertySymbol.name] === 'selected'
		) {
			const selectNode = <HTMLSelectElement>this[PropertySymbol.selectNode];

			this[PropertySymbol.selectedness] = false;

			if (selectNode) {
				selectNode[PropertySymbol.updateSelectedness]();
			}
		}
	}

	/**
	 * @override
	 */
	public override [PropertySymbol.connectedToNode](): void {
		super[PropertySymbol.connectedToNode]();

		if (this[PropertySymbol.selectNode]) {
			this[PropertySymbol.selectNode][PropertySymbol.updateSelectedness]();
		}
	}

	/**
	 * @override
	 */
	public override [PropertySymbol.disconnectedFromNode](): void {
		if (this[PropertySymbol.selectNode]) {
			(<HTMLSelectElement>this[PropertySymbol.selectNode])[PropertySymbol.updateSelectedness]();
		}

		super[PropertySymbol.disconnectedFromNode]();
	}
}
