import Element from '../element/Element.cjs';
import Node from '../node/Node.cjs';
import * as PropertySymbol from '../../PropertySymbol.cjs';
import NodeTypeEnum from '../node/NodeTypeEnum.cjs';
/**
 * Attribute node interface.
 *
 * Reference: https://developer.mozilla.org/en-US/docs/Web/API/Attr.
 */
export default class Attr extends Node implements Attr {
    cloneNode: (deep?: boolean) => Attr;
    [PropertySymbol.nodeType]: NodeTypeEnum;
    [PropertySymbol.namespaceURI]: string | null;
    [PropertySymbol.name]: string | null;
    [PropertySymbol.localName]: string | null;
    [PropertySymbol.prefix]: string | null;
    [PropertySymbol.value]: string | null;
    [PropertySymbol.specified]: boolean;
    [PropertySymbol.ownerElement]: Element | null;
    /**
     * Returns specified.
     *
     * @returns Specified.
     */
    get specified(): boolean;
    /**
     * Returns owner element.
     *
     * @returns Owner element.
     */
    get ownerElement(): Element | null;
    /**
     * Returns value.
     *
     * @returns Value.
     */
    get value(): string;
    /**
     * Sets value.
     *
     * @param value Value.
     */
    set value(value: string);
    /**
     * Returns name.
     *
     * @returns Name.
     */
    get name(): string;
    /**
     * Returns local name.
     *
     * @returns Local name.
     */
    get localName(): string;
    /**
     * Returns prefix.
     *
     * @returns Prefix.
     */
    get prefix(): string;
    /**
     * @override
     */
    get textContent(): string;
    /**
     * Returns namespace URI.
     *
     * @returns Namespace URI.
     */
    get namespaceURI(): string | null;
    /**
     * @override
     */
    [PropertySymbol.cloneNode](deep?: boolean): Attr;
}
//# sourceMappingURL=Attr.d.ts.map