import Node from '../node/Node.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import CharacterDataUtility from './CharacterDataUtility.js';
import Element from '../element/Element.js';
import NonDocumentChildNodeUtility from '../child-node/NonDocumentChildNodeUtility.js';
import ChildNodeUtility from '../child-node/ChildNodeUtility.js';
import MutationRecord from '../../mutation-observer/MutationRecord.js';
import MutationTypeEnum from '../../mutation-observer/MutationTypeEnum.js';
import IChildNode from '../child-node/IChildNode.js';
import INonDocumentTypeChildNode from '../child-node/INonDocumentTypeChildNode.js';

/**
 * Character data base class.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/CharacterData.
 */
export default abstract class CharacterData
	extends Node
	implements IChildNode, INonDocumentTypeChildNode
{
	public [PropertySymbol.data] = '';
	public declare cloneNode: (deep?: boolean) => CharacterData;

	/**
	 * Constructor.
	 *
	 * @param [data] Data.
	 */
	constructor(data?: string) {
		super();

		this[PropertySymbol.data] = data !== undefined ? String(data) : '';
	}

	/**
	 * Returns text content.
	 *
	 * @returns Text content.
	 */
	public get length(): number {
		return this[PropertySymbol.data].length;
	}

	/**
	 * Returns text content.
	 *
	 * @returns Text content.
	 */
	public get data(): string {
		return this[PropertySymbol.data];
	}

	/**
	 * Sets text content.
	 *
	 * @param textContent Text content.
	 */
	public set data(data: string) {
		const oldValue = this[PropertySymbol.data];
		this[PropertySymbol.data] = String(data);

		this[PropertySymbol.reportMutation](
			new MutationRecord({
				target: this,
				type: MutationTypeEnum.characterData,
				oldValue
			})
		);
	}

	/**
	 * Returns text content.
	 *
	 * @returns Text content.
	 */
	public get textContent(): string {
		return this[PropertySymbol.data];
	}

	/**
	 * Sets text content.
	 *
	 * @param textContent Text content.
	 */
	public set textContent(textContent: string) {
		this.data = textContent;
	}

	/**
	 * Returns node value.
	 *
	 * @returns Node value.
	 */
	public get nodeValue(): string {
		return this[PropertySymbol.data];
	}

	/**
	 * Sets node value.
	 *
	 * @param nodeValue Node value.
	 */
	public set nodeValue(nodeValue: string) {
		this.textContent = nodeValue;
	}

	/**
	 * Previous element sibling.
	 *
	 * @returns Element.
	 */
	public get previousElementSibling(): Element {
		return NonDocumentChildNodeUtility.previousElementSibling(this);
	}

	/**
	 * Next element sibling.
	 *
	 * @returns Element.
	 */
	public get nextElementSibling(): Element {
		return NonDocumentChildNodeUtility.nextElementSibling(this);
	}

	/**
	 * Appends the given DOMString to the CharacterData.data string; when this method returns, data contains the concatenated DOMString.
	 *
	 * @param data Data.
	 */
	public appendData(data: string): void {
		CharacterDataUtility.appendData(this, data);
	}

	/**
	 * Removes the specified amount of characters, starting at the specified offset, from the CharacterData.data string; when this method returns, data contains the shortened DOMString.
	 *
	 * @param offset Offset.
	 * @param count Count.
	 */
	public deleteData(offset: number, count: number): void {
		CharacterDataUtility.deleteData(this, offset, count);
	}

	/**
	 * Inserts the specified characters, at the specified offset, in the CharacterData.data string; when this method returns, data contains the modified DOMString.
	 *
	 * @param offset Offset.
	 * @param data Data.
	 */
	public insertData(offset: number, data: string): void {
		CharacterDataUtility.insertData(this, offset, data);
	}

	/**
	 * Replaces the specified amount of characters, starting at the specified offset, with the specified DOMString; when this method returns, data contains the modified DOMString.
	 *
	 * @param offset Offset.
	 * @param count Count.
	 * @param data Data.
	 */
	public replaceData(offset: number, count: number, data: string): void {
		CharacterDataUtility.replaceData(this, offset, count, data);
	}

	/**
	 * Returns a DOMString containing the part of CharacterData.data of the specified length and starting at the specified offset.
	 *
	 * @param offset Offset.
	 * @param count Count.
	 */
	public substringData(offset: number, count: number): string {
		return CharacterDataUtility.substringData(this, offset, count);
	}

	/**
	 * Removes the object from its parent children list.
	 */
	public remove(): void {
		ChildNodeUtility.remove(this);
	}

	/**
	 * The Node.replaceWith() method replaces this Node in the children list of its parent with a set of Node or DOMString objects.
	 *
	 * @param nodes List of Node or DOMString.
	 */
	public replaceWith(...nodes: (Node | string)[]): void {
		ChildNodeUtility.replaceWith(this, ...nodes);
	}

	/**
	 * Inserts a set of Node or DOMString objects in the children list of this ChildNode's parent, just before this ChildNode. DOMString objects are inserted as equivalent Text nodes.
	 *
	 * @param nodes List of Node or DOMString.
	 */
	public before(...nodes: (string | Node)[]): void {
		ChildNodeUtility.before(this, ...nodes);
	}

	/**
	 * Inserts a set of Node or DOMString objects in the children list of this ChildNode's parent, just after this ChildNode. DOMString objects are inserted as equivalent Text nodes.
	 *
	 * @param nodes List of Node or DOMString.
	 */
	public after(...nodes: (string | Node)[]): void {
		ChildNodeUtility.after(this, ...nodes);
	}

	/**
	 * @override
	 */
	public override [PropertySymbol.cloneNode](deep = false): CharacterData {
		const clone = <CharacterData>super[PropertySymbol.cloneNode](deep);
		clone[PropertySymbol.data] = this[PropertySymbol.data];
		return clone;
	}
}
