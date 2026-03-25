import HTMLElement from '../html-element/HTMLElement.cjs';
import HTMLTableRowElement from '../html-table-row-element/HTMLTableRowElement.cjs';
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
    insertRow(index?: number): HTMLTableRowElement;
    /**
     * Removes the row corresponding to the index given in parameter. If the index value is -1 the last row is removed; if it is smaller than -1 or greater than the amount of rows in the collection, a DOMException with the value IndexSizeError is raised.
     *
     * @param index Index.
     */
    deleteRow(index: number): void;
}
//# sourceMappingURL=HTMLTableSectionElement.d.ts.map