import DocumentType from '../nodes/document-type/DocumentType.js';
import * as PropertySymbol from '../PropertySymbol.js';
import Document from '../nodes/document/Document.js';
import NodeFactory from '../nodes/NodeFactory.js';

/**
 * The DOMImplementation interface represents an object providing methods which are not dependent on any particular document. Such an object is returned by the.
 */
export default class DOMImplementation {
	#document: Document;

	/**
	 * Constructor.
	 *
	 * @param document Document.
	 */
	constructor(document: Document) {
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
	public createDocument(
		_namespaceURI: string,
		_qualifiedName: string,
		_docType?: string
	): Document {
		return new this.#document[PropertySymbol.window].HTMLDocument();
	}

	/**
	 * Creates and returns an HTML Document.
	 */
	public createHTMLDocument(): Document {
		return new this.#document[PropertySymbol.window].HTMLDocument();
	}

	/**
	 * Creates and returns an HTML Document.
	 *
	 * @param qualifiedName Qualified name.
	 * @param publicId Public ID.
	 * @param systemId System ID.
	 */
	public createDocumentType(
		qualifiedName: string,
		publicId: string,
		systemId: string
	): DocumentType {
		const documentType = NodeFactory.createNode<DocumentType>(this.#document, DocumentType);

		documentType[PropertySymbol.name] = qualifiedName;
		documentType[PropertySymbol.publicId] = publicId;
		documentType[PropertySymbol.systemId] = systemId;

		return documentType;
	}
}
