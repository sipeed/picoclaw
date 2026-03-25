import HTMLElement from '../html-element/HTMLElement.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import HTMLSerializer from '../../html-serializer/HTMLSerializer.js';
import HTMLParser from '../../html-parser/HTMLParser.js';
/**
 * HTML Template Element.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/HTMLTemplateElement.
 */
export default class HTMLTemplateElement extends HTMLElement {
    // Internal properties
    [PropertySymbol.content] = this[PropertySymbol.ownerDocument].createDocumentFragment();
    /**
     * Returns content.
     *
     * @returns Content.
     */
    get content() {
        return this[PropertySymbol.content];
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
        const content = this[PropertySymbol.content];
        const childNodes = content[PropertySymbol.nodeArray];
        while (childNodes.length) {
            content.removeChild(childNodes[0]);
        }
        new HTMLParser(this[PropertySymbol.window]).parse(html, this[PropertySymbol.content]);
    }
    /**
     * @override
     */
    get firstChild() {
        return this[PropertySymbol.content].firstChild;
    }
    /**
     * @override
     */
    get lastChild() {
        return this[PropertySymbol.content].lastChild;
    }
    /**
     * @deprecated
     * @override
     */
    getInnerHTML(_options) {
        const serializer = new HTMLSerializer();
        // Options should be ignored as shadow roots should not be serialized for HTMLTemplateElement.
        const content = this[PropertySymbol.content];
        let html = '';
        for (const node of content[PropertySymbol.nodeArray]) {
            html += serializer.serializeToString(node);
        }
        return html;
    }
    /**
     * @override
     */
    getHTML(_options) {
        const serializer = new HTMLSerializer();
        // Options should be ignored as shadow roots should not be serialized for HTMLTemplateElement.
        const content = this[PropertySymbol.content];
        let html = '';
        for (const node of content[PropertySymbol.nodeArray]) {
            html += serializer.serializeToString(node);
        }
        return html;
    }
    /**
     * @override
     */
    [PropertySymbol.appendChild](node, disableValidations = false) {
        return this[PropertySymbol.content][PropertySymbol.appendChild](node, disableValidations);
    }
    /**
     * @override
     */
    [PropertySymbol.removeChild](node) {
        return this[PropertySymbol.content][PropertySymbol.removeChild](node);
    }
    /**
     * @override
     */
    [PropertySymbol.insertBefore](newNode, referenceNode, disableValidations = false) {
        return this[PropertySymbol.content][PropertySymbol.insertBefore](newNode, referenceNode, disableValidations);
    }
    /**
     * @override
     */
    [PropertySymbol.replaceChild](newChild, oldChild) {
        return this[PropertySymbol.content][PropertySymbol.replaceChild](newChild, oldChild);
    }
    /**
     * @override
     */
    [PropertySymbol.cloneNode](deep = false) {
        const clone = super[PropertySymbol.cloneNode](deep);
        clone[PropertySymbol.content] = this[PropertySymbol.content].cloneNode(deep);
        return clone;
    }
}
//# sourceMappingURL=HTMLTemplateElement.js.map