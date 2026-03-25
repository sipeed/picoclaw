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
Object.defineProperty(exports, "__esModule", { value: true });
const PropertySymbol = __importStar(require("../PropertySymbol.cjs"));
/**
 * Node factory used for setting the owner document to nodes.
 */
class NodeFactory {
    static ownerDocuments = [];
    /**
     * Creates a node instance with the given owner document.
     *
     * @param ownerDocument Owner document.
     * @param nodeClass Node class.
     * @param [args] Node arguments.
     * @returns Node instance.
     */
    static createNode(ownerDocument, nodeClass, ...args) {
        if (!nodeClass.prototype[PropertySymbol.window]) {
            this.ownerDocuments.push(ownerDocument);
        }
        return new nodeClass(...args);
    }
    /**
     * Pulls an owner document from the queue.
     *
     * @returns Document.
     */
    static pullOwnerDocument() {
        return this.ownerDocuments.pop();
    }
}
exports.default = NodeFactory;
//# sourceMappingURL=NodeFactory.cjs.map