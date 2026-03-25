import QuerySelector from '../../query-selector/QuerySelector.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import HTMLElement from '../html-element/HTMLElement.js';
import HTMLTableRowElement from '../html-table-row-element/HTMLTableRowElement.js';
import DOMExceptionNameEnum from '../../exception/DOMExceptionNameEnum.js';
/**
 * HTMLTableSectionElement
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/HTMLTableSectionElement
 */
export default class HTMLTableSectionElement extends HTMLElement {
	/**
	 * Returns an HTMLTableRowElement representing a new row of the table. It inserts it in the rows collection immediately before the <tr> element at the given index position. If the index is -1, the new row is appended to the collection. If the index is smaller than -1 or greater than the number of rows in the collection, a DOMException with the value IndexSizeError is raised.
	 *
	 * @param [index] Index.
	 * @returns Row.
	 */
	public insertRow(index: number = -1): HTMLTableRowElement {
		if (typeof index !== 'number') {
			index = -1;
		}

		const rows = QuerySelector.querySelectorAll(this, 'tr')[PropertySymbol.items];

		if (index < -1) {
			throw new this[PropertySymbol.window].DOMException(
				`Failed to execute 'insertRow' on 'HTMLTableSectionElement': The index provided (${index}) is less than -1.`,
				DOMExceptionNameEnum.indexSizeError
			);
		}

		if (index > rows.length) {
			throw new this[PropertySymbol.window].DOMException(
				`Failed to execute 'insertRow' on 'HTMLTableSectionElement': The index provided (${index}) is greater than the number of rows (${rows.length}).`,
				DOMExceptionNameEnum.indexSizeError
			);
		}

		const row = this[PropertySymbol.ownerDocument].createElement('tr');

		if (index === -1 || index === rows.length) {
			this.appendChild(row);
		} else {
			this.insertBefore(row, rows[index]);
		}

		return row;
	}

	/**
	 * Removes the row corresponding to the index given in parameter. If the index value is -1 the last row is removed; if it is smaller than -1 or greater than the amount of rows in the collection, a DOMException with the value IndexSizeError is raised.
	 *
	 * @param index Index.
	 */
	public deleteRow(index: number): void {
		if (arguments.length === 0) {
			throw new this[PropertySymbol.window].TypeError(
				"Failed to execute 'deleteRow' on 'HTMLTableSectionElement': 1 argument required, but only 0 present."
			);
		}

		if (typeof index !== 'number') {
			index = -1;
		}

		if (index < -1) {
			throw new this[PropertySymbol.window].DOMException(
				`Failed to execute 'deleteRow' on 'HTMLTableSectionElement': The index provided (${index}) is less than -1.`,
				DOMExceptionNameEnum.indexSizeError
			);
		}

		const rows = QuerySelector.querySelectorAll(this, 'tr')[PropertySymbol.items];

		if (index >= rows.length) {
			throw new this[PropertySymbol.window].DOMException(
				`Failed to execute 'deleteRow' on 'HTMLTableSectionElement': The index provided (${index}) is greater than the number of rows in the table (${rows.length}).`,
				DOMExceptionNameEnum.indexSizeError
			);
		}

		if (index === -1) {
			index = rows.length - 1;
		}

		rows[index].remove();
	}
}
