import HTMLCollection from '../element/HTMLCollection.js';
import HTMLElement from '../html-element/HTMLElement.js';
import HTMLTableCellElement from '../html-table-cell-element/HTMLTableCellElement.js';
import * as PropertySymbol from '../../PropertySymbol.js';
/**
 * HTMLTableRowElement
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/HTMLTableRowElement
 */
export default class HTMLTableRowElement extends HTMLElement {
    [PropertySymbol.cells]: HTMLCollection<HTMLTableCellElement> | null;
    /**
     * Returns cells.
     *
     * @returns Cells.
     */
    get cells(): HTMLCollection<HTMLTableCellElement>;
    /**
     * Returns a number that gives the logical position of the row within the entire table. If the row is not part of a table, returns -1.
     *
     * @returns Row index.
     */
    get rowIndex(): number;
    /**
     * Returns a number that gives the logical position of the row within the table section it belongs to. If the row is not part of a section, returns -1.
     */
    get sectionRowIndex(): number;
    /**
     * Returns an HTMLTableCellElement representing a new cell of the row. The cell is inserted in the collection of cells immediately before the given index position in the row. If index is -1, the new cell is appended to the collection. If index is less than -1 or greater than the number of cells in the collection, a DOMException with the value IndexSizeError is raised.
     *
     * @param [index] Index.
     * @returns Cell.
     */
    insertCell(index?: number): HTMLTableCellElement;
    /**
     * Removes the cell corresponding to index. If index is -1, the last cell of the row is removed. If index is less than -1 or greater than the amount of cells in the collection, a DOMException with the value IndexSizeError is raised.
     *
     * @param index Index.
     */
    deleteCell(index: number): void;
}
//# sourceMappingURL=HTMLTableRowElement.d.ts.map