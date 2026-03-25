import HTMLElement from '../html-element/HTMLElement.js';
import NodeTypeEnum from '../node/NodeTypeEnum.js';
import * as PropertySymbol from '../../PropertySymbol.js';

/**
 * HTMLTitleElement
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/HTMLTitleElement
 */
export default class HTMLTitleElement extends HTMLElement {
	/**
	 * Returns text.
	 *
	 * @returns Text.
	 */
	public get text(): string {
		let text = '';
		for (const child of this[PropertySymbol.nodeArray]) {
			if (child[PropertySymbol.nodeType] === NodeTypeEnum.textNode) {
				text += child.textContent;
			}
		}
		return text;
	}

	/**
	 * Sets text.
	 *
	 * @param text Text.
	 */
	public set text(text: string) {
		this.textContent = text;
	}

	/**
	 * @override
	 */
	public override get innerHTML(): string {
		return this.getHTML();
	}

	/**
	 * @override
	 */
	public override set innerHTML(html: string) {
		this.textContent = html;
	}
}
