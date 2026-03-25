import DOMException from '../exception/DOMException.js';
import * as PropertySymbol from '../PropertySymbol.js';
import DOMExceptionNameEnum from '../exception/DOMExceptionNameEnum.js';
import NodeTypeEnum from '../nodes/node/NodeTypeEnum.js';
import Node from '../nodes/node/Node.js';
import NodeUtility from '../nodes/node/NodeUtility.js';
import Range from './Range.js';
import IRangeBoundaryPoint from './IRangeBoundaryPoint.js';

/**
 * Range utility.
 *
 * Based on:
 * https://github.com/jsdom/jsdom/blob/master/lib/jsdom/living/range/boundary-point.js.
 */
export default class RangeUtility {
	/**
	 * Compares boundary points.
	 *
	 * Based on logic from:
	 * https://github.com/jsdom/jsdom/blob/master/lib/jsdom/living/range/boundary-point.js
	 *
	 * @see https://dom.spec.whatwg.org/#concept-range-bp-after
	 * @param pointA Point A.
	 * @param pointB Point B.
	 * @returns A number, -1, 0, or 1, indicating whether the corresponding boundary-point of the Range is respectively before, equal to, or after the corresponding boundary-point of sourceRange.
	 */
	public static compareBoundaryPointsPosition(
		pointA: IRangeBoundaryPoint,
		pointB: IRangeBoundaryPoint
	): number {
		if (pointA.node === pointB.node) {
			if (pointA.offset === pointB.offset) {
				return 0;
			} else if (pointA.offset < pointB.offset) {
				return -1;
			}

			return 1;
		}

		if (NodeUtility.isFollowing(pointA.node, pointB.node)) {
			return this.compareBoundaryPointsPosition(pointB, pointA) === -1 ? 1 : -1;
		}

		if (NodeUtility.isInclusiveAncestor(pointA.node, pointB.node)) {
			let child = pointB.node;

			while (child[PropertySymbol.parentNode] !== pointA.node) {
				child = child[PropertySymbol.parentNode];
			}

			if (
				(<Node>child[PropertySymbol.parentNode])[PropertySymbol.nodeArray].indexOf(child) <
				pointA.offset
			) {
				return 1;
			}
		}

		return -1;
	}

	/**
	 * Validates a boundary point.
	 *
	 * @throws DOMException
	 * @param point Boundary point.
	 */
	public static validateBoundaryPoint(point: IRangeBoundaryPoint): void {
		if (point.node[PropertySymbol.nodeType] === NodeTypeEnum.documentTypeNode) {
			throw new DOMException(
				`DocumentType Node can't be used as boundary point.`,
				DOMExceptionNameEnum.invalidNodeTypeError
			);
		}

		if (point.offset > NodeUtility.getNodeLength(point.node)) {
			throw new DOMException(`Offset out of bound.`, DOMExceptionNameEnum.indexSizeError);
		}
	}

	/**
	 * Returns "true" if contained.
	 *
	 * @param node Node.
	 * @param range Range.
	 * @returns "true" if contained.
	 */
	public static isContained(node: Node, range: Range): boolean {
		return (
			this.compareBoundaryPointsPosition(
				{ node, offset: 0 },
				{ node: range.startContainer, offset: range.startOffset }
			) === 1 &&
			this.compareBoundaryPointsPosition(
				{ node, offset: NodeUtility.getNodeLength(node) },
				{ node: range.endContainer, offset: range.endOffset }
			) === -1
		);
	}

	/**
	 * Returns "true" if partially contained.
	 *
	 * @param node Node.
	 * @param range Range.
	 * @returns "true" if partially contained.
	 */
	public static isPartiallyContained(node: Node, range: Range): boolean {
		return (
			(NodeUtility.isInclusiveAncestor(node, range.startContainer) &&
				!NodeUtility.isInclusiveAncestor(node, range.endContainer)) ||
			(!NodeUtility.isInclusiveAncestor(node, range.startContainer) &&
				NodeUtility.isInclusiveAncestor(node, range.endContainer))
		);
	}
}
