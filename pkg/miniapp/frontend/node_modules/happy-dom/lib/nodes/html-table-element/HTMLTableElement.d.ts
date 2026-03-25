import HTMLCollection from '../element/HTMLCollection.js';
import HTMLElement from '../html-element/HTMLElement.js';
import HTMLTableRowElement from '../html-table-row-element/HTMLTableRowElement.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import HTMLTableCaptionElement from '../html-table-caption-element/HTMLTableCaptionElement.js';
import HTMLTableSectionElement from '../html-table-section-element/HTMLTableSectionElement.js';
/**
 * HTMLTableElement
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/HTMLTableElement
 */
export default class HTMLTableElement extends HTMLElement {
    [PropertySymbol.rows]: HTMLCollection<HTMLTableRowElement> | null;
    [PropertySymbol.tBodies]: HTMLCollection<HTMLTableSectionElement> | null;
    /**
     * Returns caption.
     *
     * @returns Caption.
     */
    get caption(): HTMLTableCaptionElement | null;
    /**
     * Sets caption.
     *
     * @param caption Caption.
     */
    set caption(caption: HTMLTableCaptionElement | null);
    /**
     * Returns table section element.
     *
     * @returns Table section element.
     */
    get tHead(): HTMLTableSectionElement | null;
    /**
     * Sets table section element.
     *
     * @param tHead Table section element.
     */
    set tHead(tHead: HTMLTableSectionElement | null);
    /**
     * Returns table section element.
     *
     * @returns Table section element.
     */
    get tFoot(): HTMLTableSectionElement | null;
    /**
     * Sets table section element.
     *
     * @param tFoot Table section element.
     */
    set tFoot(tFoot: HTMLTableSectionElement | null);
    /**
     * Returns rows.
     *
     * @returns Rows.
     */
    get rows(): HTMLCollection<HTMLTableRowElement>;
    /**
     * Returns bodies.
     *
     * @returns Bodies.
     */
    get tBodies(): HTMLCollection<HTMLTableSectionElement>;
    /**
     * Returns an HTMLTableSectionElement representing the first <thead> that is a child of the element. If none is found, a new one is created and inserted in the tree immediately before the first element that is neither a <caption>, nor a <colgroup>, or as the last child if there is no such element.
     *
     * @returns Table section element.
     */
    createTHead(): HTMLTableSectionElement;
    /**
     * Removes the first <thead> that is a child of the element.
     */
    deleteTHead(): void;
    /**
     * Returns an HTMLTableSectionElement representing the first <tfoot> that is a child of the element. If none is found, a new one is created and inserted in the tree as the last child.
     *
     * @returns Table section element.
     */
    createTFoot(): HTMLTableSectionElement;
    /**
     * Removes the first <tfoot> that is a child of the element.
     */
    deleteTFoot(): void;
    /**
     * Returns a HTMLTableSectionElement representing a new <tbody> that is a child of the element. It is inserted in the tree after the last element that is a <tbody>, or as the last child if there is no such element.
     *
     * @returns Table section element.
     */
    createTBody(): HTMLTableSectionElement;
    /**
     * Returns an HTMLTableCaptionElement representing the first <caption> that is a child of the element. If none is found, a new one is created and inserted in the tree as the first child of the <table> element.
     */
    createCaption(): HTMLTableCaptionElement;
    /**
     * Removes the first <caption> that is a child of the element.
     */
    deleteCaption(): void;
    /**
     * Returns an HTMLTableRowElement representing a new row of the table. It inserts it in the rows collection immediately before the <tr> element at the given index position. If necessary a <tbody> is created. If the index is -1, the new row is appended to the collection. If the index is smaller than -1 or greater than the number of rows in the collection, a DOMException with the value IndexSizeError is raised.
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
//# sourceMappingURL=HTMLTableElement.d.ts.map