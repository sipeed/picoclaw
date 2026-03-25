"use strict";
var __createBinding = (this && this.__createBinding) || (Object.create ? (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    var desc = Object.getOwnPropertyDescriptor(m, k);
    if (!desc || ("get" in desc ? !m.__esModule : desc.writable || desc.configurable)) {
      desc = { enumerable: true, get: function() { return m[k]; } };
    }
    Object.defineProperty(o, k2, desc);
}) : (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    o[k2] = m[k];
}));
var __setModuleDefault = (this && this.__setModuleDefault) || (Object.create ? (function(o, v) {
    Object.defineProperty(o, "default", { enumerable: true, value: v });
}) : function(o, v) {
    o["default"] = v;
});
var __importStar = (this && this.__importStar) || function (mod) {
    if (mod && mod.__esModule) return mod;
    var result = {};
    if (mod != null) for (var k in mod) if (k !== "default" && Object.prototype.hasOwnProperty.call(mod, k)) __createBinding(result, mod, k);
    __setModuleDefault(result, mod);
    return result;
};
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const DOMException_js_1 = __importDefault(require("../exception/DOMException.cjs"));
const PropertySymbol = __importStar(require("../PropertySymbol.cjs"));
const DOMExceptionNameEnum_js_1 = __importDefault(require("../exception/DOMExceptionNameEnum.cjs"));
const NodeTypeEnum_js_1 = __importDefault(require("../nodes/node/NodeTypeEnum.cjs"));
const NodeUtility_js_1 = __importDefault(require("../nodes/node/NodeUtility.cjs"));
/**
 * Range utility.
 *
 * Based on:
 * https://github.com/jsdom/jsdom/blob/master/lib/jsdom/living/range/boundary-point.js.
 */
class RangeUtility {
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
    static compareBoundaryPointsPosition(pointA, pointB) {
        if (pointA.node === pointB.node) {
            if (pointA.offset === pointB.offset) {
                return 0;
            }
            else if (pointA.offset < pointB.offset) {
                return -1;
            }
            return 1;
        }
        if (NodeUtility_js_1.default.isFollowing(pointA.node, pointB.node)) {
            return this.compareBoundaryPointsPosition(pointB, pointA) === -1 ? 1 : -1;
        }
        if (NodeUtility_js_1.default.isInclusiveAncestor(pointA.node, pointB.node)) {
            let child = pointB.node;
            while (child[PropertySymbol.parentNode] !== pointA.node) {
                child = child[PropertySymbol.parentNode];
            }
            if (child[PropertySymbol.parentNode][PropertySymbol.nodeArray].indexOf(child) <
                pointA.offset) {
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
    static validateBoundaryPoint(point) {
        if (point.node[PropertySymbol.nodeType] === NodeTypeEnum_js_1.default.documentTypeNode) {
            throw new DOMException_js_1.default(`DocumentType Node can't be used as boundary point.`, DOMExceptionNameEnum_js_1.default.invalidNodeTypeError);
        }
        if (point.offset > NodeUtility_js_1.default.getNodeLength(point.node)) {
            throw new DOMException_js_1.default(`Offset out of bound.`, DOMExceptionNameEnum_js_1.default.indexSizeError);
        }
    }
    /**
     * Returns "true" if contained.
     *
     * @param node Node.
     * @param range Range.
     * @returns "true" if contained.
     */
    static isContained(node, range) {
        return (this.compareBoundaryPointsPosition({ node, offset: 0 }, { node: range.startContainer, offset: range.startOffset }) === 1 &&
            this.compareBoundaryPointsPosition({ node, offset: NodeUtility_js_1.default.getNodeLength(node) }, { node: range.endContainer, offset: range.endOffset }) === -1);
    }
    /**
     * Returns "true" if partially contained.
     *
     * @param node Node.
     * @param range Range.
     * @returns "true" if partially contained.
     */
    static isPartiallyContained(node, range) {
        return ((NodeUtility_js_1.default.isInclusiveAncestor(node, range.startContainer) &&
            !NodeUtility_js_1.default.isInclusiveAncestor(node, range.endContainer)) ||
            (!NodeUtility_js_1.default.isInclusiveAncestor(node, range.startContainer) &&
                NodeUtility_js_1.default.isInclusiveAncestor(node, range.endContainer)));
    }
}
exports.default = RangeUtility;
//# sourceMappingURL=RangeUtility.cjs.map