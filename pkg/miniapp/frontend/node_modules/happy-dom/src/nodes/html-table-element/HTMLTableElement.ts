import HTMLCollection from '../element/HTMLCollection.js';
import HTMLElement from '../html-element/HTMLElement.js';
import HTMLTableRowElement from '../html-table-row-element/HTMLTableRowElement.js';
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
	public [PropertySymbol.rows]: HTMLCollection<HTMLTableRowElement> | null = null;
	public [PropertySymbol.tBodies]: HTMLCollection<HTMLTableSectionElement> | null = null;

	/**
	 * Returns caption.
	 *
	 * @returns Caption.
	 */
	public get caption(): HTMLTableCaptionElement | null {
		return QuerySelector.querySelector(this, 'caption');
	}

	/**
	 * Sets caption.
	 *
	 * @param caption Caption.
	 */
	public set caption(caption: HTMLTableCaptionElement | null) {
		if (caption) {
			if (!(caption instanceof HTMLTableCaptionElement)) {
				throw new this[PropertySymbol.window].TypeError(
					"Failed to set the 'caption' property on 'HTMLTableElement': Failed to convert value to 'HTMLTableCaptionElement'."
				);
			}

			this.caption?.remove();
			this.insertBefore(caption, this.firstChild);
		} else {
			this.caption?.remove();
		}
	}

	/**
	 * Returns table section element.
	 *
	 * @returns Table section element.
	 */
	public get tHead(): HTMLTableSectionElement | null {
		return QuerySelector.querySelector(this, 'thead');
	}

	/**
	 * Sets table section element.
	 *
	 * @param tHead Table section element.
	 */
	public set tHead(tHead: HTMLTableSectionElement | null) {
		if (tHead) {
			if (!(tHead instanceof HTMLTableSectionElement)) {
				throw new this[PropertySymbol.window].TypeError(
					"Failed to set the 'tHead' property on 'HTMLTableElement': Failed to convert value to 'HTMLTableSectionElement'."
				);
			}

			this.tHead?.remove();

			// It should be inserted in the tree immediately before the first element that is neither a <caption>, nor a <colgroup>
			let found = false;
			for (const child of this[PropertySymbol.elementArray]) {
				if (
					child[PropertySymbol.tagName] !== 'CAPTION' &&
					child[PropertySymbol.tagName] !== 'COLGROUP'
				) {
					this.insertBefore(tHead, child);
					found = true;
					break;
				}
			}

			// Add as last child if there is no such element
			if (!found) {
				this.appendChild(tHead);
			}
		} else {
			this.tHead?.remove();
		}
	}

	/**
	 * Returns table section element.
	 *
	 * @returns Table section element.
	 */
	public get tFoot(): HTMLTableSectionElement | null {
		return QuerySelector.querySelector(this, 'tfoot');
	}

	/**
	 * Sets table section element.
	 *
	 * @param tFoot Table section element.
	 */
	public set tFoot(tFoot: HTMLTableSectionElement | null) {
		if (tFoot) {
			if (!(tFoot instanceof HTMLTableSectionElement)) {
				throw new this[PropertySymbol.window].TypeError(
					"Failed to set the 'tFoot' property on 'HTMLTableElement': Failed to convert value to 'HTMLTableSectionElement'."
				);
			}

			this.tFoot?.remove();

			// It should be inserted in the tree immediately before the first element that is neither a <caption>, <colgroup>, nor a <thead>
			let found = false;
			for (const child of this[PropertySymbol.elementArray]) {
				if (
					child[PropertySymbol.tagName] !== 'CAPTION' &&
					child[PropertySymbol.tagName] !== 'COLGROUP' &&
					child[PropertySymbol.tagName] !== 'THEAD'
				) {
					this.insertBefore(tFoot, child);
					found = true;
					break;
				}
			}

			// Add as last child if there is no such element
			if (!found) {
				this.appendChild(tFoot);
			}
		} else {
			this.tFoot?.remove();
		}
	}

	/**
	 * Returns rows.
	 *
	 * @returns Rows.
	 */
	public get rows(): HTMLCollection<HTMLTableRowElement> {
		if (!this[PropertySymbol.rows]) {
			this[PropertySymbol.rows] = new HTMLCollection<HTMLTableRowElement>(
				PropertySymbol.illegalConstructor,
				() =>
					<HTMLTableRowElement[]>QuerySelector.querySelectorAll(this, 'tr')[PropertySymbol.items]
			);
		}
		return this[PropertySymbol.rows];
	}

	/**
	 * Returns bodies.
	 *
	 * @returns Bodies.
	 */
	public get tBodies(): HTMLCollection<HTMLTableSectionElement> {
		if (!this[PropertySymbol.tBodies]) {
			this[PropertySymbol.tBodies] = new HTMLCollection<HTMLTableSectionElement>(
				PropertySymbol.illegalConstructor,
				() =>
					<HTMLTableSectionElement[]>(
						QuerySelector.querySelectorAll(this, 'tbody')[PropertySymbol.items]
					)
			);
		}
		return this[PropertySymbol.tBodies];
	}

	/**
	 * Returns an HTMLTableSectionElement representing the first <thead> that is a child of the element. If none is found, a new one is created and inserted in the tree immediately before the first element that is neither a <caption>, nor a <colgroup>, or as the last child if there is no such element.
	 *
	 * @returns Table section element.
	 */
	public createTHead(): HTMLTableSectionElement {
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
	public deleteTHead(): void {
		this.tHead = null;
	}

	/**
	 * Returns an HTMLTableSectionElement representing the first <tfoot> that is a child of the element. If none is found, a new one is created and inserted in the tree as the last child.
	 *
	 * @returns Table section element.
	 */
	public createTFoot(): HTMLTableSectionElement {
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
	public deleteTFoot(): void {
		this.tFoot = null;
	}

	/**
	 * Returns a HTMLTableSectionElement representing a new <tbody> that is a child of the element. It is inserted in the tree after the last element that is a <tbody>, or as the last child if there is no such element.
	 *
	 * @returns Table section element.
	 */
	public createTBody(): HTMLTableSectionElement {
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
	public createCaption(): HTMLTableCaptionElement {
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
	public deleteCaption(): void {
		this.caption = null;
	}

	/**
	 * Returns an HTMLTableRowElement representing a new row of the table. It inserts it in the rows collection immediately before the <tr> element at the given index position. If necessary a <tbody> is created. If the index is -1, the new row is appended to the collection. If the index is smaller than -1 or greater than the number of rows in the collection, a DOMException with the value IndexSizeError is raised.
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
				`Failed to execute 'insertRow' on 'HTMLTableElement': The index provided (${index}) is less than -1.`,
				DOMExceptionNameEnum.indexSizeError
			);
		}

		if (index > rows.length) {
			throw new this[PropertySymbol.window].DOMException(
				`Failed to execute 'insertRow' on 'HTMLTableElement': The index provided (${index}) is greater than the number of rows (${rows.length}).`,
				DOMExceptionNameEnum.indexSizeError
			);
		}

		const row = this[PropertySymbol.ownerDocument].createElement('tr');

		if (index === -1 || index === rows.length) {
			const tbody = QuerySelector.querySelector(this, 'tbody');
			if (tbody) {
				tbody.appendChild(row);
			} else {
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
	public deleteRow(index: number): void {
		if (arguments.length === 0) {
			throw new this[PropertySymbol.window].TypeError(
				"Failed to execute 'deleteRow' on 'HTMLTableElement': 1 argument required, but only 0 present."
			);
		}

		if (typeof index !== 'number') {
			index = -1;
		}

		if (index < -1) {
			throw new this[PropertySymbol.window].DOMException(
				`Failed to execute 'deleteRow' on 'HTMLTableElement': The index provided (${index}) is less than -1.`,
				DOMExceptionNameEnum.indexSizeError
			);
		}

		const rows = QuerySelector.querySelectorAll(this, 'tr')[PropertySymbol.items];

		if (index >= rows.length) {
			throw new this[PropertySymbol.window].DOMException(
				`Failed to execute 'deleteRow' on 'HTMLTableElement': The index provided (${index}) is greater than the number of rows in the table (${rows.length}).`,
				DOMExceptionNameEnum.indexSizeError
			);
		}

		if (index === -1) {
			index = rows.length - 1;
		}

		rows[index].remove();
	}
}
