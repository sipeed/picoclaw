import HTMLCollection from '../element/HTMLCollection.js';
import HTMLElement from '../html-element/HTMLElement.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import QuerySelector from '../../query-selector/QuerySelector.js';
import HTMLTableCaptionElement from '../html-table-caption-element/HTMLTableCaptionElement.js';
import HTMLTableSectionElement from '../html-table-section-element/HTMLTableSectionElement.js';
import DOMExceptionNameEnum from '../../exception/DOMExceptionNameEnum.js';
/**
 * HTMLTableElement
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/HTMLTableElement
 */
export default class HTMLTableElement extends HTMLElement {
    [PropertySymbol.rows] = null;
    [PropertySymbol.tBodies] = null;
    /**
     * Returns caption.
     *
     * @returns Caption.
     */
    get caption() {
        return QuerySelector.querySelector(this, 'caption');
    }
    /**
     * Sets caption.
     *
     * @param caption Caption.
     */
    set caption(caption) {
        if (caption) {
            if (!(caption instanceof HTMLTableCaptionElement)) {
                throw new this[PropertySymbol.window].TypeError("Failed to set the 'caption' property on 'HTMLTableElement': Failed to convert value to 'HTMLTableCaptionElement'.");
            }
            this.caption?.remove();
            this.insertBefore(caption, this.firstChild);
        }
        else {
            this.caption?.remove();
        }
    }
    /**
     * Returns table section element.
     *
     * @returns Table section element.
     */
    get tHead() {
        return QuerySelector.querySelector(this, 'thead');
    }
    /**
     * Sets table section element.
     *
     * @param tHead Table section element.
     */
    set tHead(tHead) {
        if (tHead) {
            if (!(tHead instanceof HTMLTableSectionElement)) {
                throw new this[PropertySymbol.window].TypeError("Failed to set the 'tHead' property on 'HTMLTableElement': Failed to convert value to 'HTMLTableSectionElement'.");
            }
            this.tHead?.remove();
            // It should be inserted in the tree immediately before the first element that is neither a <caption>, nor a <colgroup>
            let found = false;
            for (const child of this[PropertySymbol.elementArray]) {
                if (child[PropertySymbol.tagName] !== 'CAPTION' &&
                    child[PropertySymbol.tagName] !== 'COLGROUP') {
                    this.insertBefore(tHead, child);
                    found = true;
                    break;
                }
            }
            // Add as last child if there is no such element
            if (!found) {
                this.appendChild(tHead);
            }
        }
        else {
            this.tHead?.remove();
        }
    }
    /**
     * Returns table section element.
     *
     * @returns Table section element.
     */
    get tFoot() {
        return QuerySelector.querySelector(this, 'tfoot');
    }
    /**
     * Sets table section element.
     *
     * @param tFoot Table section element.
     */
    set tFoot(tFoot) {
        if (tFoot) {
            if (!(tFoot instanceof HTMLTableSectionElement)) {
                throw new this[PropertySymbol.window].TypeError("Failed to set the 'tFoot' property on 'HTMLTableElement': Failed to convert value to 'HTMLTableSectionElement'.");
            }
            this.tFoot?.remove();
            // It should be inserted in the tree immediately before the first element that is neither a <caption>, <colgroup>, nor a <thead>
            let found = false;
            for (const child of this[PropertySymbol.elementArray]) {
                if (child[PropertySymbol.tagName] !== 'CAPTION' &&
                    child[PropertySymbol.tagName] !== 'COLGROUP' &&
                    child[PropertySymbol.tagName] !== 'THEAD') {
                    this.insertBefore(tFoot, child);
                    found = true;
                    break;
                }
            }
            // Add as last child if there is no such element
            if (!found) {
                this.appendChild(tFoot);
            }
        }
        else {
            this.tFoot?.remove();
        }
    }
    /**
     * Returns rows.
     *
     * @returns Rows.
     */
    get rows() {
        if (!this[PropertySymbol.rows]) {
            this[PropertySymbol.rows] = new HTMLCollection(PropertySymbol.illegalConstructor, () => QuerySelector.querySelectorAll(this, 'tr')[PropertySymbol.items]);
        }
        return this[PropertySymbol.rows];
    }
    /**
     * Returns bodies.
     *
     * @returns Bodies.
     */
    get tBodies() {
        if (!this[PropertySymbol.tBodies]) {
            this[PropertySymbol.tBodies] = new HTMLCollection(PropertySymbol.illegalConstructor, () => (QuerySelector.querySelectorAll(this, 'tbody')[PropertySymbol.items]));
        }
        return this[PropertySymbol.tBodies];
    }
    /**
     * Returns an HTMLTableSectionElement representing the first <thead> that is a child of the element. If none is found, a new one is created and inserted in the tree immediately before the first element that is neither a <caption>, nor a <colgroup>, or as the last child if there is no such element.
     *
     * @returns Table section element.
     */
    createTHead() {
        const existingTHead = this.tHead;
        if (existingTHead) {
            return existingTHead;
        }
        const tHead = this[PropertySymbol.ownerDocument].createElement('thead');
        this.tHead = tHead;
        return tHead;
    }
    /**
     * Removes the first <thead> that is a child of the element.
     */
    deleteTHead() {
        this.tHead = null;
    }
    /**
     * Returns an HTMLTableSectionElement representing the first <tfoot> that is a child of the element. If none is found, a new one is created and inserted in the tree as the last child.
     *
     * @returns Table section element.
     */
    createTFoot() {
        const existingTFoot = this.tFoot;
        if (existingTFoot) {
            return existingTFoot;
        }
        const tFoot = this[PropertySymbol.ownerDocument].createElement('tfoot');
        this.tFoot = tFoot;
        return tFoot;
    }
    /**
     * Removes the first <tfoot> that is a child of the element.
     */
    deleteTFoot() {
        this.tFoot = null;
    }
    /**
     * Returns a HTMLTableSectionElement representing a new <tbody> that is a child of the element. It is inserted in the tree after the last element that is a <tbody>, or as the last child if there is no such element.
     *
     * @returns Table section element.
     */
    createTBody() {
        const tBodies = QuerySelector.querySelectorAll(this, 'tbody')[PropertySymbol.items];
        const tBody = this[PropertySymbol.ownerDocument].createElement('tbody');
        if (tBodies.length > 0) {
            const lastTBody = tBodies[tBodies.length - 1];
            lastTBody.parentNode.insertBefore(tBody, lastTBody.nextSibling);
            return tBody;
        }
        this.appendChild(tBody);
        return tBody;
    }
    /**
     * Returns an HTMLTableCaptionElement representing the first <caption> that is a child of the element. If none is found, a new one is created and inserted in the tree as the first child of the <table> element.
     */
    createCaption() {
        const existingCaption = this.caption;
        if (existingCaption) {
            return existingCaption;
        }
        const caption = this[PropertySymbol.ownerDocument].createElement('caption');
        this.caption = caption;
        return caption;
    }
    /**
     * Removes the first <caption> that is a child of the element.
     */
    deleteCaption() {
        this.caption = null;
    }
    /**
     * Returns an HTMLTableRowElement representing a new row of the table. It inserts it in the rows collection immediately before the <tr> element at the given index position. If necessary a <tbody> is created. If the index is -1, the new row is appended to the collection. If the index is smaller than -1 or greater than the number of rows in the collection, a DOMException with the value IndexSizeError is raised.
     *
     * @param [index] Index.
     * @returns Row.
     */
    insertRow(index = -1) {
        if (typeof index !== 'number') {
            index = -1;
        }
        const rows = QuerySelector.querySelectorAll(this, 'tr')[PropertySymbol.items];
        if (index < -1) {
            throw new this[PropertySymbol.window].DOMException(`Failed to execute 'insertRow' on 'HTMLTableElement': The index provided (${index}) is less than -1.`, DOMExceptionNameEnum.indexSizeError);
        }
        if (index > rows.length) {
            throw new this[PropertySymbol.window].DOMException(`Failed to execute 'insertRow' on 'HTMLTableElement': The index provided (${index}) is greater than the number of rows (${rows.length}).`, DOMExceptionNameEnum.indexSizeError);
        }
        const row = this[PropertySymbol.ownerDocument].createElement('tr');
        if (index === -1 || index === rows.length) {
            const tbody = QuerySelector.querySelector(this, 'tbody');
            if (tbody) {
                tbody.appendChild(row);
            }
            else {
                const tbody = this[PropertySymbol.ownerDocument].createElement('tbody');
                tbody.appendChild(row);
                this.appendChild(tbody);
            }
            return row;
        }
        rows[index].parentNode.insertBefore(row, rows[index]);
        return row;
    }
    /**
     * Removes the row corresponding to the index given in parameter. If the index value is -1 the last row is removed; if it is smaller than -1 or greater than the amount of rows in the collection, a DOMException with the value IndexSizeError is raised.
     *
     * @param index Index.
     */
    deleteRow(index) {
        if (arguments.length === 0) {
            throw new this[PropertySymbol.window].TypeError("Failed to execute 'deleteRow' on 'HTMLTableElement': 1 argument required, but only 0 present.");
        }
        if (typeof index !== 'number') {
            index = -1;
        }
        if (index < -1) {
            throw new this[PropertySymbol.window].DOMException(`Failed to execute 'deleteRow' on 'HTMLTableElement': The index provided (${index}) is less than -1.`, DOMExceptionNameEnum.indexSizeError);
        }
        const rows = QuerySelector.querySelectorAll(this, 'tr')[PropertySymbol.items];
        if (index >= rows.length) {
            throw new this[PropertySymbol.window].DOMException(`Failed to execute 'deleteRow' on 'HTMLTableElement': The index provided (${index}) is greater than the number of rows in the table (${rows.length}).`, DOMExceptionNameEnum.indexSizeError);
        }
        if (index === -1) {
            index = rows.length - 1;
        }
        rows[index].remove();
    }
}
//# sourceMappingURL=HTMLTableElement.js.map