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
const DocumentType_js_1 = __importDefault(require("../nodes/document-type/DocumentType.cjs"));
const PropertySymbol = __importStar(require("../PropertySymbol.cjs"));
const NodeFactory_js_1 = __importDefault(require("../nodes/NodeFactory.cjs"));
/**
 * The DOMImplementation interface represents an object providing methods which are not dependent on any particular document. Such an object is returned by the.
 */
class DOMImplementation {
    #document;
    /**
     * Constructor.
     *
     * @param document Document.
     */
    constructor(document) {
        this.#document = document;
    }
    /**
     * Creates and returns an XML Document.
     *
     * TODO: Not fully implemented.
     *
     * @param _namespaceURI Namespace URI.
     * @param _qualifiedName Qualified name.
     * @param [_docType] Document type.
     */
    createDocument(_namespaceURI, _qualifiedName, _docType) {
        return new this.#document[PropertySymbol.window].HTMLDocument();
    }
    /**
     * Creates and returns an HTML Document.
     */
    createHTMLDocument() {
        return new this.#document[PropertySymbol.window].HTMLDocument();
    }
    /**
     * Creates and returns an HTML Document.
     *
     * @param qualifiedName Qualified name.
     * @param publicId Public ID.
     * @param systemId System ID.
     */
    createDocumentType(qualifiedName, publicId, systemId) {
        const documentType = NodeFactory_js_1.default.createNode(this.#document, DocumentType_js_1.default);
        documentType[PropertySymbol.name] = qualifiedName;
        documentType[PropertySymbol.publicId] = publicId;
        documentType[PropertySymbol.systemId] = systemId;
        return documentType;
    }
}
exports.default = DOMImplementation;
//# sourceMappingURL=DOMImplementation.cjs.map