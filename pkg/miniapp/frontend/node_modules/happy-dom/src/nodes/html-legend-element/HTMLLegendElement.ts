import HTMLElement from '../html-element/HTMLElement.js';
import HTMLFormElement from '../html-form-element/HTMLFormElement.js';
import HTMLFieldSetElement from '../html-field-set-element/HTMLFieldSetElement.js';
import Node from '../node/Node.js';

/**
 * HTMLLegendElement
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/HTMLLegendElement
 */
export default class HTMLLegendElement extends HTMLElement {
	/**
	 * Returns the parent form element.
	 *
	 * @returns Form.
	 */
	public get form(): HTMLFormElement {
		let parent: Node | null = this;
		while (parent) {
			if (parent instanceof HTMLFieldSetElement) {
				return parent.form;
			}
			parent = parent.parentNode;
		}
		return null;
	}
}
