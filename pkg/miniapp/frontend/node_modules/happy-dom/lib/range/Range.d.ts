import Node from '../nodes/node/Node.js';
import * as PropertySymbol from '../PropertySymbol.js';
import Document from '../nodes/document/Document.js';
import DocumentFragment from '../nodes/document-fragment/DocumentFragment.js';
import DOMRect from '../dom/DOMRect.js';
import RangeHowEnum from './RangeHowEnum.js';
import DOMRectList from '../dom/DOMRectList.js';
import IRangeBoundaryPoint from './IRangeBoundaryPoint.js';
import BrowserWindow from '../window/BrowserWindow.js';
/**
 * Range.
 *
 * Based on logic from:
 * https://github.com/jsdom/jsdom/blob/master/lib/jsdom/living/range/Range-impl.js
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/Range.
 */
export default class Range {
    protected [PropertySymbol.window]: BrowserWindow;
    static readonly END_TO_END: number;
    static readonly END_TO_START: number;
    static readonly START_TO_END: number;
    static readonly START_TO_START: number;
    readonly END_TO_END: number;
    readonly END_TO_START: number;
    readonly START_TO_END: number;
    readonly START_TO_START: number;
    [PropertySymbol.start]: IRangeBoundaryPoint | null;
    [PropertySymbol.end]: IRangeBoundaryPoint | null;
    readonly [PropertySymbol.ownerDocument]: Document;
    /**
     * Constructor.
     *
     * @param window Window.
     */
    constructor();
    /**
     * Returns start container.
     *
     * @see https://dom.spec.whatwg.org/#dom-range-startcontainer
     * @returns Start container.
     */
    get startContainer(): Node;
    /**
     * Returns end container.
     *
     * @see https://dom.spec.whatwg.org/#dom-range-endcontainer
     * @returns End container.
     */
    get endContainer(): Node;
    /**
     * Returns start offset.
     *
     * @see https://dom.spec.whatwg.org/#dom-range-startoffset
     * @returns Start offset.
     */
    get startOffset(): number;
    /**
     * Returns end offset.
     *
     * @see https://dom.spec.whatwg.org/#dom-range-endoffset
     * @returns End offset.
     */
    get endOffset(): number;
    /**
     * Returns a boolean value indicating whether the range's start and end points are at the same position.
     *
     * @see https://dom.spec.whatwg.org/#dom-range-collapsed
     * @returns Collapsed.
     */
    get collapsed(): boolean;
    /**
     * Returns the deepest Node that contains the startContainer and endContainer nodes.
     *
     * @see https://dom.spec.whatwg.org/#dom-range-commonancestorcontainer
     * @returns Node.
     */
    get commonAncestorContainer(): Node;
    /**
     * Returns -1, 0, or 1 depending on whether the referenceNode is before, the same as, or after the Range.
     *
     * @see https://dom.spec.whatwg.org/#dom-range-collapse
     * @param toStart A boolean value: true collapses the Range to its start, false to its end. If omitted, it defaults to false.
     */
    collapse(toStart?: boolean): void;
    /**
     * Compares the boundary points of the Range with those of another range.
     *
     * @see https://dom.spec.whatwg.org/#dom-range-compareboundarypoints
     * @param how How.
     * @param sourceRange Range.
     * @returns A number, -1, 0, or 1, indicating whether the corresponding boundary-point of the Range is respectively before, equal to, or after the corresponding boundary-point of sourceRange.
     */
    compareBoundaryPoints(how: RangeHowEnum, sourceRange: Range): number;
    /**
     * Returns -1, 0, or 1 depending on whether the referenceNode is before, the same as, or after the Range.
     *
     * @see https://dom.spec.whatwg.org/#dom-range-comparepoint
     * @param node Reference node.
     * @param offset Offset.
     * @returns -1,0, or 1.
     */
    comparePoint(node: Node, offset: any): number;
    /**
     * Returns a DocumentFragment copying the objects of type Node included in the Range.
     *
     * @see https://dom.spec.whatwg.org/#concept-range-clone
     * @returns Document fragment.
     */
    cloneContents(): DocumentFragment;
    /**
     * Returns a Range object with boundary points identical to the cloned Range.
     *
     * @see https://dom.spec.whatwg.org/#dom-range-clonerange
     * @returns Range.
     */
    cloneRange(): Range;
    /**
     * Returns a DocumentFragment by invoking the HTML fragment parsing algorithm or the XML fragment parsing algorithm with the start of the range (the parent of the selected node) as the context node. The HTML fragment parsing algorithm is used if the range belongs to a Document whose HTMLness bit is set. In the HTML case, if the context node would be html, for historical reasons the fragment parsing algorithm is invoked with body as the context instead.
     *
     * @see https://w3c.github.io/DOM-Parsing/#dfn-fragment-parsing-algorithm
     * @param tagString Tag string.
     * @returns Document fragment.
     */
    createContextualFragment(tagString: string): DocumentFragment;
    /**
     * Removes the contents of the Range from the Document.
     *
     * @see https://dom.spec.whatwg.org/#dom-range-deletecontents
     */
    deleteContents(): void;
    /**
     * Does nothing. It used to disable the Range object and enable the browser to release associated resources. The method has been kept for compatibility.
     *
     * @see https://dom.spec.whatwg.org/#dom-range-detach
     */
    detach(): void;
    /**
     * Moves contents of the Range from the document tree into a DocumentFragment.
     *
     * @see https://dom.spec.whatwg.org/#dom-range-extractcontents
     * @returns Document fragment.
     */
    extractContents(): DocumentFragment;
    /**
     * Returns a DOMRect object that bounds the contents of the range; this is a rectangle enclosing the union of the bounding rectangles for all the elements in the range.
     *
     * @returns DOMRect object.
     */
    getBoundingClientRect(): DOMRect;
    /**
     * The Range.getClientRects() method returns a list of DOMRect objects representing the area of the screen occupied by the range. This is created by aggregating the results of calls to Element.getClientRects() for all the elements in the range.
     *
     * @returns DOMRect objects.
     */
    getClientRects(): DOMRectList;
    /**
     * Returns a boolean indicating whether the given point is in the Range.
     *
     * @see https://dom.spec.whatwg.org/#dom-range-ispointinrange
     * @param node Reference node.
     * @param offset Offset.
     * @returns "true" if in range.
     */
    isPointInRange(node: Node, offset?: number): boolean;
    /**
     * Inserts a node at the start of the Range.
     *
     * @see https://dom.spec.whatwg.org/#concept-range-insert
     * @param newNode New node.
     */
    insertNode(newNode: Node): void;
    /**
     * Returns a boolean indicating whether the given Node intersects the Range.
     *
     * @see https://dom.spec.whatwg.org/#dom-range-intersectsnode
     * @param node Reference node.
     * @returns "true" if it intersects.
     */
    intersectsNode(node: Node): boolean;
    /**
     * Sets the Range to contain the Node and its contents.
     *
     * @see https://dom.spec.whatwg.org/#concept-range-select
     * @param node Reference node.
     */
    selectNode(node: Node): void;
    /**
     * Sets the Range to contain the contents of a Node.
     *
     * @see https://dom.spec.whatwg.org/#dom-range-selectnodecontents
     * @param node Reference node.
     */
    selectNodeContents(node: Node): void;
    /**
     * Sets the end position of a Range to be located at the given offset into the specified node x.
     *
     * @see https://dom.spec.whatwg.org/#dom-range-setend
     * @param node End node.
     * @param offset End offset.
     */
    setEnd(node: Node, offset?: number): void;
    /**
     * Sets the start position of a Range.
     *
     * @see https://dom.spec.whatwg.org/#dom-range-setstart
     * @param node Start node.
     * @param offset Start offset.
     */
    setStart(node: Node, offset?: number): void;
    /**
     * Sets the end position of a Range relative to another Node.
     *
     * @see https://dom.spec.whatwg.org/#dom-range-setendafter
     * @param node Reference node.
     */
    setEndAfter(node: Node): void;
    /**
     * Sets the end position of a Range relative to another Node.
     *
     * @see https://dom.spec.whatwg.org/#dom-range-setendbefore
     * @param node Reference node.
     */
    setEndBefore(node: Node): void;
    /**
     * Sets the start position of a Range relative to a Node.
     *
     * @see https://dom.spec.whatwg.org/#dom-range-setstartafter
     * @param node Reference node.
     */
    setStartAfter(node: Node): void;
    /**
     * Sets the start position of a Range relative to another Node.
     *
     * @see https://dom.spec.whatwg.org/#dom-range-setstartbefore
     * @param node Reference node.
     */
    setStartBefore(node: Node): void;
    /**
     * Moves content of the Range into a new node, placing the new node at the start of the specified range.
     *
     * @see https://dom.spec.whatwg.org/#dom-range-surroundcontents
     * @param newParent New parent.
     */
    surroundContents(newParent: Node): void;
    /**
     * Returns the text of the Range.
     *
     * @see https://dom.spec.whatwg.org/#dom-range-stringifier
     */
    toString(): string;
}
//# sourceMappingURL=Range.d.ts.map