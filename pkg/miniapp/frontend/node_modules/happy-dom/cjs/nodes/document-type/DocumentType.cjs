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
const Node_js_1 = __importDefault(require("../node/Node.cjs"));
const PropertySymbol = __importStar(require("../../PropertySymbol.cjs"));
const NodeTypeEnum_js_1 = __importDefault(require("../node/NodeTypeEnum.cjs"));
/**
 * DocumentType.
 */
class DocumentType extends Node_js_1.default {
    [PropertySymbol.nodeType] = NodeTypeEnum_js_1.default.documentTypeNode;
    [PropertySymbol.name] = '';
    [PropertySymbol.publicId] = '';
    [PropertySymbol.systemId] = '';
    /**
     * Returns name.
     *
     * @returns Name.
     */
    get name() {
        return this[PropertySymbol.name];
    }
    /**
     * Returns public ID.
     *
     * @returns Public ID.
     */
    get publicId() {
        return this[PropertySymbol.publicId];
    }
    /**
     * Returns system ID.
     *
     * @returns System ID.
     */
    get systemId() {
        return this[PropertySymbol.systemId];
    }
    /**
     * Node name.
     *
     * @returns Node name.
     */
    get nodeName() {
        return this.name;
    }
    /**
     * Converts to string.
     *
     * @returns String.
     */
    toString() {
        return '[object DocumentType]';
    }
    /**
     * @override
     */
    [PropertySymbol.cloneNode](deep = false) {
        const clone = super[PropertySymbol.cloneNode](deep);
        clone[PropertySymbol.name] = this[PropertySymbol.name];
        clone[PropertySymbol.publicId] = this[PropertySymbol.publicId];
        clone[PropertySymbol.systemId] = this[PropertySymbol.systemId];
        return clone;
    }
}
exports.default = DocumentType;
//# sourceMappingURL=DocumentType.cjs.map