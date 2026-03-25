import DocumentType from '../nodes/document-type/DocumentType.js';
import Document from '../nodes/document/Document.js';
/**
 * The DOMImplementation interface represents an object providing methods which are not dependent on any particular document. Such an object is returned by the.
 */
export default class DOMImplementation {
    #private;
    /**
     * Constructor.
     *
     * @param document Document.
     */
    constructor(document: Document);
    /**
     * Creates and returns an XML Document.
     *
     * TODO: Not fully implemented.
     *
     * @param _namespaceURI Namespace URI.
     * @param _qualifiedName Qualified name.
     * @param [_docType] Document type.
     */
    createDocument(_namespaceURI: string, _qualifiedName: string, _docType?: string): Document;
    /**
     * Creates and returns an HTML Document.
     */
    createHTMLDocument(): Document;
    /**
     * Creates and returns an HTML Document.
     *
     * @param qualifiedName Qualified name.
     * @param publicId Public ID.
     * @param systemId System ID.
     */
    createDocumentType(qualifiedName: string, publicId: string, systemId: string): DocumentType;
}
//# sourceMappingURL=DOMImplementation.d.ts.map