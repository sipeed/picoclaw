import Node from '../nodes/node/Node.js';
import * as PropertySymbol from '../PropertySymbol.js';
import Document from '../nodes/document/Document.js';
import DocumentFragment from '../nodes/document-fragment/DocumentFragment.js';
import DOMRect from '../dom/DOMRect.js';
import RangeHowEnum from './RangeHowEnum.js';
import DOMExceptionNameEnum from '../exception/DOMExceptionNameEnum.js';
import RangeUtility from './RangeUtility.js';
import NodeTypeEnum from '../nodes/node/NodeTypeEnum.js';
import NodeUtility from '../nodes/node/NodeUtility.js';
import Comment from '../nodes/comment/Comment.js';
import Text from '../nodes/text/Text.js';
import DOMRectList from '../dom/DOMRectList.js';
import IRangeBoundaryPoint from './IRangeBoundaryPoint.js';
import BrowserWindow from '../window/BrowserWindow.js';
import HTMLParser from '../html-parser/HTMLParser.js';

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
	// Injected by WindowContextClassExtender
	protected declare [PropertySymbol.window]: BrowserWindow;

	public static readonly END_TO_END: number = RangeHowEnum.endToEnd;
	public static readonly END_TO_START: number = RangeHowEnum.endToStart;
	public static readonly START_TO_END: number = RangeHowEnum.startToEnd;
	public static readonly START_TO_START: number = RangeHowEnum.startToStart;
	public readonly END_TO_END: number = RangeHowEnum.endToEnd;
	public readonly END_TO_START: number = RangeHowEnum.endToStart;
	public readonly START_TO_END: number = RangeHowEnum.startToEnd;
	public readonly START_TO_START: number = RangeHowEnum.startToStart;
	public [PropertySymbol.start]: IRangeBoundaryPoint | null = null;
	public [PropertySymbol.end]: IRangeBoundaryPoint | null = null;
	public readonly [PropertySymbol.ownerDocument]: Document;

	/**
	 * Constructor.
	 *
	 * @param window Window.
	 */
	constructor() {
		const window = this[PropertySymbol.window];

		if (!window) {
			throw new TypeError(
				`Failed to construct '${this.constructor.name}': '${this.constructor.name}' was constructed outside a Window context.`
			);
		}

		this[PropertySymbol.ownerDocument] = window.document;
		this[PropertySymbol.start] = { node: window.document, offset: 0 };
		this[PropertySymbol.end] = { node: window.document, offset: 0 };
	}

	/**
	 * Returns start container.
	 *
	 * @see https://dom.spec.whatwg.org/#dom-range-startcontainer
	 * @returns Start container.
	 */
	public get startContainer(): Node {
		return this[PropertySymbol.start].node;
	}

	/**
	 * Returns end container.
	 *
	 * @see https://dom.spec.whatwg.org/#dom-range-endcontainer
	 * @returns End container.
	 */
	public get endContainer(): Node {
		return this[PropertySymbol.end].node;
	}

	/**
	 * Returns start offset.
	 *
	 * @see https://dom.spec.whatwg.org/#dom-range-startoffset
	 * @returns Start offset.
	 */
	public get startOffset(): number {
		if (this[PropertySymbol.start].offset > 0) {
			const length = NodeUtility.getNodeLength(this[PropertySymbol.start].node);
			if (this[PropertySymbol.start].offset > length) {
				this[PropertySymbol.start].offset = length;
			}
		}

		return this[PropertySymbol.start].offset;
	}

	/**
	 * Returns end offset.
	 *
	 * @see https://dom.spec.whatwg.org/#dom-range-endoffset
	 * @returns End offset.
	 */
	public get endOffset(): number {
		if (this[PropertySymbol.end].offset > 0) {
			const length = NodeUtility.getNodeLength(this[PropertySymbol.end].node);
			if (this[PropertySymbol.end].offset > length) {
				this[PropertySymbol.end].offset = length;
			}
		}

		return this[PropertySymbol.end].offset;
	}

	/**
	 * Returns a boolean value indicating whether the range's start and end points are at the same position.
	 *
	 * @see https://dom.spec.whatwg.org/#dom-range-collapsed
	 * @returns Collapsed.
	 */
	public get collapsed(): boolean {
		return (
			this[PropertySymbol.start].node === this[PropertySymbol.end].node &&
			this.startOffset === this.endOffset
		);
	}

	/**
	 * Returns the deepest Node that contains the startContainer and endContainer nodes.
	 *
	 * @see https://dom.spec.whatwg.org/#dom-range-commonancestorcontainer
	 * @returns Node.
	 */
	public get commonAncestorContainer(): Node {
		let container = this[PropertySymbol.start].node;

		while (container) {
			if (NodeUtility.isInclusiveAncestor(container, this[PropertySymbol.end].node)) {
				return container;
			}
			container = container[PropertySymbol.parentNode];
		}

		return null;
	}

	/**
	 * Returns -1, 0, or 1 depending on whether the referenceNode is before, the same as, or after the Range.
	 *
	 * @see https://dom.spec.whatwg.org/#dom-range-collapse
	 * @param toStart A boolean value: true collapses the Range to its start, false to its end. If omitted, it defaults to false.
	 */
	public collapse(toStart = false): void {
		if (toStart) {
			this[PropertySymbol.end] = Object.assign({}, this[PropertySymbol.start]);
		} else {
			this[PropertySymbol.start] = Object.assign({}, this[PropertySymbol.end]);
		}
	}

	/**
	 * Compares the boundary points of the Range with those of another range.
	 *
	 * @see https://dom.spec.whatwg.org/#dom-range-compareboundarypoints
	 * @param how How.
	 * @param sourceRange Range.
	 * @returns A number, -1, 0, or 1, indicating whether the corresponding boundary-point of the Range is respectively before, equal to, or after the corresponding boundary-point of sourceRange.
	 */
	public compareBoundaryPoints(how: RangeHowEnum, sourceRange: Range): number {
		if (
			how !== RangeHowEnum.startToStart &&
			how !== RangeHowEnum.startToEnd &&
			how !== RangeHowEnum.endToEnd &&
			how !== RangeHowEnum.endToStart
		) {
			throw new this[PropertySymbol.window].DOMException(
				`The comparison method provided must be one of '${RangeHowEnum.startToStart}', '${RangeHowEnum.startToEnd}', '${RangeHowEnum.endToEnd}' or '${RangeHowEnum.endToStart}'.`,
				DOMExceptionNameEnum.notSupportedError
			);
		}

		if (this[PropertySymbol.ownerDocument] !== sourceRange[PropertySymbol.ownerDocument]) {
			throw new this[PropertySymbol.window].DOMException(
				`The two Ranges are not in the same tree.`,
				DOMExceptionNameEnum.wrongDocumentError
			);
		}

		const thisPoint: { node: Node; offset: number } = {
			node: null,
			offset: 0
		};
		const sourcePoint: { node: Node; offset: number } = {
			node: null,
			offset: 0
		};

		switch (how) {
			case RangeHowEnum.startToStart:
				thisPoint.node = this[PropertySymbol.start].node;
				thisPoint.offset = this.startOffset;
				sourcePoint.node = sourceRange[PropertySymbol.start].node;
				sourcePoint.offset = sourceRange.startOffset;
				break;
			case RangeHowEnum.startToEnd:
				thisPoint.node = this[PropertySymbol.end].node;
				thisPoint.offset = this.endOffset;
				sourcePoint.node = sourceRange[PropertySymbol.start].node;
				sourcePoint.offset = sourceRange.startOffset;
				break;
			case RangeHowEnum.endToEnd:
				thisPoint.node = this[PropertySymbol.end].node;
				thisPoint.offset = this.endOffset;
				sourcePoint.node = sourceRange[PropertySymbol.end].node;
				sourcePoint.offset = sourceRange.endOffset;
				break;
			case RangeHowEnum.endToStart:
				thisPoint.node = this[PropertySymbol.start].node;
				thisPoint.offset = this.startOffset;
				sourcePoint.node = sourceRange[PropertySymbol.end].node;
				sourcePoint.offset = sourceRange.endOffset;
				break;
		}

		return RangeUtility.compareBoundaryPointsPosition(thisPoint, sourcePoint);
	}

	/**
	 * Returns -1, 0, or 1 depending on whether the referenceNode is before, the same as, or after the Range.
	 *
	 * @see https://dom.spec.whatwg.org/#dom-range-comparepoint
	 * @param node Reference node.
	 * @param offset Offset.
	 * @returns -1,0, or 1.
	 */
	public comparePoint(node: Node, offset): number {
		if (node[PropertySymbol.ownerDocument] !== this[PropertySymbol.ownerDocument]) {
			throw new this[PropertySymbol.window].DOMException(
				`The two Ranges are not in the same tree.`,
				DOMExceptionNameEnum.wrongDocumentError
			);
		}

		RangeUtility.validateBoundaryPoint({ node, offset });

		const boundaryPoint = { node, offset };

		if (
			RangeUtility.compareBoundaryPointsPosition(boundaryPoint, {
				node: this[PropertySymbol.start].node,
				offset: this.startOffset
			}) === -1
		) {
			return -1;
		} else if (
			RangeUtility.compareBoundaryPointsPosition(boundaryPoint, {
				node: this[PropertySymbol.end].node,
				offset: this.endOffset
			}) === 1
		) {
			return 1;
		}

		return 0;
	}

	/**
	 * Returns a DocumentFragment copying the objects of type Node included in the Range.
	 *
	 * @see https://dom.spec.whatwg.org/#concept-range-clone
	 * @returns Document fragment.
	 */
	public cloneContents(): DocumentFragment {
		const window = this[PropertySymbol.window];
		const fragment = this[PropertySymbol.ownerDocument].createDocumentFragment();
		const startOffset = this.startOffset;
		const endOffset = this.endOffset;

		if (this.collapsed) {
			return fragment;
		}

		if (
			this[PropertySymbol.start].node === this[PropertySymbol.end].node &&
			(this[PropertySymbol.start].node[PropertySymbol.nodeType] === NodeTypeEnum.textNode ||
				this[PropertySymbol.start].node[PropertySymbol.nodeType] ===
					NodeTypeEnum.processingInstructionNode ||
				this[PropertySymbol.start].node[PropertySymbol.nodeType] === NodeTypeEnum.commentNode)
		) {
			const clone = (<Text | Comment>this[PropertySymbol.start].node).cloneNode(false);
			clone[PropertySymbol.data] = clone.substringData(startOffset, endOffset - startOffset);
			fragment.appendChild(clone);
			return fragment;
		}

		let commonAncestor = this[PropertySymbol.start].node;
		while (!NodeUtility.isInclusiveAncestor(commonAncestor, this[PropertySymbol.end].node)) {
			commonAncestor = commonAncestor[PropertySymbol.parentNode];
		}

		let firstPartialContainedChild = null;
		if (
			!NodeUtility.isInclusiveAncestor(
				this[PropertySymbol.start].node,
				this[PropertySymbol.end].node
			)
		) {
			let candidate = commonAncestor.firstChild;
			while (!firstPartialContainedChild) {
				if (RangeUtility.isPartiallyContained(candidate, this)) {
					firstPartialContainedChild = candidate;
				}

				candidate = candidate.nextSibling;
			}
		}

		let lastPartiallyContainedChild = null;
		if (
			!NodeUtility.isInclusiveAncestor(
				this[PropertySymbol.end].node,
				this[PropertySymbol.start].node
			)
		) {
			let candidate = commonAncestor.lastChild;
			while (!lastPartiallyContainedChild) {
				if (RangeUtility.isPartiallyContained(candidate, this)) {
					lastPartiallyContainedChild = candidate;
				}

				candidate = candidate.previousSibling;
			}
		}

		const containedChildren = [];

		for (const node of (<Node>commonAncestor)[PropertySymbol.nodeArray]) {
			if (RangeUtility.isContained(node, this)) {
				if (node[PropertySymbol.nodeType] === NodeTypeEnum.documentTypeNode) {
					throw new window.DOMException(
						'Invalid document type element.',
						DOMExceptionNameEnum.hierarchyRequestError
					);
				}
				containedChildren.push(node);
			}
		}

		if (
			firstPartialContainedChild !== null &&
			(firstPartialContainedChild[PropertySymbol.nodeType] === NodeTypeEnum.textNode ||
				firstPartialContainedChild[PropertySymbol.nodeType] ===
					NodeTypeEnum.processingInstructionNode ||
				firstPartialContainedChild[PropertySymbol.nodeType] === NodeTypeEnum.commentNode)
		) {
			const clone = (<Text | Comment>this[PropertySymbol.start].node).cloneNode(false);
			clone[PropertySymbol.data] = clone.substringData(
				startOffset,
				NodeUtility.getNodeLength(this[PropertySymbol.start].node) - startOffset
			);

			fragment.appendChild(clone);
		} else if (firstPartialContainedChild !== null) {
			const clone = firstPartialContainedChild.cloneNode();
			fragment.appendChild(clone);

			const subRange = new window.Range();
			subRange[PropertySymbol.start].node = this[PropertySymbol.start].node;
			subRange[PropertySymbol.start].offset = startOffset;
			subRange[PropertySymbol.end].node = firstPartialContainedChild;
			subRange[PropertySymbol.end].offset = NodeUtility.getNodeLength(firstPartialContainedChild);

			const subDocumentFragment = subRange.cloneContents();
			clone.appendChild(subDocumentFragment);
		}

		for (const containedChild of containedChildren) {
			const clone = containedChild.cloneNode(true);
			fragment.appendChild(clone);
		}

		if (
			lastPartiallyContainedChild !== null &&
			(lastPartiallyContainedChild[PropertySymbol.nodeType] === NodeTypeEnum.textNode ||
				lastPartiallyContainedChild[PropertySymbol.nodeType] ===
					NodeTypeEnum.processingInstructionNode ||
				lastPartiallyContainedChild[PropertySymbol.nodeType] === NodeTypeEnum.commentNode)
		) {
			const clone = (<Text | Comment>this[PropertySymbol.end].node).cloneNode(false);
			clone[PropertySymbol.data] = clone.substringData(0, endOffset);

			fragment.appendChild(clone);
		} else if (lastPartiallyContainedChild !== null) {
			const clone = lastPartiallyContainedChild.cloneNode(false);
			fragment.appendChild(clone);

			const subRange = new window.Range();
			subRange[PropertySymbol.start].node = lastPartiallyContainedChild;
			subRange[PropertySymbol.start].offset = 0;
			subRange[PropertySymbol.end].node = this[PropertySymbol.end].node;
			subRange[PropertySymbol.end].offset = endOffset;

			const subFragment = subRange.cloneContents();
			clone.appendChild(subFragment);
		}

		return fragment;
	}

	/**
	 * Returns a Range object with boundary points identical to the cloned Range.
	 *
	 * @see https://dom.spec.whatwg.org/#dom-range-clonerange
	 * @returns Range.
	 */
	public cloneRange(): Range {
		const clone = new this[PropertySymbol.window].Range();

		clone[PropertySymbol.start].node = this[PropertySymbol.start].node;
		clone[PropertySymbol.start].offset = this[PropertySymbol.start].offset;
		clone[PropertySymbol.end].node = this[PropertySymbol.end].node;
		clone[PropertySymbol.end].offset = this[PropertySymbol.end].offset;

		return clone;
	}

	/**
	 * Returns a DocumentFragment by invoking the HTML fragment parsing algorithm or the XML fragment parsing algorithm with the start of the range (the parent of the selected node) as the context node. The HTML fragment parsing algorithm is used if the range belongs to a Document whose HTMLness bit is set. In the HTML case, if the context node would be html, for historical reasons the fragment parsing algorithm is invoked with body as the context instead.
	 *
	 * @see https://w3c.github.io/DOM-Parsing/#dfn-fragment-parsing-algorithm
	 * @param tagString Tag string.
	 * @returns Document fragment.
	 */
	public createContextualFragment(tagString: string): DocumentFragment {
		// TODO: Implement support for checking which context to use
		return <DocumentFragment>new HTMLParser(this[PropertySymbol.window]).parse(tagString);
	}

	/**
	 * Removes the contents of the Range from the Document.
	 *
	 * @see https://dom.spec.whatwg.org/#dom-range-deletecontents
	 */
	public deleteContents(): void {
		const startOffset = this.startOffset;
		const endOffset = this.endOffset;

		if (this.collapsed) {
			return;
		}

		if (
			this[PropertySymbol.start].node === this[PropertySymbol.end].node &&
			(this[PropertySymbol.start].node[PropertySymbol.nodeType] === NodeTypeEnum.textNode ||
				this[PropertySymbol.start].node[PropertySymbol.nodeType] ===
					NodeTypeEnum.processingInstructionNode ||
				this[PropertySymbol.start].node[PropertySymbol.nodeType] === NodeTypeEnum.commentNode)
		) {
			(<Text | Comment>this[PropertySymbol.start].node).replaceData(
				startOffset,
				endOffset - startOffset,
				''
			);
			return;
		}

		const nodesToRemove = [];
		let currentNode = this[PropertySymbol.start].node;
		const endNode = NodeUtility.nextDescendantNode(this[PropertySymbol.end].node);
		while (currentNode && currentNode !== endNode) {
			if (
				RangeUtility.isContained(currentNode, this) &&
				!RangeUtility.isContained(currentNode[PropertySymbol.parentNode], this)
			) {
				nodesToRemove.push(currentNode);
			}

			currentNode = NodeUtility.following(currentNode);
		}

		let newNode;
		let newOffset;
		if (
			NodeUtility.isInclusiveAncestor(
				this[PropertySymbol.start].node,
				this[PropertySymbol.end].node
			)
		) {
			newNode = this[PropertySymbol.start].node;
			newOffset = startOffset;
		} else {
			let referenceNode = this[PropertySymbol.start].node;

			while (
				referenceNode &&
				!NodeUtility.isInclusiveAncestor(
					referenceNode[PropertySymbol.parentNode],
					this[PropertySymbol.end].node
				)
			) {
				referenceNode = referenceNode[PropertySymbol.parentNode];
			}

			newNode = referenceNode[PropertySymbol.parentNode];
			newOffset =
				(<Node>referenceNode[PropertySymbol.parentNode])[PropertySymbol.nodeArray].indexOf(
					referenceNode
				) + 1;
		}

		if (
			this[PropertySymbol.start].node[PropertySymbol.nodeType] === NodeTypeEnum.textNode ||
			this[PropertySymbol.start].node[PropertySymbol.nodeType] ===
				NodeTypeEnum.processingInstructionNode ||
			this[PropertySymbol.start].node[PropertySymbol.nodeType] === NodeTypeEnum.commentNode
		) {
			(<Text | Comment>this[PropertySymbol.start].node).replaceData(
				this.startOffset,
				NodeUtility.getNodeLength(this[PropertySymbol.start].node) - this.startOffset,
				''
			);
		}

		for (const node of nodesToRemove) {
			const parent = node[PropertySymbol.parentNode];
			parent.removeChild(node);
		}

		if (
			this[PropertySymbol.end].node[PropertySymbol.nodeType] === NodeTypeEnum.textNode ||
			this[PropertySymbol.end].node[PropertySymbol.nodeType] ===
				NodeTypeEnum.processingInstructionNode ||
			this[PropertySymbol.end].node[PropertySymbol.nodeType] === NodeTypeEnum.commentNode
		) {
			(<Text | Comment>this[PropertySymbol.end].node).replaceData(0, endOffset, '');
		}

		this[PropertySymbol.start].node = newNode;
		this[PropertySymbol.start].offset = newOffset;
		this[PropertySymbol.end].node = newNode;
		this[PropertySymbol.end].offset = newOffset;
	}

	/**
	 * Does nothing. It used to disable the Range object and enable the browser to release associated resources. The method has been kept for compatibility.
	 *
	 * @see https://dom.spec.whatwg.org/#dom-range-detach
	 */
	public detach(): void {
		// Do nothing by spec
	}

	/**
	 * Moves contents of the Range from the document tree into a DocumentFragment.
	 *
	 * @see https://dom.spec.whatwg.org/#dom-range-extractcontents
	 * @returns Document fragment.
	 */
	public extractContents(): DocumentFragment {
		const window = this[PropertySymbol.window];
		const fragment = this[PropertySymbol.ownerDocument].createDocumentFragment();
		const startOffset = this.startOffset;
		const endOffset = this.endOffset;

		if (this.collapsed) {
			return fragment;
		}

		if (
			this[PropertySymbol.start].node === this[PropertySymbol.end].node &&
			(this[PropertySymbol.start].node[PropertySymbol.nodeType] === NodeTypeEnum.textNode ||
				this[PropertySymbol.start].node[PropertySymbol.nodeType] ===
					NodeTypeEnum.processingInstructionNode ||
				this[PropertySymbol.start].node[PropertySymbol.nodeType] === NodeTypeEnum.commentNode)
		) {
			const clone = <Text | Comment>this[PropertySymbol.start].node.cloneNode(false);
			clone[PropertySymbol.data] = clone.substringData(startOffset, endOffset - startOffset);

			fragment.appendChild(clone);

			(<Text | Comment>this[PropertySymbol.start].node).replaceData(
				startOffset,
				endOffset - startOffset,
				''
			);

			return fragment;
		}

		let commonAncestor = this[PropertySymbol.start].node;
		while (!NodeUtility.isInclusiveAncestor(commonAncestor, this[PropertySymbol.end].node)) {
			commonAncestor = commonAncestor[PropertySymbol.parentNode];
		}

		let firstPartialContainedChild = null;
		if (
			!NodeUtility.isInclusiveAncestor(
				this[PropertySymbol.start].node,
				this[PropertySymbol.end].node
			)
		) {
			let candidate = commonAncestor.firstChild;
			while (!firstPartialContainedChild) {
				if (RangeUtility.isPartiallyContained(candidate, this)) {
					firstPartialContainedChild = candidate;
				}

				candidate = candidate.nextSibling;
			}
		}

		let lastPartiallyContainedChild = null;
		if (
			!NodeUtility.isInclusiveAncestor(
				this[PropertySymbol.end].node,
				this[PropertySymbol.start].node
			)
		) {
			let candidate = commonAncestor.lastChild;
			while (!lastPartiallyContainedChild) {
				if (RangeUtility.isPartiallyContained(candidate, this)) {
					lastPartiallyContainedChild = candidate;
				}

				candidate = candidate.previousSibling;
			}
		}

		const containedChildren = [];

		for (const node of (<Node>commonAncestor)[PropertySymbol.nodeArray]) {
			if (RangeUtility.isContained(node, this)) {
				if (node[PropertySymbol.nodeType] === NodeTypeEnum.documentTypeNode) {
					throw new this[PropertySymbol.window].DOMException(
						'Invalid document type element.',
						DOMExceptionNameEnum.hierarchyRequestError
					);
				}
				containedChildren.push(node);
			}
		}

		let newNode;
		let newOffset;
		if (
			NodeUtility.isInclusiveAncestor(
				this[PropertySymbol.start].node,
				this[PropertySymbol.end].node
			)
		) {
			newNode = this[PropertySymbol.start].node;
			newOffset = startOffset;
		} else {
			let referenceNode = this[PropertySymbol.start].node;

			while (
				referenceNode &&
				!NodeUtility.isInclusiveAncestor(
					referenceNode[PropertySymbol.parentNode],
					this[PropertySymbol.end].node
				)
			) {
				referenceNode = referenceNode[PropertySymbol.parentNode];
			}

			newNode = referenceNode[PropertySymbol.parentNode];
			newOffset =
				(<Node>referenceNode[PropertySymbol.parentNode])[PropertySymbol.nodeArray].indexOf(
					referenceNode
				) + 1;
		}

		if (
			firstPartialContainedChild !== null &&
			(firstPartialContainedChild[PropertySymbol.nodeType] === NodeTypeEnum.textNode ||
				firstPartialContainedChild[PropertySymbol.nodeType] ===
					NodeTypeEnum.processingInstructionNode ||
				firstPartialContainedChild[PropertySymbol.nodeType] === NodeTypeEnum.commentNode)
		) {
			const clone = <Text | Comment>this[PropertySymbol.start].node.cloneNode(false);
			clone[PropertySymbol.data] = clone.substringData(
				startOffset,
				NodeUtility.getNodeLength(this[PropertySymbol.start].node) - startOffset
			);

			fragment.appendChild(clone);

			(<Text | Comment>this[PropertySymbol.start].node).replaceData(
				startOffset,
				NodeUtility.getNodeLength(this[PropertySymbol.start].node) - startOffset,
				''
			);
		} else if (firstPartialContainedChild !== null) {
			const clone = firstPartialContainedChild.cloneNode(false);
			fragment.appendChild(clone);

			const subRange = new window.Range();
			subRange[PropertySymbol.start].node = this[PropertySymbol.start].node;
			subRange[PropertySymbol.start].offset = startOffset;
			subRange[PropertySymbol.end].node = firstPartialContainedChild;
			subRange[PropertySymbol.end].offset = NodeUtility.getNodeLength(firstPartialContainedChild);

			const subFragment = subRange.extractContents();
			clone.appendChild(subFragment);
		}

		for (const containedChild of containedChildren) {
			fragment.appendChild(containedChild);
		}

		if (
			lastPartiallyContainedChild !== null &&
			(lastPartiallyContainedChild[PropertySymbol.nodeType] === NodeTypeEnum.textNode ||
				lastPartiallyContainedChild[PropertySymbol.nodeType] ===
					NodeTypeEnum.processingInstructionNode ||
				lastPartiallyContainedChild[PropertySymbol.nodeType] === NodeTypeEnum.commentNode)
		) {
			const clone = <Text | Comment>this[PropertySymbol.end].node.cloneNode(false);
			clone[PropertySymbol.data] = clone.substringData(0, endOffset);

			fragment.appendChild(clone);

			(<Text | Comment>this[PropertySymbol.end].node).replaceData(0, endOffset, '');
		} else if (lastPartiallyContainedChild !== null) {
			const clone = lastPartiallyContainedChild.cloneNode(false);
			fragment.appendChild(clone);

			const subRange = new window.Range();
			subRange[PropertySymbol.start].node = lastPartiallyContainedChild;
			subRange[PropertySymbol.start].offset = 0;
			subRange[PropertySymbol.end].node = this[PropertySymbol.end].node;
			subRange[PropertySymbol.end].offset = endOffset;

			const subFragment = subRange.extractContents();
			clone.appendChild(subFragment);
		}

		this[PropertySymbol.start].node = newNode;
		this[PropertySymbol.start].offset = newOffset;
		this[PropertySymbol.end].node = newNode;
		this[PropertySymbol.end].offset = newOffset;

		return fragment;
	}

	/**
	 * Returns a DOMRect object that bounds the contents of the range; this is a rectangle enclosing the union of the bounding rectangles for all the elements in the range.
	 *
	 * @returns DOMRect object.
	 */
	public getBoundingClientRect(): DOMRect {
		// TODO: Not full implementation
		return new DOMRect();
	}

	/**
	 * The Range.getClientRects() method returns a list of DOMRect objects representing the area of the screen occupied by the range. This is created by aggregating the results of calls to Element.getClientRects() for all the elements in the range.
	 *
	 * @returns DOMRect objects.
	 */
	public getClientRects(): DOMRectList {
		// TODO: Not full implementation
		return new DOMRectList(PropertySymbol.illegalConstructor);
	}

	/**
	 * Returns a boolean indicating whether the given point is in the Range.
	 *
	 * @see https://dom.spec.whatwg.org/#dom-range-ispointinrange
	 * @param node Reference node.
	 * @param offset Offset.
	 * @returns "true" if in range.
	 */
	public isPointInRange(node: Node, offset = 0): boolean {
		if (node[PropertySymbol.ownerDocument] !== this[PropertySymbol.ownerDocument]) {
			return false;
		}

		const boundaryPoint = { node, offset };

		RangeUtility.validateBoundaryPoint(boundaryPoint);

		if (
			RangeUtility.compareBoundaryPointsPosition(boundaryPoint, {
				node: this[PropertySymbol.start].node,
				offset: this.startOffset
			}) === -1 ||
			RangeUtility.compareBoundaryPointsPosition(boundaryPoint, {
				node: this[PropertySymbol.end].node,
				offset: this.endOffset
			}) === 1
		) {
			return false;
		}

		return true;
	}

	/**
	 * Inserts a node at the start of the Range.
	 *
	 * @see https://dom.spec.whatwg.org/#concept-range-insert
	 * @param newNode New node.
	 */
	public insertNode(newNode: Node): void {
		if (
			this[PropertySymbol.start].node[PropertySymbol.nodeType] ===
				NodeTypeEnum.processingInstructionNode ||
			this[PropertySymbol.start].node[PropertySymbol.nodeType] === NodeTypeEnum.commentNode ||
			(this[PropertySymbol.start].node[PropertySymbol.nodeType] === NodeTypeEnum.textNode &&
				!this[PropertySymbol.start].node[PropertySymbol.parentNode]) ||
			newNode === this[PropertySymbol.start].node
		) {
			throw new this[PropertySymbol.window].DOMException(
				'Invalid start node.',
				DOMExceptionNameEnum.hierarchyRequestError
			);
		}

		let referenceNode =
			this[PropertySymbol.start].node[PropertySymbol.nodeType] === NodeTypeEnum.textNode
				? this[PropertySymbol.start].node
				: (<Node>this[PropertySymbol.start].node)[PropertySymbol.nodeArray][this.startOffset] ||
					null;
		const parent = !referenceNode
			? this[PropertySymbol.start].node
			: referenceNode[PropertySymbol.parentNode];

		if (this[PropertySymbol.start].node[PropertySymbol.nodeType] === NodeTypeEnum.textNode) {
			referenceNode = (<Text>this[PropertySymbol.start].node).splitText(this.startOffset);
		}

		if (newNode === referenceNode) {
			referenceNode = referenceNode.nextSibling;
		}

		const nodeParent = newNode[PropertySymbol.parentNode];
		if (nodeParent) {
			nodeParent.removeChild(newNode);
		}

		let newOffset = !referenceNode
			? NodeUtility.getNodeLength(parent)
			: (<Node>referenceNode[PropertySymbol.parentNode])[PropertySymbol.nodeArray].indexOf(
					referenceNode
				);
		newOffset +=
			newNode[PropertySymbol.nodeType] === NodeTypeEnum.documentFragmentNode
				? NodeUtility.getNodeLength(newNode)
				: 1;

		parent.insertBefore(newNode, referenceNode);

		if (this.collapsed) {
			this[PropertySymbol.end].node = parent;
			this[PropertySymbol.end].offset = newOffset;
		}
	}

	/**
	 * Returns a boolean indicating whether the given Node intersects the Range.
	 *
	 * @see https://dom.spec.whatwg.org/#dom-range-intersectsnode
	 * @param node Reference node.
	 * @returns "true" if it intersects.
	 */
	public intersectsNode(node: Node): boolean {
		if (node[PropertySymbol.ownerDocument] !== this[PropertySymbol.ownerDocument]) {
			return false;
		}

		const parent = node[PropertySymbol.parentNode];

		if (!parent) {
			return true;
		}

		const offset = (<Node>parent)[PropertySymbol.nodeArray].indexOf(node);

		return (
			RangeUtility.compareBoundaryPointsPosition(
				{ node: parent, offset },
				{ node: this[PropertySymbol.end].node, offset: this.endOffset }
			) === -1 &&
			RangeUtility.compareBoundaryPointsPosition(
				{ node: parent, offset: offset + 1 },
				{ node: this[PropertySymbol.start].node, offset: this.startOffset }
			) === 1
		);
	}

	/**
	 * Sets the Range to contain the Node and its contents.
	 *
	 * @see https://dom.spec.whatwg.org/#concept-range-select
	 * @param node Reference node.
	 */
	public selectNode(node: Node): void {
		if (!node[PropertySymbol.parentNode]) {
			throw new this[PropertySymbol.window].DOMException(
				`The given Node has no parent.`,
				DOMExceptionNameEnum.invalidNodeTypeError
			);
		}

		const index = (<Node>node[PropertySymbol.parentNode])[PropertySymbol.nodeArray].indexOf(node);

		this[PropertySymbol.start].node = node[PropertySymbol.parentNode];
		this[PropertySymbol.start].offset = index;
		this[PropertySymbol.end].node = node[PropertySymbol.parentNode];
		this[PropertySymbol.end].offset = index + 1;
	}

	/**
	 * Sets the Range to contain the contents of a Node.
	 *
	 * @see https://dom.spec.whatwg.org/#dom-range-selectnodecontents
	 * @param node Reference node.
	 */
	public selectNodeContents(node: Node): void {
		if (node[PropertySymbol.nodeType] === NodeTypeEnum.documentTypeNode) {
			throw new this[PropertySymbol.window].DOMException(
				"DocumentType Node can't be used as boundary point.",
				DOMExceptionNameEnum.invalidNodeTypeError
			);
		}

		this[PropertySymbol.start].node = node;
		this[PropertySymbol.start].offset = 0;
		this[PropertySymbol.end].node = node;
		this[PropertySymbol.end].offset = NodeUtility.getNodeLength(node);
	}

	/**
	 * Sets the end position of a Range to be located at the given offset into the specified node x.
	 *
	 * @see https://dom.spec.whatwg.org/#dom-range-setend
	 * @param node End node.
	 * @param offset End offset.
	 */
	public setEnd(node: Node, offset = 0): void {
		RangeUtility.validateBoundaryPoint({ node, offset });

		const boundaryPoint = { node, offset };

		if (
			node[PropertySymbol.ownerDocument] !== this[PropertySymbol.ownerDocument] ||
			RangeUtility.compareBoundaryPointsPosition(boundaryPoint, {
				node: this[PropertySymbol.start].node,
				offset: this.startOffset
			}) === -1
		) {
			this[PropertySymbol.start].node = node;
			this[PropertySymbol.start].offset = offset;
		}

		this[PropertySymbol.end].node = node;
		this[PropertySymbol.end].offset = offset;
	}

	/**
	 * Sets the start position of a Range.
	 *
	 * @see https://dom.spec.whatwg.org/#dom-range-setstart
	 * @param node Start node.
	 * @param offset Start offset.
	 */
	public setStart(node: Node, offset = 0): void {
		RangeUtility.validateBoundaryPoint({ node, offset });

		const boundaryPoint = { node, offset };

		if (
			node[PropertySymbol.ownerDocument] !== this[PropertySymbol.ownerDocument] ||
			RangeUtility.compareBoundaryPointsPosition(boundaryPoint, {
				node: this[PropertySymbol.end].node,
				offset: this.endOffset
			}) === 1
		) {
			this[PropertySymbol.end].node = node;
			this[PropertySymbol.end].offset = offset;
		}

		this[PropertySymbol.start].node = node;
		this[PropertySymbol.start].offset = offset;
	}

	/**
	 * Sets the end position of a Range relative to another Node.
	 *
	 * @see https://dom.spec.whatwg.org/#dom-range-setendafter
	 * @param node Reference node.
	 */
	public setEndAfter(node: Node): void {
		if (!node[PropertySymbol.parentNode]) {
			throw new this[PropertySymbol.window].DOMException(
				'The given Node has no parent.',
				DOMExceptionNameEnum.invalidNodeTypeError
			);
		}
		this.setEnd(
			node[PropertySymbol.parentNode],
			(<Node>node[PropertySymbol.parentNode])[PropertySymbol.nodeArray].indexOf(node) + 1
		);
	}

	/**
	 * Sets the end position of a Range relative to another Node.
	 *
	 * @see https://dom.spec.whatwg.org/#dom-range-setendbefore
	 * @param node Reference node.
	 */
	public setEndBefore(node: Node): void {
		if (!node[PropertySymbol.parentNode]) {
			throw new this[PropertySymbol.window].DOMException(
				'The given Node has no parent.',
				DOMExceptionNameEnum.invalidNodeTypeError
			);
		}
		this.setEnd(
			node[PropertySymbol.parentNode],
			(<Node>node[PropertySymbol.parentNode])[PropertySymbol.nodeArray].indexOf(node)
		);
	}

	/**
	 * Sets the start position of a Range relative to a Node.
	 *
	 * @see https://dom.spec.whatwg.org/#dom-range-setstartafter
	 * @param node Reference node.
	 */
	public setStartAfter(node: Node): void {
		if (!node[PropertySymbol.parentNode]) {
			throw new this[PropertySymbol.window].DOMException(
				'The given Node has no parent.',
				DOMExceptionNameEnum.invalidNodeTypeError
			);
		}
		this.setStart(
			node[PropertySymbol.parentNode],
			(<Node>node[PropertySymbol.parentNode])[PropertySymbol.nodeArray].indexOf(node) + 1
		);
	}

	/**
	 * Sets the start position of a Range relative to another Node.
	 *
	 * @see https://dom.spec.whatwg.org/#dom-range-setstartbefore
	 * @param node Reference node.
	 */
	public setStartBefore(node: Node): void {
		if (!node[PropertySymbol.parentNode]) {
			throw new this[PropertySymbol.window].DOMException(
				'The given Node has no parent.',
				DOMExceptionNameEnum.invalidNodeTypeError
			);
		}
		this.setStart(
			node[PropertySymbol.parentNode],
			(<Node>node[PropertySymbol.parentNode])[PropertySymbol.nodeArray].indexOf(node)
		);
	}

	/**
	 * Moves content of the Range into a new node, placing the new node at the start of the specified range.
	 *
	 * @see https://dom.spec.whatwg.org/#dom-range-surroundcontents
	 * @param newParent New parent.
	 */
	public surroundContents(newParent: Node): void {
		let node = this.commonAncestorContainer;
		const endNode = NodeUtility.nextDescendantNode(node);
		while (node !== endNode) {
			if (
				node[PropertySymbol.nodeType] !== NodeTypeEnum.textNode &&
				RangeUtility.isPartiallyContained(node, this)
			) {
				throw new this[PropertySymbol.window].DOMException(
					'The Range has partially contains a non-Text node.',
					DOMExceptionNameEnum.invalidStateError
				);
			}

			node = NodeUtility.following(node);
		}

		if (
			newParent[PropertySymbol.nodeType] === NodeTypeEnum.documentNode ||
			newParent[PropertySymbol.nodeType] === NodeTypeEnum.documentTypeNode ||
			newParent[PropertySymbol.nodeType] === NodeTypeEnum.documentFragmentNode
		) {
			throw new this[PropertySymbol.window].DOMException(
				'Invalid element type.',
				DOMExceptionNameEnum.invalidNodeTypeError
			);
		}

		const fragment = this.extractContents();

		while (newParent.firstChild) {
			newParent.removeChild(newParent.firstChild);
		}

		this.insertNode(newParent);

		newParent.appendChild(fragment);

		this.selectNode(newParent);
	}

	/**
	 * Returns the text of the Range.
	 *
	 * @see https://dom.spec.whatwg.org/#dom-range-stringifier
	 */
	public toString(): string {
		const startOffset = this.startOffset;
		const endOffset = this.endOffset;
		let string = '';

		if (
			this[PropertySymbol.start].node === this[PropertySymbol.end].node &&
			this[PropertySymbol.start].node[PropertySymbol.nodeType] === NodeTypeEnum.textNode
		) {
			return (<Text>this[PropertySymbol.start].node).data.slice(startOffset, endOffset);
		}

		if (this[PropertySymbol.start].node[PropertySymbol.nodeType] === NodeTypeEnum.textNode) {
			string += (<Text>this[PropertySymbol.start].node).data.slice(startOffset);
		}

		const endNode = NodeUtility.nextDescendantNode(this[PropertySymbol.end].node);
		let currentNode = this[PropertySymbol.start].node;

		while (currentNode && currentNode !== endNode) {
			if (
				currentNode[PropertySymbol.nodeType] === NodeTypeEnum.textNode &&
				RangeUtility.isContained(currentNode, this)
			) {
				string += (<Text>currentNode).data;
			}

			currentNode = NodeUtility.following(currentNode);
		}

		if (this[PropertySymbol.end].node[PropertySymbol.nodeType] === NodeTypeEnum.textNode) {
			string += (<Text>this[PropertySymbol.end].node).data.slice(0, endOffset);
		}

		return string;
	}
}
