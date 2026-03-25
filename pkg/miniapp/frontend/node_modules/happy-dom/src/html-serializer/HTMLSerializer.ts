import Element from '../nodes/element/Element.js';
import * as PropertySymbol from '../PropertySymbol.js';
import Node from '../nodes/node/Node.js';
import DocumentType from '../nodes/document-type/DocumentType.js';
import HTMLTemplateElement from '../nodes/html-template-element/HTMLTemplateElement.js';
import NodeTypeEnum from '../nodes/node/NodeTypeEnum.js';
import ProcessingInstruction from '../nodes/processing-instruction/ProcessingInstruction.js';
import DocumentFragment from '../nodes/document-fragment/DocumentFragment.js';
import ShadowRoot from '../nodes/shadow-root/ShadowRoot.js';
import HTMLElementConfig from '../config/HTMLElementConfig.js';
import HTMLElementConfigContentModelEnum from '../config/HTMLElementConfigContentModelEnum.js';
import XMLEncodeUtility from '../utilities/XMLEncodeUtility.js';

/**
 * Serializes a node into HTML.
 */
export default class HTMLSerializer {
	private options: {
		serializableShadowRoots: boolean;
		shadowRoots: ShadowRoot[] | null;
		allShadowRoots: boolean;
	} = {
		serializableShadowRoots: false,
		shadowRoots: null,
		allShadowRoots: false
	};

	/**
	 * Constructor.
	 *
	 * @param [options] Options.
	 * @param [options.serializableShadowRoots] If shadow roots should be serialized.
	 * @param [options.shadowRoots] Shadow roots to serialize.
	 * @param [options.allShadowRoots] If all shadow roots should be serialized.
	 */
	constructor(options?: {
		serializableShadowRoots?: boolean;
		shadowRoots?: ShadowRoot[] | null;
		allShadowRoots?: boolean;
	}) {
		if (options) {
			if (options.serializableShadowRoots) {
				this.options.serializableShadowRoots = options.serializableShadowRoots;
			}

			if (options.shadowRoots) {
				this.options.shadowRoots = options.shadowRoots;
			}

			if (options.allShadowRoots) {
				this.options.allShadowRoots = options.allShadowRoots;
			}
		}
	}

	/**
	 * Renders an element as HTML.
	 *
	 * @param root Root element.
	 * @returns Result.
	 */
	public serializeToString(root: Node): string {
		switch (root[PropertySymbol.nodeType]) {
			case NodeTypeEnum.elementNode:
				const element = <Element>root;
				const prefix = element[PropertySymbol.prefix];
				const localName = element[PropertySymbol.localName];
				const config = HTMLElementConfig[element[PropertySymbol.localName]];
				const tagName = prefix ? `${prefix}:${localName}` : localName;

				if (config?.contentModel === HTMLElementConfigContentModelEnum.noDescendants) {
					return `<${tagName}${this.getAttributes(element)}>`;
				}

				let innerHTML = '';

				// TODO: Should we include closed shadow roots? We are currently only including open shadow roots.
				if (
					element.shadowRoot &&
					(this.options.allShadowRoots ||
						(this.options.serializableShadowRoots &&
							element.shadowRoot[PropertySymbol.serializable]) ||
						this.options.shadowRoots?.includes(element.shadowRoot))
				) {
					innerHTML += `<template shadowrootmode="${element.shadowRoot[PropertySymbol.mode]}"${
						element.shadowRoot[PropertySymbol.serializable] ? ' shadowrootserializable=""' : ''
					}>`;

					for (const node of (<ShadowRoot>element.shadowRoot)[PropertySymbol.nodeArray]) {
						innerHTML += this.serializeToString(node);
					}

					innerHTML += '</template>';
				}

				const childNodes =
					tagName === 'template'
						? (<DocumentFragment>(<HTMLTemplateElement>root).content)[PropertySymbol.nodeArray]
						: (<DocumentFragment>root)[PropertySymbol.nodeArray];

				for (const node of childNodes) {
					innerHTML += this.serializeToString(node);
				}

				return `<${tagName}${this.getAttributes(element)}>${innerHTML}</${tagName}>`;
			case Node.DOCUMENT_FRAGMENT_NODE:
			case Node.DOCUMENT_NODE:
				let html = '';
				for (const node of (<Node>root)[PropertySymbol.nodeArray]) {
					html += this.serializeToString(node);
				}
				return html;
			case NodeTypeEnum.commentNode:
				return `<!--${root.textContent}-->`;
			case NodeTypeEnum.processingInstructionNode:
				return `<!--?${(<ProcessingInstruction>root).target} ${root.textContent}?-->`;
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
	 * Returns attributes as a string.
	 *
	 * @param element Element.
	 * @returns Attributes.
	 */
	private getAttributes(element: Element): string {
		let attributeString = '';

		const namedItems = (<Element>element)[PropertySymbol.attributes][PropertySymbol.namedItems];

		if (!namedItems.has('is') && element[PropertySymbol.isValue]) {
			attributeString +=
				' is="' + XMLEncodeUtility.encodeHTMLAttributeValue(element[PropertySymbol.isValue]) + '"';
		}

		for (const attributes of namedItems.values()) {
			const escapedValue = XMLEncodeUtility.encodeHTMLAttributeValue(
				attributes[0][PropertySymbol.value]
			);
			attributeString += ' ' + attributes[0][PropertySymbol.name] + '="' + escapedValue + '"';
		}

		return attributeString;
	}
}
