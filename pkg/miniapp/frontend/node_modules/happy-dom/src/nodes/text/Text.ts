import * as PropertySymbol from '../../PropertySymbol.js';
import CharacterData from '../character-data/CharacterData.js';
import DOMExceptionNameEnum from '../../exception/DOMExceptionNameEnum.js';
import HTMLTextAreaElement from '../html-text-area-element/HTMLTextAreaElement.js';
import NodeTypeEnum from '../node/NodeTypeEnum.js';
import HTMLStyleElement from '../html-style-element/HTMLStyleElement.js';

/**
 * Text node.
 */
export default class Text extends CharacterData {
	public declare cloneNode: (deep?: boolean) => Text;
	public override [PropertySymbol.nodeType] = NodeTypeEnum.textNode;
	public override [PropertySymbol.textAreaNode]: HTMLTextAreaElement | null = null;
	public override [PropertySymbol.styleNode]: HTMLStyleElement | null = null;

	/**
	 * Node name.
	 *
	 * @returns Node name.
	 */
	public get nodeName(): string {
		return '#text';
	}

	/**
	 * @override
	 */
	public override get data(): string {
		return this[PropertySymbol.data];
	}

	/**
	 * @override
	 */
	public override set data(data: string) {
		super.data = data;

		if (this[PropertySymbol.textAreaNode]) {
			(<HTMLTextAreaElement>this[PropertySymbol.textAreaNode])[PropertySymbol.resetSelection]();
		}

		if (this[PropertySymbol.styleNode]) {
			this[PropertySymbol.styleNode][PropertySymbol.updateSheet]();
		}
	}

	/**
	 * Breaks the Text node into two nodes at the specified offset, keeping both nodes in the tree as siblings.
	 *
	 * @see https://dom.spec.whatwg.org/#dom-text-splittext
	 * @param offset Offset.
	 * @returns New text node.
	 */
	public splitText(offset: number): Text {
		const length = this[PropertySymbol.data].length;

		if (offset < 0 || offset > length) {
			throw new this[PropertySymbol.window].DOMException(
				'The index is not in the allowed range.',
				DOMExceptionNameEnum.indexSizeError
			);
		}

		const count = length - offset;
		const newData = this.substringData(offset, count);
		const newNode = <Text>this[PropertySymbol.ownerDocument].createTextNode(newData);

		if (this[PropertySymbol.parentNode] !== null) {
			this[PropertySymbol.parentNode].insertBefore(newNode, this.nextSibling);
		}

		this.replaceData(offset, count, '');

		return newNode;
	}

	/**
	 * Converts to string.
	 *
	 * @returns String.
	 */
	public toString(): string {
		return '[object Text]';
	}

	/**
	 * @override
	 */
	public override [PropertySymbol.cloneNode](deep = false): Text {
		return <Text>super[PropertySymbol.cloneNode](deep);
	}

	/**
	 * @override
	 */
	public override [PropertySymbol.connectedToNode](): void {
		super[PropertySymbol.connectedToNode]();

		if (this[PropertySymbol.textAreaNode]) {
			(<HTMLTextAreaElement>this[PropertySymbol.textAreaNode])[PropertySymbol.resetSelection]();
		}

		if (this[PropertySymbol.styleNode] && this[PropertySymbol.data]) {
			this[PropertySymbol.styleNode][PropertySymbol.updateSheet]();
		}
	}

	/**
	 * @override
	 */
	public override [PropertySymbol.disconnectedFromNode](): void {
		if (this[PropertySymbol.textAreaNode]) {
			(<HTMLTextAreaElement>this[PropertySymbol.textAreaNode])[PropertySymbol.resetSelection]();
		}

		if (this[PropertySymbol.styleNode] && this[PropertySymbol.data]) {
			this[PropertySymbol.styleNode][PropertySymbol.updateSheet]();
		}

		super[PropertySymbol.disconnectedFromNode]();
	}
}
