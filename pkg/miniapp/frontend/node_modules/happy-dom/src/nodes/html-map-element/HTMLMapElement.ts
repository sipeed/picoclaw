import HTMLCollection from '../element/HTMLCollection.js';
import HTMLElement from '../html-element/HTMLElement.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import HTMLAreaElement from '../html-area-element/HTMLAreaElement.js';
import ParentNodeUtility from '../parent-node/ParentNodeUtility.js';

/**
 * HTMLMapElement
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/HTMLMapElement
 */
export default class HTMLMapElement extends HTMLElement {
	public [PropertySymbol.areas]: HTMLCollection<HTMLAreaElement> | null = null;

	/**
	 * Returns areas.
	 *
	 * @returns Areas.
	 */
	public get areas(): HTMLCollection<HTMLAreaElement> {
		if (!this[PropertySymbol.areas]) {
			this[PropertySymbol.areas] = ParentNodeUtility.getElementsByTagName<HTMLAreaElement>(
				this,
				'area'
			);
		}
		return this[PropertySymbol.areas];
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
}
