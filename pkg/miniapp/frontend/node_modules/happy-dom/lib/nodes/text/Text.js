import * as PropertySymbol from '../../PropertySymbol.js';
import CharacterData from '../character-data/CharacterData.js';
import DOMExceptionNameEnum from '../../exception/DOMExceptionNameEnum.js';
import NodeTypeEnum from '../node/NodeTypeEnum.js';
/**
 * Text node.
 */
export default class Text extends CharacterData {
    [PropertySymbol.nodeType] = NodeTypeEnum.textNode;
    [PropertySymbol.textAreaNode] = null;
    [PropertySymbol.styleNode] = null;
    /**
     * Node name.
     *
     * @returns Node name.
     */
    get nodeName() {
        return '#text';
    }
    /**
     * @override
     */
    get data() {
        return this[PropertySymbol.data];
    }
    /**
     * @override
     */
    set data(data) {
        super.data = data;
        if (this[PropertySymbol.textAreaNode]) {
            this[PropertySymbol.textAreaNode][PropertySymbol.resetSelection]();
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
    splitText(offset) {
        const length = this[PropertySymbol.data].length;
        if (offset < 0 || offset > length) {
            throw new this[PropertySymbol.window].DOMException('The index is not in the allowed range.', DOMExceptionNameEnum.indexSizeError);
        }
        const count = length - offset;
        const newData = this.substringData(offset, count);
        const newNode = this[PropertySymbol.ownerDocument].createTextNode(newData);
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
    toString() {
        return '[object Text]';
    }
    /**
     * @override
     */
    [PropertySymbol.cloneNode](deep = false) {
        return super[PropertySymbol.cloneNode](deep);
    }
    /**
     * @override
     */
    [PropertySymbol.connectedToNode]() {
        super[PropertySymbol.connectedToNode]();
        if (this[PropertySymbol.textAreaNode]) {
            this[PropertySymbol.textAreaNode][PropertySymbol.resetSelection]();
        }
        if (this[PropertySymbol.styleNode] && this[PropertySymbol.data]) {
            this[PropertySymbol.styleNode][PropertySymbol.updateSheet]();
        }
    }
    /**
     * @override
     */
    [PropertySymbol.disconnectedFromNode]() {
        if (this[PropertySymbol.textAreaNode]) {
            this[PropertySymbol.textAreaNode][PropertySymbol.resetSelection]();
        }
        if (this[PropertySymbol.styleNode] && this[PropertySymbol.data]) {
            this[PropertySymbol.styleNode][PropertySymbol.updateSheet]();
        }
        super[PropertySymbol.disconnectedFromNode]();
    }
}
//# sourceMappingURL=Text.js.map