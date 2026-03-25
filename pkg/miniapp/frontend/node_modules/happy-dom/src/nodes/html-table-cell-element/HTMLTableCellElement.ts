import QuerySelector from '../../query-selector/QuerySelector.js';
import HTMLElement from '../html-element/HTMLElement.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import DOMTokenList from '../../dom/DOMTokenList.js';

/**
 * HTMLTableCellElement
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/HTMLTableCellElement
 */
export default class HTMLTableCellElement extends HTMLElement {
	public [PropertySymbol.headers]: DOMTokenList | null = null;

	/**
	 * Returns abbr.
	 *
	 * @returns Abbr.
	 */
	public get abbr(): string {
		return this.getAttribute('abbr') || '';
	}

	/**
	 * Sets abbr.
	 *
	 * @param value Abbr.
	 */
	public set abbr(value: string) {
		this.setAttribute('abbr', value);
	}

	/**
	 * A number representing the cell's position in the cells collection of the <tr> the cell is contained within. If the cell doesn't belong to a <tr>, it returns -1.
	 *
	 * @returns Cell index.
	 */
	public get cellIndex(): number {
		let parent = this.parentNode;
		while (parent) {
			if (parent[PropertySymbol.tagName] === 'TR') {
				const cells = QuerySelector.querySelectorAll(<HTMLElement>parent, 'td,th')[
					PropertySymbol.items
				];
				return cells.indexOf(this);
			}
			parent = parent.parentNode;
		}
		return -1;
	}

	/**
	 * Returns colspan.
	 *
	 * @returns Colspan.
	 */
	public get colSpan(): number {
		const value = Number(this.getAttribute('colspan'));
		return isNaN(value) || value < 1 ? 1 : value;
	}

	/**
	 * Sets colspan.
	 *
	 * @param value Colspan.
	 */
	public set colSpan(value: number) {
		const parsedValue = Number(value);
		this.setAttribute('colspan', isNaN(parsedValue) || parsedValue < 1 ? '1' : String(parsedValue));
	}

	/**
	 * Returns headers.
	 *
	 * @returns headers.
	 */
	public get headers(): string {
		return this.getAttribute('headers') || '';
	}

	/**
	 * Sets headers.
	 *
	 * @param value headers.
	 */
	public set headers(value: string) {
		this.setAttribute('headers', String(value));
		// TODO: implement metadata update if needed.
	}

	/**
	 * Returns rowspan.
	 *
	 * @returns Rowspan.
	 */
	public get rowSpan(): number {
		const value = Number(this.getAttribute('rowspan'));
		return isNaN(value) || value < 1 ? 1 : value;
	}

	/**
	 * Sets rowspan.
	 *
	 * @param value Rowspan.
	 */
	public set rowSpan(value: number) {
		const parsedValue = Number(value);
		this.setAttribute('rowspan', isNaN(parsedValue) || parsedValue < 1 ? '1' : String(parsedValue));
	}

	/**
	 * Returns scope.
	 *
	 * @returns Scope.
	 */
	public get scope(): string {
		return this.getAttribute('scope') || '';
	}

	/**
	 * Sets scope.
	 *
	 * @param value Scope.
	 */
	public set scope(value: string) {
		this.setAttribute('scope', value);
	}
}
