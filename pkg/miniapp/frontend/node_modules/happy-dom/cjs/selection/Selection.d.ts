import Document from '../nodes/document/Document.cjs';
import Node from '../nodes/node/Node.cjs';
import Range from '../range/Range.cjs';
/**
 * Selection.
 *
 * Based on logic from:
 * https://github.com/jsdom/jsdom/blob/master/lib/jsdom/living/selection/Selection-impl.js
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/Selection.
 */
export default class Selection {
    #private;
    /**
     * Constructor.
     *
     * @param ownerDocument Owner document.
     */
    constructor(ownerDocument: Document);
    /**
     * Returns range count.
     *
     * @see https://w3c.github.io/selection-api/#dom-selection-rangecount
     * @returns Range count.
     */
    get rangeCount(): number;
    /**
     * Returns collapsed state.
     *
     * @see https://w3c.github.io/selection-api/#dom-selection-iscollapsed
     * @returns "true" if collapsed.
     */
    get isCollapsed(): boolean;
    /**
     * Returns type.
     *
     * @see https://w3c.github.io/selection-api/#dom-selection-type
     * @returns Type.
     */
    get type(): string;
    /**
     * Returns anchor node.
     *
     * @see https://w3c.github.io/selection-api/#dom-selection-anchornode
     * @returns Node.
     */
    get anchorNode(): Node;
    /**
     * Returns anchor offset.
     *
     * @see https://w3c.github.io/selection-api/#dom-selection-anchoroffset
     * @returns Node.
     */
    get anchorOffset(): number;
    /**
     * Returns anchor node.
     *
     * @deprecated
     * @alias anchorNode
     * @returns Node.
     */
    get baseNode(): Node;
    /**
     * Returns anchor offset.
     *
     * @deprecated
     * @alias anchorOffset
     * @returns Node.
     */
    get baseOffset(): number;
    /**
     * Returns focus node.
     *
     * @see https://w3c.github.io/selection-api/#dom-selection-focusnode
     * @returns Node.
     */
    get focusNode(): Node;
    /**
     * Returns focus offset.
     *
     * @see https://w3c.github.io/selection-api/#dom-selection-focusoffset
     * @returns Node.
     */
    get focusOffset(): number;
    /**
     * Returns focus node.
     *
     * @deprecated
     * @alias focusNode
     * @returns Node.
     */
    get extentNode(): Node;
    /**
     * Returns focus offset.
     *
     * @deprecated
     * @alias focusOffset
     * @returns Node.
     */
    get extentOffset(): number;
    /**
     * Adds a range.
     *
     * @see https://w3c.github.io/selection-api/#dom-selection-addrange
     * @param newRange Range.
     */
    addRange(newRange: Range): void;
    /**
     * Returns Range.
     *
     * @see https://w3c.github.io/selection-api/#dom-selection-getrangeat
     * @param index Index.
     * @returns Range.
     */
    getRangeAt(index: number): Range;
    /**
     * Removes a range from a selection.
     *
     * @see https://w3c.github.io/selection-api/#dom-selection-removerange
     * @param range Range.
     */
    removeRange(range: Range): void;
    /**
     * Removes all ranges.
     */
    removeAllRanges(): void;
    /**
     * Removes all ranges.
     *
     * @alias removeAllRanges()
     */
    empty(): void;
    /**
     * Collapses the current selection to a single point.
     *
     * @see https://w3c.github.io/selection-api/#dom-selection-collapse
     * @param node Node.
     * @param offset Offset.
     */
    collapse(node: Node, offset: number): void;
    /**
     * Collapses the current selection to a single point.
     *
     * @see https://w3c.github.io/selection-api/#dom-selection-setposition
     * @alias collapse()
     * @param node Node.
     * @param offset Offset.
     */
    setPosition(node: Node, offset: number): void;
    /**
     * Collapses the selection to the end.
     *
     * @see https://w3c.github.io/selection-api/#dom-selection-collapsetoend
     */
    collapseToEnd(): void;
    /**
     * Collapses the selection to the start.
     *
     * @see https://w3c.github.io/selection-api/#dom-selection-collapsetostart
     */
    collapseToStart(): void;
    /**
     * Indicates whether a specified node is part of the selection.
     *
     * @see https://w3c.github.io/selection-api/#dom-selection-containsnode
     * @param node Node.
     * @param [allowPartialContainment] Set to "true" to allow partial containment.
     * @returns Always returns "true" for now.
     */
    containsNode(node: Node, allowPartialContainment?: boolean): boolean;
    /**
     * Deletes the selected text from the document's DOM.
     *
     * @see https://w3c.github.io/selection-api/#dom-selection-deletefromdocument
     */
    deleteFromDocument(): void;
    /**
     * Moves the focus of the selection to a specified point.
     *
     * @see https://w3c.github.io/selection-api/#dom-selection-extend
     * @param node Node.
     * @param offset Offset.
     */
    extend(node: Node, offset: number): void;
    /**
     * Selects all children.
     *
     * @see https://w3c.github.io/selection-api/#dom-selection-selectallchildren
     * @param node Node.
     */
    selectAllChildren(node: Node): void;
    /**
     * Sets the selection to be a range including all or parts of two specified DOM nodes, and any content located between them.
     *
     * @see https://w3c.github.io/selection-api/#dom-selection-setbaseandextent
     * @param anchorNode Anchor node.
     * @param anchorOffset Anchor offset.
     * @param focusNode Focus node.
     * @param focusOffset Focus offset.
     */
    setBaseAndExtent(anchorNode: Node, anchorOffset: number, focusNode: Node, focusOffset: number): void;
    /**
     * Returns string currently being represented by the selection object.
     *
     * @returns Selection as string.
     */
    toString(): string;
}
//# sourceMappingURL=Selection.d.ts.map