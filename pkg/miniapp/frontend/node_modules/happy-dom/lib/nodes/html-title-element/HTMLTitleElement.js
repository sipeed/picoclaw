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
    get text() {
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
    set text(text) {
        this.textContent = text;
    }
    /**
     * @override
     */
    get innerHTML() {
        return this.getHTML();
    }
    /**
     * @override
     */
    set innerHTML(html) {
        this.textContent = html;
    }
}
//# sourceMappingURL=HTMLTitleElement.js.map