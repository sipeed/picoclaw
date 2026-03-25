import Event from '../event/Event.js';
import * as PropertySymbol from '../PropertySymbol.js';
import DOMExceptionNameEnum from '../exception/DOMExceptionNameEnum.js';
import NodeTypeEnum from '../nodes/node/NodeTypeEnum.js';
import NodeUtility from '../nodes/node/NodeUtility.js';
import RangeUtility from '../range/RangeUtility.js';
import SelectionDirectionEnum from './SelectionDirectionEnum.js';
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
    #ownerDocument = null;
    #range = null;
    #direction = SelectionDirectionEnum.directionless;
    /**
     * Constructor.
     *
     * @param ownerDocument Owner document.
     */
    constructor(ownerDocument) {
        this.#ownerDocument = ownerDocument;
    }
    /**
     * Returns range count.
     *
     * @see https://w3c.github.io/selection-api/#dom-selection-rangecount
     * @returns Range count.
     */
    get rangeCount() {
        return this.#range ? 1 : 0;
    }
    /**
     * Returns collapsed state.
     *
     * @see https://w3c.github.io/selection-api/#dom-selection-iscollapsed
     * @returns "true" if collapsed.
     */
    get isCollapsed() {
        return this.#range === null || this.#range.collapsed;
    }
    /**
     * Returns type.
     *
     * @see https://w3c.github.io/selection-api/#dom-selection-type
     * @returns Type.
     */
    get type() {
        if (!this.#range) {
            return 'None';
        }
        else if (this.#range.collapsed) {
            return 'Caret';
        }
        return 'Range';
    }
    /**
     * Returns anchor node.
     *
     * @see https://w3c.github.io/selection-api/#dom-selection-anchornode
     * @returns Node.
     */
    get anchorNode() {
        if (!this.#range) {
            return null;
        }
        return this.#direction === SelectionDirectionEnum.forwards
            ? this.#range.startContainer
            : this.#range.endContainer;
    }
    /**
     * Returns anchor offset.
     *
     * @see https://w3c.github.io/selection-api/#dom-selection-anchoroffset
     * @returns Node.
     */
    get anchorOffset() {
        if (!this.#range) {
            return 0;
        }
        return this.#direction === SelectionDirectionEnum.forwards
            ? this.#range.startOffset
            : this.#range.endOffset;
    }
    /**
     * Returns anchor node.
     *
     * @deprecated
     * @alias anchorNode
     * @returns Node.
     */
    get baseNode() {
        return this.anchorNode;
    }
    /**
     * Returns anchor offset.
     *
     * @deprecated
     * @alias anchorOffset
     * @returns Node.
     */
    get baseOffset() {
        return this.anchorOffset;
    }
    /**
     * Returns focus node.
     *
     * @see https://w3c.github.io/selection-api/#dom-selection-focusnode
     * @returns Node.
     */
    get focusNode() {
        return this.anchorNode;
    }
    /**
     * Returns focus offset.
     *
     * @see https://w3c.github.io/selection-api/#dom-selection-focusoffset
     * @returns Node.
     */
    get focusOffset() {
        return this.anchorOffset;
    }
    /**
     * Returns focus node.
     *
     * @deprecated
     * @alias focusNode
     * @returns Node.
     */
    get extentNode() {
        return this.focusNode;
    }
    /**
     * Returns focus offset.
     *
     * @deprecated
     * @alias focusOffset
     * @returns Node.
     */
    get extentOffset() {
        return this.focusOffset;
    }
    /**
     * Adds a range.
     *
     * @see https://w3c.github.io/selection-api/#dom-selection-addrange
     * @param newRange Range.
     */
    addRange(newRange) {
        if (!newRange) {
            throw new this.#ownerDocument[PropertySymbol.window].TypeError('Failed to execute addRange on Selection. Parameter 1 is not of type Range.');
        }
        if (!this.#range && newRange[PropertySymbol.ownerDocument] === this.#ownerDocument) {
            this.#associateRange(newRange);
        }
    }
    /**
     * Returns Range.
     *
     * @see https://w3c.github.io/selection-api/#dom-selection-getrangeat
     * @param index Index.
     * @returns Range.
     */
    getRangeAt(index) {
        if (!this.#range || index !== 0) {
            throw new this.#ownerDocument[PropertySymbol.window].DOMException('Invalid range index.', DOMExceptionNameEnum.indexSizeError);
        }
        return this.#range;
    }
    /**
     * Removes a range from a selection.
     *
     * @see https://w3c.github.io/selection-api/#dom-selection-removerange
     * @param range Range.
     */
    removeRange(range) {
        if (this.#range !== range) {
            throw new this.#ownerDocument[PropertySymbol.window].DOMException('Invalid range.', DOMExceptionNameEnum.notFoundError);
        }
        this.#associateRange(null);
    }
    /**
     * Removes all ranges.
     */
    removeAllRanges() {
        this.#associateRange(null);
    }
    /**
     * Removes all ranges.
     *
     * @alias removeAllRanges()
     */
    empty() {
        this.removeAllRanges();
    }
    /**
     * Collapses the current selection to a single point.
     *
     * @see https://w3c.github.io/selection-api/#dom-selection-collapse
     * @param node Node.
     * @param offset Offset.
     */
    collapse(node, offset) {
        if (node === null) {
            this.removeAllRanges();
            return;
        }
        if (node[PropertySymbol.nodeType] === NodeTypeEnum.documentTypeNode) {
            throw new this.#ownerDocument[PropertySymbol.window].DOMException("DocumentType Node can't be used as boundary point.", DOMExceptionNameEnum.invalidNodeTypeError);
        }
        if (offset > NodeUtility.getNodeLength(node)) {
            throw new this.#ownerDocument[PropertySymbol.window].DOMException('Invalid range index.', DOMExceptionNameEnum.indexSizeError);
        }
        if (node[PropertySymbol.ownerDocument] !== this.#ownerDocument) {
            return;
        }
        const newRange = new this.#ownerDocument[PropertySymbol.window].Range();
        newRange[PropertySymbol.start].node = node;
        newRange[PropertySymbol.start].offset = offset;
        newRange[PropertySymbol.end].node = node;
        newRange[PropertySymbol.end].offset = offset;
        this.#associateRange(newRange);
    }
    /**
     * Collapses the current selection to a single point.
     *
     * @see https://w3c.github.io/selection-api/#dom-selection-setposition
     * @alias collapse()
     * @param node Node.
     * @param offset Offset.
     */
    setPosition(node, offset) {
        this.collapse(node, offset);
    }
    /**
     * Collapses the selection to the end.
     *
     * @see https://w3c.github.io/selection-api/#dom-selection-collapsetoend
     */
    collapseToEnd() {
        if (this.#range === null) {
            throw new this.#ownerDocument[PropertySymbol.window].DOMException('There is no selection to collapse.', DOMExceptionNameEnum.invalidStateError);
        }
        const { node, offset } = this.#range[PropertySymbol.end];
        const newRange = new this.#ownerDocument[PropertySymbol.window].Range();
        newRange[PropertySymbol.start].node = node;
        newRange[PropertySymbol.start].offset = offset;
        newRange[PropertySymbol.end].node = node;
        newRange[PropertySymbol.end].offset = offset;
        this.#associateRange(newRange);
    }
    /**
     * Collapses the selection to the start.
     *
     * @see https://w3c.github.io/selection-api/#dom-selection-collapsetostart
     */
    collapseToStart() {
        if (!this.#range) {
            throw new this.#ownerDocument[PropertySymbol.window].DOMException('There is no selection to collapse.', DOMExceptionNameEnum.invalidStateError);
        }
        const { node, offset } = this.#range[PropertySymbol.start];
        const newRange = new this.#ownerDocument[PropertySymbol.window].Range();
        newRange[PropertySymbol.start].node = node;
        newRange[PropertySymbol.start].offset = offset;
        newRange[PropertySymbol.end].node = node;
        newRange[PropertySymbol.end].offset = offset;
        this.#associateRange(newRange);
    }
    /**
     * Indicates whether a specified node is part of the selection.
     *
     * @see https://w3c.github.io/selection-api/#dom-selection-containsnode
     * @param node Node.
     * @param [allowPartialContainment] Set to "true" to allow partial containment.
     * @returns Always returns "true" for now.
     */
    containsNode(node, allowPartialContainment = false) {
        if (!this.#range || node[PropertySymbol.ownerDocument] !== this.#ownerDocument) {
            return false;
        }
        const startIsBeforeNode = RangeUtility.compareBoundaryPointsPosition(this.#range[PropertySymbol.start], {
            node,
            offset: 0
        }) === -1;
        const endIsAfterNode = RangeUtility.compareBoundaryPointsPosition(this.#range[PropertySymbol.end], {
            node,
            offset: NodeUtility.getNodeLength(node)
        }) === 1;
        return allowPartialContainment
            ? startIsBeforeNode || endIsAfterNode
            : startIsBeforeNode && endIsAfterNode;
    }
    /**
     * Deletes the selected text from the document's DOM.
     *
     * @see https://w3c.github.io/selection-api/#dom-selection-deletefromdocument
     */
    deleteFromDocument() {
        if (this.#range) {
            this.#range.deleteContents();
        }
    }
    /**
     * Moves the focus of the selection to a specified point.
     *
     * @see https://w3c.github.io/selection-api/#dom-selection-extend
     * @param node Node.
     * @param offset Offset.
     */
    extend(node, offset) {
        if (node[PropertySymbol.ownerDocument] !== this.#ownerDocument) {
            return;
        }
        if (!this.#range) {
            throw new this.#ownerDocument[PropertySymbol.window].DOMException('There is no selection to extend.', DOMExceptionNameEnum.invalidStateError);
        }
        const anchorNode = this.anchorNode;
        const anchorOffset = this.anchorOffset;
        const newRange = new this.#ownerDocument[PropertySymbol.window].Range();
        newRange[PropertySymbol.start].node = node;
        newRange[PropertySymbol.start].offset = 0;
        newRange[PropertySymbol.end].node = node;
        newRange[PropertySymbol.end].offset = 0;
        if (node[PropertySymbol.ownerDocument] !== this.#range[PropertySymbol.ownerDocument]) {
            newRange[PropertySymbol.start].offset = offset;
            newRange[PropertySymbol.end].offset = offset;
        }
        else if (RangeUtility.compareBoundaryPointsPosition({ node: anchorNode, offset: anchorOffset }, { node, offset }) <= 0) {
            newRange[PropertySymbol.start].node = anchorNode;
            newRange[PropertySymbol.start].offset = anchorOffset;
            newRange[PropertySymbol.end].node = node;
            newRange[PropertySymbol.end].offset = offset;
        }
        else {
            newRange[PropertySymbol.start].node = node;
            newRange[PropertySymbol.start].offset = offset;
            newRange[PropertySymbol.end].node = anchorNode;
            newRange[PropertySymbol.end].offset = anchorOffset;
        }
        this.#associateRange(newRange);
        this.#direction =
            RangeUtility.compareBoundaryPointsPosition({ node, offset }, { node: anchorNode, offset: anchorOffset }) === -1
                ? SelectionDirectionEnum.backwards
                : SelectionDirectionEnum.forwards;
    }
    /**
     * Selects all children.
     *
     * @see https://w3c.github.io/selection-api/#dom-selection-selectallchildren
     * @param node Node.
     */
    selectAllChildren(node) {
        if (node[PropertySymbol.nodeType] === NodeTypeEnum.documentTypeNode) {
            throw new this.#ownerDocument[PropertySymbol.window].DOMException("DocumentType Node can't be used as boundary point.", DOMExceptionNameEnum.invalidNodeTypeError);
        }
        if (node[PropertySymbol.ownerDocument] !== this.#ownerDocument) {
            return;
        }
        const length = node[PropertySymbol.nodeArray].length;
        const newRange = new this.#ownerDocument[PropertySymbol.window].Range();
        newRange[PropertySymbol.start].node = node;
        newRange[PropertySymbol.start].offset = 0;
        newRange[PropertySymbol.end].node = node;
        newRange[PropertySymbol.end].offset = length;
        this.#associateRange(newRange);
    }
    /**
     * Sets the selection to be a range including all or parts of two specified DOM nodes, and any content located between them.
     *
     * @see https://w3c.github.io/selection-api/#dom-selection-setbaseandextent
     * @param anchorNode Anchor node.
     * @param anchorOffset Anchor offset.
     * @param focusNode Focus node.
     * @param focusOffset Focus offset.
     */
    setBaseAndExtent(anchorNode, anchorOffset, focusNode, focusOffset) {
        if (anchorOffset > NodeUtility.getNodeLength(anchorNode) ||
            focusOffset > NodeUtility.getNodeLength(focusNode)) {
            throw new this.#ownerDocument[PropertySymbol.window].DOMException('Invalid anchor or focus offset.', DOMExceptionNameEnum.indexSizeError);
        }
        if (anchorNode[PropertySymbol.ownerDocument] !== this.#ownerDocument ||
            focusNode[PropertySymbol.ownerDocument] !== this.#ownerDocument) {
            return;
        }
        const anchor = { node: anchorNode, offset: anchorOffset };
        const focus = { node: focusNode, offset: focusOffset };
        const newRange = new this.#ownerDocument[PropertySymbol.window].Range();
        if (RangeUtility.compareBoundaryPointsPosition(anchor, focus) === -1) {
            newRange[PropertySymbol.start] = anchor;
            newRange[PropertySymbol.end] = focus;
        }
        else {
            newRange[PropertySymbol.start] = focus;
            newRange[PropertySymbol.end] = anchor;
        }
        this.#associateRange(newRange);
        this.#direction =
            RangeUtility.compareBoundaryPointsPosition(focus, anchor) === -1
                ? SelectionDirectionEnum.backwards
                : SelectionDirectionEnum.forwards;
    }
    /**
     * Returns string currently being represented by the selection object.
     *
     * @returns Selection as string.
     */
    toString() {
        return this.#range ? this.#range.toString() : '';
    }
    /**
     * Sets the current range.
     *
     * @param range Range.
     */
    #associateRange(range) {
        const oldRange = this.#range;
        this.#range = range;
        this.#direction =
            range === null ? SelectionDirectionEnum.directionless : SelectionDirectionEnum.forwards;
        if (oldRange !== this.#range) {
            // https://w3c.github.io/selection-api/#selectionchange-event
            this.#ownerDocument.dispatchEvent(new Event('selectionchange'));
        }
    }
}
//# sourceMappingURL=Selection.js.map