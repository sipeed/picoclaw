import HTMLCollection from '../element/HTMLCollection.js';
import HTMLElement from '../html-element/HTMLElement.js';
import HTMLTableCellElement from '../html-table-cell-element/HTMLTableCellElement.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import QuerySelector from '../../query-selector/QuerySelector.js';
import HTMLTableSectionElement from '../html-table-section-element/HTMLTableSectionElement.js';
import DOMExceptionNameEnum from '../../exception/DOMExceptionNameEnum.js';

/**
 * HTMLTableRowElement
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/HTMLTableRowElement
 */
export default class HTMLTableRowElement extends HTMLElement {
	public [PropertySymbol.cells]: HTMLCollection<HTMLTableCellElement> | null = null;

	/**
	 * Returns cells.
	 *
	 * @returns Cells.
	 */
	public get cells(): HTMLCollection<HTMLTableCellElement> {
		if (!this[PropertySymbol.cells]) {
			this[PropertySymbol.cells] = new HTMLCollection<HTMLTableCellElement>(
				PropertySymbol.illegalConstructor,
				() =>
					<HTMLTableCellElement[]>(
						QuerySelector.querySelectorAll(this, 'td,th')[PropertySymbol.items]
					)
			);
		}
		return this[PropertySymbol.cells];
	}

	/**
	 * Returns a number that gives the logical position of the row within the entire table. If the row is not part of a table, returns -1.
	 *
	 * @returns Row index.
	 */
	public get rowIndex(): number {
		let parent = this.parentNode;
		while (parent) {
			if (parent[PropertySymbol.tagName] === 'TABLE') {
				const rows = QuerySelector.querySelectorAll(<HTMLElement>parent, 'tr')[
					PropertySymbol.items
				];
				return rows.indexOf(this);
			}
			parent = parent.parentNode;
		}
		return -1;
	}

	/**
	 * Returns a number that gives the logical position of the row within the table section it belongs to. If the row is not part of a section, returns -1.
	 */
	public get sectionRowIndex(): number {
		let parent = this.parentNode;
		while (parent) {
			if (parent instanceof HTMLTableSectionElement) {
				const rows = QuerySelector.querySelectorAll(<HTMLElement>parent, 'tr')[
					PropertySymbol.items
				];
				return rows.indexOf(this);
			}
			parent = parent.parentNode;
		}
		return -1;
	}

	/**
	 * Returns an HTMLTableCellElement representing a new cell of the row. The cell is inserted in the collection of cells immediately before the given index position in the row. If index is -1, the new cell is appended to the collection. If index is less than -1 or greater than the number of cells in the collection, a DOMException with the value IndexSizeError is raised.
	 *
	 * @param [index] Index.
	 * @returns Cell.
	 */
	public insertCell(index: number = -1): HTMLTableCellElement {
		if (typeof index !== 'number') {
			index = -1;
		}

		const cells = QuerySelector.querySelectorAll(this, 'td,th')[PropertySymbol.items];

		if (index < -1) {
			throw new this[PropertySymbol.window].DOMException(
				`Failed to execute 'insertCell' on 'HTMLTableRowElement': The index provided (${index}) is less than -1.`,
				DOMExceptionNameEnum.indexSizeError
			);
		}

		if (index > cells.length) {
			throw new this[PropertySymbol.window].DOMException(
				`Failed to execute 'insertCell' on 'HTMLTableRowElement': The index provided (${index}) is greater than the number of cells (${cells.length}).`,
				DOMExceptionNameEnum.indexSizeError
			);
		}

		const cell = this[PropertySymbol.ownerDocument].createElement('td');

		if (index === -1 || index === cells.length) {
			this.appendChild(cell);
			return cell;
		}

		cells[index].parentNode.insertBefore(cell, cells[index]);

		return cell;
	}

	/**
	 * Removes the cell corresponding to index. If index is -1, the last cell of the row is removed. If index is less than -1 or greater than the amount of cells in the collection, a DOMException with the value IndexSizeError is raised.
	 *
	 * @param index Index.
	 */
	public deleteCell(index: number): void {
		if (arguments.length === 0) {
			throw new this[PropertySymbol.window].TypeError(
				"Failed to execute 'deleteCell' on 'HTMLTableRowElement': 1 argument required, but only 0 present."
			);
		}

		if (typeof index !== 'number') {
			index = -1;
		}

		if (index < -1) {
			throw new this[PropertySymbol.window].DOMException(
				`Failed to execute 'deleteCell' on 'HTMLTableRowElement': The index provided (${index}) is less than -1.`,
				DOMExceptionNameEnum.indexSizeError
			);
		}

		const cells = QuerySelector.querySelectorAll(this, 'td,th')[PropertySymbol.items];

		if (index >= cells.length) {
			throw new this[PropertySymbol.window].DOMException(
				`Failed to execute 'deleteCell' on 'HTMLTableRowElement': The index provided (${index}) is greater than the number of cells in the row (${cells.length}).`,
				DOMExceptionNameEnum.indexSizeError
			);
		}

		if (index === -1) {
			index = cells.length - 1;
		}

		cells[index].remove();
	}
}
