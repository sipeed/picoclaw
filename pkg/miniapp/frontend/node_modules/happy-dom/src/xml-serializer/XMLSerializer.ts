import Element from '../nodes/element/Element.js';
import * as PropertySymbol from '../PropertySymbol.js';
import Node from '../nodes/node/Node.js';
import DocumentType from '../nodes/document-type/DocumentType.js';
import HTMLTemplateElement from '../nodes/html-template-element/HTMLTemplateElement.js';
import NodeTypeEnum from '../nodes/node/NodeTypeEnum.js';
import DocumentFragment from '../nodes/document-fragment/DocumentFragment.js';
import HTMLElementConfig from '../config/HTMLElementConfig.js';
import HTMLElementConfigContentModelEnum from '../config/HTMLElementConfigContentModelEnum.js';
import ProcessingInstruction from '../nodes/processing-instruction/ProcessingInstruction.js';
import NamespaceURI from '../config/NamespaceURI.js';
import XMLEncodeUtility from '../utilities/XMLEncodeUtility.js';

/**
 * Serializes a node into XML.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/XMLSerializer
 */
export default class XMLSerializer {
	/**
	 * Serializes a node into XML.
	 *
	 * @param root Root node.
	 * @returns Result.
	 */
	public serializeToString(root: Node): string {
		return this.#serializeToString(root);
	}

	/**
	 * Serializes a node into XML.
	 *
	 * @param root Root node.
	 * @param [inheritedDefaultNamespace] Default namespace.
	 * @param [inheritedNamespacePrefixes] Inherited namespace prefixes.
	 * @returns Result.
	 */
	#serializeToString(
		root: Node,
		inheritedDefaultNamespace: string | null = null,
		inheritedNamespacePrefixes: Map<string, string> = null
	): string {
		switch (root[PropertySymbol.nodeType]) {
			case NodeTypeEnum.elementNode:
				const element = <Element>root;
				const localName = element[PropertySymbol.localName];

				let innerHTML = '';

				const namespacePrefixes = this.#getNamespacePrefixes(element, inheritedNamespacePrefixes);
				const elementPrefix =
					element[PropertySymbol.namespaceURI] === NamespaceURI.html
						? element[PropertySymbol.prefix]
						: this.#getElementPrefix(element, namespacePrefixes);
				const tagName = `${elementPrefix ? elementPrefix + ':' : ''}${localName}`;
				const defaultNamespace = elementPrefix
					? inheritedDefaultNamespace
					: element[PropertySymbol.namespaceURI] || inheritedDefaultNamespace;
				const attributes = this.#getAttributes(
					element,
					elementPrefix,
					inheritedDefaultNamespace,
					inheritedNamespacePrefixes
				);

				const childNodes =
					defaultNamespace === NamespaceURI.html && localName === 'template'
						? (<DocumentFragment>(<HTMLTemplateElement>root).content)[PropertySymbol.nodeArray]
						: (<DocumentFragment>root)[PropertySymbol.nodeArray];

				for (const node of childNodes) {
					innerHTML += this.#serializeToString(node, defaultNamespace, namespacePrefixes);
				}

				if (
					!innerHTML &&
					defaultNamespace === NamespaceURI.html &&
					HTMLElementConfig[localName.toLowerCase()]?.contentModel ===
						HTMLElementConfigContentModelEnum.noDescendants
				) {
					return `<${tagName}${attributes} />`;
				}

				if (!innerHTML && defaultNamespace !== NamespaceURI.html) {
					return `<${tagName}${attributes}/>`;
				}

				return `<${tagName}${attributes}>${innerHTML}</${tagName}>`;
			case Node.DOCUMENT_FRAGMENT_NODE:
			case Node.DOCUMENT_NODE:
				let html = '';
				if (root[PropertySymbol.xmlProcessingInstruction]) {
					html += this.#serializeToString(
						root[PropertySymbol.xmlProcessingInstruction],
						inheritedDefaultNamespace,
						new Map(inheritedNamespacePrefixes)
					);
				}
				for (const node of (<Node>root)[PropertySymbol.nodeArray]) {
					html += this.#serializeToString(
						node,
						inheritedDefaultNamespace,
						new Map(inheritedNamespacePrefixes)
					);
				}
				return html;
			case NodeTypeEnum.commentNode:
				return `<!--${root.textContent}-->`;
			case NodeTypeEnum.processingInstructionNode:
				return `<?${(<ProcessingInstruction>root).target} ${root.textContent}?>`;
			case NodeTypeEnum.textNode:
				const parentElement = root.parentElement;
				if (parentElement) {
					const parentConfig = HTMLElementConfig[parentElement[PropertySymbol.localName]];
					if (parentConfig?.contentModel === HTMLElementConfigContentModelEnum.rawText) {
						return root.textContent;
					}
				}
				return XMLEncodeUtility.encodeTextContent(root.textContent);
			case NodeTypeEnum.documentTypeNode:
				const doctype = <DocumentType>root;
				const identifier = doctype.publicId ? ' PUBLIC' : doctype.systemId ? ' SYSTEM' : '';
				const publicId = doctype.publicId ? ` "${doctype.publicId}"` : '';
				const systemId = doctype.systemId ? ` "${doctype.systemId}"` : '';
				return `<!DOCTYPE ${doctype.name}${identifier}${publicId}${systemId}>`;
		}

		return '';
	}

	/**
	 * Returns namespace prefixes.
	 *
	 * @param element Element.
	 * @param inheritedNamespacePrefixes Inherited namespace prefixes.
	 * @returns Namespace prefixes.
	 */
	#getNamespacePrefixes(
		element: Element,
		inheritedNamespacePrefixes: Map<string, string> | null
	): Map<string, string> | null {
		const namespacePrefixes = new Map<string, string>(inheritedNamespacePrefixes);
		const namedItems = (<Element>element)[PropertySymbol.attributes][PropertySymbol.namedItems];

		for (const attributes of namedItems.values()) {
			if (
				attributes[0][PropertySymbol.namespaceURI] === NamespaceURI.xmlns &&
				attributes[0][PropertySymbol.prefix]
			) {
				namespacePrefixes.set(
					attributes[0][PropertySymbol.value],
					attributes[0][PropertySymbol.localName]
				);
			}
		}

		return namespacePrefixes;
	}

	/**
	 * Returns namespace prefixes.
	 *
	 * @param element Element.
	 * @param namespacePrefixes Inherited namespace prefixes.
	 * @returns Element prefix.
	 */
	#getElementPrefix(
		element: Element,
		namespacePrefixes: Map<string, string> | null
	): string | null {
		if (element[PropertySymbol.prefix] && !element[PropertySymbol.namespaceURI]) {
			throw new Error('Element has a prefix but no namespace.');
		}

		if (!element[PropertySymbol.prefix]) {
			return null;
		}

		const elementPrefix = namespacePrefixes.get(element[PropertySymbol.namespaceURI]);

		if (elementPrefix) {
			return elementPrefix;
		}

		const existingPrefixes = new Set(namespacePrefixes.values());

		if (existingPrefixes.has(element[PropertySymbol.prefix])) {
			let i = 1;
			while (existingPrefixes.has('n' + i)) {
				i++;
			}
			namespacePrefixes.set(element[PropertySymbol.namespaceURI], 'n' + i);
			return 'n' + i;
		}

		namespacePrefixes.set(element[PropertySymbol.namespaceURI], element[PropertySymbol.prefix]);

		return element[PropertySymbol.prefix];
	}

	/**
	 * Returns attributes as a string.
	 *
	 * @param element Element.
	 * @param elementPrefix Element prefix.
	 * @param inheritedDefaultNamespace Inherited default namespace.
	 * @param inheritedNamespacePrefixes Inherited namespace prefixes.
	 * @returns Attributes.
	 */
	#getAttributes(
		element: Element,
		elementPrefix: string | null,
		inheritedDefaultNamespace: string | null,
		inheritedNamespacePrefixes: Map<string, string> | null
	): string {
		let attributeString = '';
		let namespaceString = '';

		const namedItems = (<Element>element)[PropertySymbol.attributes][PropertySymbol.namedItems];
		const handledNamespaces = new Set();

		for (const attributes of namedItems.values()) {
			const attribute = attributes[0];

			// Namespace attributes should be in the beginning of the string.
			if (attribute[PropertySymbol.namespaceURI] === NamespaceURI.xmlns) {
				if (
					elementPrefix &&
					attribute[PropertySymbol.localName] === elementPrefix &&
					element[PropertySymbol.namespaceURI]
				) {
					namespaceString += ` xmlns:${elementPrefix}="${XMLEncodeUtility.encodeXMLAttributeValue(
						element[PropertySymbol.namespaceURI]
					)}"`;
					handledNamespaces.add(element[PropertySymbol.namespaceURI]);
				} else if (
					!elementPrefix &&
					attribute[PropertySymbol.name] === 'xmlns' &&
					element[PropertySymbol.namespaceURI]
				) {
					namespaceString += ` xmlns="${XMLEncodeUtility.encodeXMLAttributeValue(
						element[PropertySymbol.namespaceURI]
					)}"`;
					handledNamespaces.add(element[PropertySymbol.namespaceURI]);
				} else {
					namespaceString += ` ${
						attribute[PropertySymbol.name]
					}="${XMLEncodeUtility.encodeXMLAttributeValue(attribute[PropertySymbol.value])}"`;
					handledNamespaces.add(attribute[PropertySymbol.value]);
				}
			} else {
				attributeString += ` ${
					attribute[PropertySymbol.name]
				}="${XMLEncodeUtility.encodeXMLAttributeValue(attribute[PropertySymbol.value])}"`;
			}
		}

		// We should add the namespace as an attribute if it has not been added yet.
		if (
			element[PropertySymbol.namespaceURI] &&
			inheritedDefaultNamespace !== element[PropertySymbol.namespaceURI] &&
			!handledNamespaces.has(element[PropertySymbol.namespaceURI])
		) {
			if (elementPrefix && !inheritedNamespacePrefixes.has(element[PropertySymbol.namespaceURI])) {
				namespaceString += ` xmlns:${elementPrefix}="${XMLEncodeUtility.encodeXMLAttributeValue(
					element[PropertySymbol.namespaceURI]
				)}"`;
			} else if (
				!elementPrefix &&
				inheritedDefaultNamespace !== element[PropertySymbol.namespaceURI]
			) {
				namespaceString += ` xmlns="${XMLEncodeUtility.encodeXMLAttributeValue(
					element[PropertySymbol.namespaceURI]
				)}"`;
			}
		}

		return namespaceString + attributeString;
	}
}
