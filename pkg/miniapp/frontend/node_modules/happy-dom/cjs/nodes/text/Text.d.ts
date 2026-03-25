import * as PropertySymbol from '../../PropertySymbol.cjs';
import CharacterData from '../character-data/CharacterData.cjs';
import HTMLTextAreaElement from '../html-text-area-element/HTMLTextAreaElement.cjs';
import NodeTypeEnum from '../node/NodeTypeEnum.cjs';
import HTMLStyleElement from '../html-style-element/HTMLStyleElement.cjs';
/**
 * Text node.
 */
export default class Text extends CharacterData {
    cloneNode: (deep?: boolean) => Text;
    [PropertySymbol.nodeType]: NodeTypeEnum;
    [PropertySymbol.textAreaNode]: HTMLTextAreaElement | null;
    [PropertySymbol.styleNode]: HTMLStyleElement | null;
    /**
     * Node name.
     *
     * @returns Node name.
     */
    get nodeName(): string;
    /**
     * @override
     */
    get data(): string;
    /**
     * @override
     */
    set data(data: string);
    /**
     * Breaks the Text node into two nodes at the specified offset, keeping both nodes in the tree as siblings.
     *
     * @see https://dom.spec.whatwg.org/#dom-text-splittext
     * @param offset Offset.
     * @returns New text node.
     */
    splitText(offset: number): Text;
    /**
     * Converts to string.
     *
     * @returns String.
     */
    toString(): string;
    /**
     * @override
     */
    [PropertySymbol.cloneNode](deep?: boolean): Text;
    /**
     * @override
     */
    [PropertySymbol.connectedToNode](): void;
    /**
     * @override
     */
    [PropertySymbol.disconnectedFromNode](): void;
}
//# sourceMappingURL=Text.d.ts.map