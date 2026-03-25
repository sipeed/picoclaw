import Document from '../nodes/document/Document.js';
import * as PropertySymbol from '../PropertySymbol.js';
import NamespaceURI from '../config/NamespaceURI.js';
import HTMLScriptElement from '../nodes/html-script-element/HTMLScriptElement.js';
import Element from '../nodes/element/Element.js';
import HTMLLinkElement from '../nodes/html-link-element/HTMLLinkElement.js';
import Node from '../nodes/node/Node.js';
import DocumentFragment from '../nodes/document-fragment/DocumentFragment.js';
import HTMLElementConfig from '../config/HTMLElementConfig.js';
import HTMLElementConfigContentModelEnum from '../config/HTMLElementConfigContentModelEnum.js';
import SVGElementConfig from '../config/SVGElementConfig.js';
import StringUtility from '../utilities/StringUtility.js';
import BrowserWindow from '../window/BrowserWindow.js';
import DocumentType from '../nodes/document-type/DocumentType.js';
import HTMLHeadElement from '../nodes/html-head-element/HTMLHeadElement.js';
import HTMLBodyElement from '../nodes/html-body-element/HTMLBodyElement.js';
import HTMLHtmlElement from '../nodes/html-html-element/HTMLHtmlElement.js';
import XMLEncodeUtility from '../utilities/XMLEncodeUtility.js';
import NodeTypeEnum from '../nodes/node/NodeTypeEnum.js';

/**
 * Markup RegExp.
 *
 * Group 1: Beginning of start tag (e.g. "div" in "<div").
 * Group 2: End tag (e.g. "div" in "</div>").
 * Group 3: Comment start tag "<!--"
 * Group 4: Comment end tag "-->"
 * Group 5: Document type start tag "<!"
 * Group 6: Processing instruction start tag "<?"
 * Group 7: End of self closing start tag (e.g. "/>" in "<img/>").
 * Group 8: End of start tag or comment tag (e.g. ">" in "<div>").
 */
const MARKUP_REGEXP = /<([^\s/!>?]+)|<\/([^\s/!>?]+)\s*>|(<!--)|(-->|--!>)|(<!)|(<\?)|(\/>)|(>)/gm;

/**
 * Attribute RegExp.
 *
 * Group 1: Attribute name when the attribute has a value with no apostrophes (e.g. "name" in "<div name=value>").
 * Group 2: Attribute value when the attribute has a value with no apostrophes (e.g. "value" in "<div name="value">").
 * Group 3: Attribute name when the attribute has a value using double apostrophe (e.g. "name" in "<div name="value">").
 * Group 4: Attribute value when the attribute has a value using double apostrophe (e.g. "value" in "<div name="value">").
 * Group 5: Attribute end apostrophe when the attribute has a value using double apostrophe (e.g. '"' in "<div name="value">").
 * Group 6: Attribute name when the attribute has a value using single apostrophe (e.g. "name" in "<div name='value'>").
 * Group 7: Attribute value when the attribute has a value using single apostrophe (e.g. "value" in "<div name='value'>").
 * Group 8: Attribute end apostrophe when the attribute has a value using single apostrophe (e.g. "'" in "<div name='value'>").
 * Group 9: Attribute name when the attribute has no value (e.g. "disabled" in "<div disabled>").
 */
const ATTRIBUTE_REGEXP =
	/\s*([a-zA-Z0-9-_:.$@?\\<\[\]]+)\s*=\s*([a-zA-Z0-9-_:.$@?{}/<]+)|\s*([a-zA-Z0-9-_:.$@?\\<\[\]]+)\s*=\s*"([^"]*)("{0,1})|\s*([a-zA-Z0-9-_:.$@?\\<\[\]]+)\s*=\s*'([^']*)('{0,1})|\s*([a-zA-Z0-9-_:.$@?\\<\[\]]+)/gm;

/**
 * Document type attribute RegExp.
 *
 * Group 1: Attribute value.
 */
const DOCUMENT_TYPE_ATTRIBUTE_REGEXP = /"([^"]+)"/gm;

/**
 * Space RegExp.
 */
const SPACE_REGEXP = /\s+/;

/**
 * Space in the beginning of string RegExp.
 */
const SPACE_IN_BEGINNING_REGEXP = /^\s+/;

/**
 * Markup read state (which state the parser is in).
 */
enum MarkupReadStateEnum {
	any = 'any',
	startTag = 'startTag',
	comment = 'comment',
	documentType = 'documentType',
	processingInstruction = 'processingInstruction',
	rawTextElement = 'rawTextElement'
}

/**
 * Document type.
 */
interface IDocumentType {
	name: string;
	publicId: string;
	systemId: string;
}

/**
 * How much of the HTML document that has been parsed (where the parser level is).
 */
enum HTMLDocumentStructureLevelEnum {
	root = 0,
	doctype = 1,
	documentElement = 2,
	head = 3,
	additionalHeadWithoutBody = 4,
	body = 5,
	afterBody = 6
}

interface IHTMLDocumentStructure {
	nodes: {
		doctype: DocumentType;
		documentElement: HTMLHtmlElement;
		head: HTMLHeadElement;
		body: HTMLBodyElement;
	};
	level: HTMLDocumentStructureLevelEnum;
}

/**
 * HTML parser.
 */
export default class HTMLParser {
	private window: BrowserWindow;
	private evaluateScripts: boolean = false;
	private rootNode: Element | DocumentFragment | Document | null = null;
	private rootDocument: Document | null = null;
	private nodeStack: Node[] = [];
	private tagNameStack: string[] = [];
	private documentStructure: IHTMLDocumentStructure | null = null;
	private startTagIndex = 0;
	private markupRegExp: RegExp | null = null;
	private nextElement: Element | null = null;
	private currentNode: Node | null = null;
	private readState: MarkupReadStateEnum = MarkupReadStateEnum.any;

	/**
	 * Constructor.
	 *
	 * @param window Window.
	 * @param [options] Options.
	 * @param [options.evaluateScripts] Set to "true" to enable script execution
	 */
	constructor(
		window: BrowserWindow,
		options?: {
			evaluateScripts?: boolean;
		}
	) {
		this.window = window;

		if (options?.evaluateScripts) {
			this.evaluateScripts = true;
		}
	}
	/**
	 * Parses HTML a root element containing nodes found.
	 *
	 * @param html HTML string.
	 * @param [rootNode] Root node.
	 * @returns Root node.
	 */
	public parse(
		html: string,
		rootNode?: Element | DocumentFragment | Document
	): Element | DocumentFragment | Document {
		this.rootNode = rootNode || this.window.document.createDocumentFragment();
		this.rootDocument = this.rootNode instanceof Document ? this.rootNode : this.window.document;
		this.nodeStack = [this.rootNode];
		this.tagNameStack = [null];
		this.currentNode = this.rootNode;
		this.readState = <MarkupReadStateEnum>MarkupReadStateEnum.any;
		this.documentStructure = null;
		this.startTagIndex = 0;
		this.markupRegExp = new RegExp(MARKUP_REGEXP, 'gm');

		if (this.rootNode instanceof Document) {
			const { doctype, documentElement, head, body } = <Document>this.rootNode;

			if (!documentElement || !head || !body) {
				throw new Error(
					'Failed to parse HTML: The root node must have "documentElement", "head" and "body".\n\nWe should not end up here and it is therefore a bug in Happy DOM. Please report this issue.'
				);
			}

			this.documentStructure = {
				nodes: {
					doctype: doctype || null,
					documentElement,
					head,
					body
				},
				level: HTMLDocumentStructureLevelEnum.root
			};
		}

		if (this.rootNode instanceof this.window.HTMLHtmlElement) {
			const head = this.rootDocument.createElement('head');
			const body = this.rootDocument.createElement('body');
			while (this.rootNode[PropertySymbol.nodeArray].length > 0) {
				this.rootNode[PropertySymbol.removeChild](
					this.rootNode[PropertySymbol.nodeArray][
						this.rootNode[PropertySymbol.nodeArray].length - 1
					]
				);
			}

			this.rootNode[PropertySymbol.appendChild](head);
			this.rootNode[PropertySymbol.appendChild](body);

			this.documentStructure = {
				nodes: {
					doctype: null,
					documentElement: this.rootNode,
					head,
					body
				},
				level: HTMLDocumentStructureLevelEnum.documentElement
			};
		}

		let match: RegExpExecArray;
		let lastIndex = 0;

		html = String(html);

		while ((match = this.markupRegExp.exec(html))) {
			switch (this.readState) {
				case MarkupReadStateEnum.any:
					// Plain text between tags.
					if (
						match.index !== lastIndex &&
						(match[1] || match[2] || match[3] || match[4] || match[5] !== undefined || match[6])
					) {
						this.parsePlainText(html.substring(lastIndex, match.index));
					}

					if (match[1]) {
						// Start tag.
						this.nextElement = this.getStartTagElement(match[1]);

						this.startTagIndex = this.markupRegExp.lastIndex;
						this.readState = MarkupReadStateEnum.startTag;
					} else if (match[2]) {
						// End tag.
						this.parseEndTag(match[2]);
					} else if (match[3]) {
						// Comment.
						this.startTagIndex = this.markupRegExp.lastIndex;
						this.readState = MarkupReadStateEnum.comment;
					} else if (match[5] !== undefined) {
						// Document type.
						this.startTagIndex = this.markupRegExp.lastIndex;
						this.readState = MarkupReadStateEnum.documentType;
					} else if (match[6]) {
						// Processing instruction.
						this.startTagIndex = this.markupRegExp.lastIndex;
						this.readState = MarkupReadStateEnum.processingInstruction;
					} else {
						// Plain text between tags, including the matched tag as it is not a valid start or end tag.
						this.parsePlainText(html.substring(lastIndex, this.markupRegExp.lastIndex));
					}

					break;
				case MarkupReadStateEnum.startTag:
					// End of start tag

					// match[2] is matching an end tag in case the start tag wasn't closed (e.g. "<div\n</ul>" instead of "<div>\n</ul>").
					// match[7] is matching "/>" (e.g. "<img/>").
					// match[8] is matching ">" (e.g. "<div>").
					if (match[7] || match[8] || match[2]) {
						if (this.nextElement) {
							const attributeString = html.substring(
								this.startTagIndex,
								match[2] ? this.markupRegExp.lastIndex - 1 : match.index
							);
							const isSelfClosed = !!match[7];

							this.parseEndOfStartTag(attributeString, isSelfClosed);
						} else {
							// If "nextElement" is set to null, the tag is not allowed (<html>, <head> and <body> are not allowed in an HTML fragment or to be nested).
							this.readState = MarkupReadStateEnum.any;
						}
					}
					break;
				case MarkupReadStateEnum.comment:
					// Comment end tag.

					if (match[4]) {
						this.parseComment(html.substring(this.startTagIndex, match.index));
					}
					break;
				case MarkupReadStateEnum.documentType:
					// Document type end tag.

					if (match[7] || match[8]) {
						this.parseDocumentType(html.substring(this.startTagIndex, match.index));
					}
					break;
				case MarkupReadStateEnum.processingInstruction:
					// Processing instruction end tag.

					if (match[7] || match[8]) {
						// Processing instructions are not supported in HTML and are rendered as comments.
						this.parseComment('?' + html.substring(this.startTagIndex, match.index));
					}
					break;
				case MarkupReadStateEnum.rawTextElement:
					// End tag of raw text content.

					// <script> and <style> elements are raw text elements.

					if (match[2]) {
						this.parseRawTextElementContent(
							match[2],
							html.substring(this.startTagIndex, match.index)
						);
					}
					break;
			}

			lastIndex = this.markupRegExp.lastIndex;
		}

		// Plain text after tags.

		if (lastIndex !== html.length && this.currentNode) {
			this.parsePlainText(html.substring(lastIndex));
		}

		return this.rootNode;
	}

	/**
	 * Parses plain text.
	 *
	 * @param text Text.
	 */
	private parsePlainText(text: string): void {
		if (this.documentStructure) {
			const level = this.documentStructure.level;
			const { documentElement, head, body } = this.documentStructure.nodes;

			// We should remove space in beginning inside the document and the <html> tag.
			const htmlText =
				(this.currentNode === this.rootNode || this.currentNode === documentElement) &&
				level < HTMLDocumentStructureLevelEnum.head &&
				body[PropertySymbol.elementArray].length === 0
					? text.replace(SPACE_IN_BEGINNING_REGEXP, '')
					: text;

			if (htmlText) {
				const textNode = this.rootDocument.createTextNode(
					XMLEncodeUtility.decodeHTMLEntities(htmlText)
				);
				if (
					this.currentNode === head &&
					level === HTMLDocumentStructureLevelEnum.additionalHeadWithoutBody
				) {
					documentElement[PropertySymbol.insertBefore](textNode, body, true);
				} else if (
					this.currentNode === this.rootNode ||
					this.currentNode === documentElement ||
					(this.currentNode === head && level >= HTMLDocumentStructureLevelEnum.body)
				) {
					if (level === HTMLDocumentStructureLevelEnum.head) {
						// Space between <head> and <body> is allowed
						documentElement[PropertySymbol.insertBefore](textNode, body, true);
					} else if (body.lastChild?.[PropertySymbol.nodeType] === NodeTypeEnum.textNode) {
						// If the last child of the body is a text node, we should append the text to it.
						body.lastChild[PropertySymbol.data] += text;
					} else {
						// Nodes outside <html>, <head> and <body> should be appended to the body
						body[PropertySymbol.appendChild](textNode, true);
					}
				} else {
					this.currentNode[PropertySymbol.appendChild](textNode, true);
				}
			}
		} else {
			const textNode = this.rootDocument.createTextNode(XMLEncodeUtility.decodeHTMLEntities(text));
			this.currentNode[PropertySymbol.appendChild](textNode, true);
		}
	}

	/**
	 * Parses end of start tag.
	 *
	 * @param attributeString Attribute string.
	 * @param isSelfClosed Is self closed.
	 */
	private parseEndOfStartTag(attributeString: string, isSelfClosed: boolean): void {
		if (
			attributeString &&
			(!this.documentStructure ||
				this.nextElement !== this.documentStructure.nodes.head ||
				this.documentStructure.level < HTMLDocumentStructureLevelEnum.body)
		) {
			const attributeRegexp = new RegExp(ATTRIBUTE_REGEXP, 'gm');
			let attributeMatch: RegExpExecArray;

			while ((attributeMatch = attributeRegexp.exec(attributeString))) {
				if (
					(attributeMatch[1] && attributeMatch[2]) ||
					(attributeMatch[3] && attributeMatch[5] === '"') ||
					(attributeMatch[6] && attributeMatch[8] === "'") ||
					attributeMatch[9]
				) {
					// Valid attribute name and value.
					const name =
						attributeMatch[1] || attributeMatch[3] || attributeMatch[6] || attributeMatch[9] || '';
					const rawValue = attributeMatch[2] || attributeMatch[4] || attributeMatch[7] || '';
					const value = rawValue ? XMLEncodeUtility.decodeHTMLAttributeValue(rawValue) : '';
					const attributes = this.nextElement[PropertySymbol.attributes];

					if (this.nextElement[PropertySymbol.namespaceURI] === NamespaceURI.svg) {
						// In the SVG namespace, the attribute "xmlns" should be set to the "http://www.w3.org/2000/xmlns/" namespace.
						const namespaceURI = name === 'xmlns' ? NamespaceURI.xmlns : null;

						if (!attributes.getNamedItemNS(namespaceURI, name)) {
							const attributeItem = this.rootDocument.createAttributeNS(namespaceURI, name);
							attributeItem[PropertySymbol.value] = value;
							attributes[PropertySymbol.setNamedItem](attributeItem);
						}
					} else if (!attributes.getNamedItem(name)) {
						const attributeItem = this.rootDocument.createAttribute(name);
						attributeItem[PropertySymbol.value] = value;
						attributes[PropertySymbol.setNamedItem](attributeItem);
					}

					this.startTagIndex += attributeMatch[0].length;
				} else if (
					!attributeMatch[1] &&
					((attributeMatch[3] && !attributeMatch[5]) || (attributeMatch[6] && !attributeMatch[8]))
				) {
					// End attribute apostrophe is missing (e.g. "attr='value" or 'attr="value').
					// We should continue to the next end of start tag match.
					return;
				}
			}
		}

		const tagName = this.nextElement[PropertySymbol.tagName];
		const lowerTagName = tagName.toLowerCase();
		const config = HTMLElementConfig[lowerTagName];
		let previousCurrentNode: Node | null = null;

		while (previousCurrentNode !== this.rootNode) {
			const parentLowerTagName = this.currentNode[PropertySymbol.tagName]?.toLowerCase();
			const parentConfig = HTMLElementConfig[parentLowerTagName];

			if (previousCurrentNode === this.currentNode) {
				throw new Error(
					'Failed to parse HTML: The parser is stuck in an infinite loop. Please report this issue.'
				);
			}

			previousCurrentNode = this.currentNode;

			// Some elements are not allowed to be nested (e.g. "<a><a></a></a>" is not allowed.).
			// Therefore we need to auto-close tags with the same name matching the config, so that it become valid (e.g. "<a></a><a></a>").
			if (
				(config?.contentModel === HTMLElementConfigContentModelEnum.noFirstLevelSelfDescendants &&
					this.tagNameStack[this.tagNameStack.length - 1] === tagName) ||
				parentConfig?.contentModel === HTMLElementConfigContentModelEnum.textOrComments ||
				(parentConfig?.contentModel ===
					HTMLElementConfigContentModelEnum.noForbiddenFirstLevelDescendants &&
					parentConfig?.forbiddenDescendants?.includes(lowerTagName)) ||
				(parentConfig?.contentModel === HTMLElementConfigContentModelEnum.permittedDescendants &&
					!parentConfig?.permittedDescendants?.includes(lowerTagName) &&
					(!config ||
						!config.addPermittedParent ||
						(HTMLElementConfig[config.addPermittedParent].permittedParents &&
							!HTMLElementConfig[config.addPermittedParent].permittedParents.includes(
								parentLowerTagName
							)) ||
						(HTMLElementConfig[config.addPermittedParent].permittedDescendants &&
							!HTMLElementConfig[config.addPermittedParent].permittedDescendants.includes(
								lowerTagName
							))))
			) {
				// We need to move forbidden elements inside <table> outside of the table if possible.
				// E.g. "<table><div><tr><td></td></tr></div></table>"" should become "<div></div><table><tbody><tr><td></td></tr></tbody></table>" (<tbody> is added as <tr> has addPermittedParent as config).
				if (
					parentConfig?.contentModel === HTMLElementConfigContentModelEnum.permittedDescendants &&
					parentConfig.moveForbiddenDescendant &&
					!parentConfig.moveForbiddenDescendant.exclude.includes(lowerTagName)
				) {
					// We add the element before the first element that is not forbidden.
					let before: Node | null = this.currentNode;
					while (before) {
						if (
							!before.parentNode ||
							!HTMLElementConfig[before.parentNode[PropertySymbol.localName]]
								?.permittedDescendants ||
							HTMLElementConfig[
								before.parentNode[PropertySymbol.localName]
							]?.permittedDescendants?.includes(lowerTagName)
						) {
							break;
						} else {
							before = before.parentNode;
						}
					}
					if (before && before.parentNode) {
						before.parentNode.insertBefore(this.nextElement, before);
					} else {
						// If there is no element that is not forbidden, we append the element
						before.appendChild(this.nextElement);
					}
					this.startTagIndex = this.markupRegExp.lastIndex;
					this.readState = MarkupReadStateEnum.any;
					return;
				}
				// This will close the current node
				// E.g. "<a><a></a></a>" will become "<a></a><a></a>"
				this.nodeStack.pop();
				this.tagNameStack.pop();
				this.currentNode = this.nodeStack[this.nodeStack.length - 1] || this.rootNode;
			} else if (
				config?.contentModel === HTMLElementConfigContentModelEnum.noSelfDescendants &&
				this.tagNameStack.includes(tagName)
			) {
				while (this.currentNode !== this.rootNode) {
					if ((<Element>this.currentNode)[PropertySymbol.tagName] === tagName) {
						this.nodeStack.pop();
						this.tagNameStack.pop();
						this.currentNode = this.nodeStack[this.nodeStack.length - 1] || this.rootNode;
						break;
					}
					this.nodeStack.pop();
					this.tagNameStack.pop();
					this.currentNode = this.nodeStack[this.nodeStack.length - 1] || this.rootNode;
				}
			} else if (
				config?.permittedParents &&
				!config.permittedParents.includes(parentLowerTagName)
			) {
				// <thead>, <tbody> and <tfoot> are only allowed as children of <table>.
				// <tr> is only allowed as a child of <table>, <thead>, <tbody> and <tfoot>.
				// <tbody> should be added automatically when <tr> is added directly to the table.
				if (
					!config.addPermittedParent ||
					(HTMLElementConfig[config.addPermittedParent].permittedParents &&
						!HTMLElementConfig[config.addPermittedParent].permittedParents.includes(
							parentLowerTagName
						)) ||
					(HTMLElementConfig[config.addPermittedParent].permittedDescendants &&
						!HTMLElementConfig[config.addPermittedParent].permittedDescendants.includes(
							lowerTagName
						))
				) {
					this.readState = MarkupReadStateEnum.any;
					this.startTagIndex = this.markupRegExp.lastIndex;
					return;
				}

				const permittedParent = this.rootDocument.createElement(config.addPermittedParent);

				this.currentNode[PropertySymbol.appendChild](permittedParent, true);
				this.nodeStack.push(permittedParent);
				this.tagNameStack.push(permittedParent[PropertySymbol.tagName]);
				this.currentNode = permittedParent;
			} else {
				break;
			}
		}

		// Appends the new element to its parent
		if (this.documentStructure) {
			const { documentElement, head, body } = this.documentStructure.nodes;
			const level = this.documentStructure.level;
			// Appends the new node to its parent and sets is as current node.

			// Raw text elements (e.g. <script>) should be appended after the raw text has been added as content to the element.
			// <html>, <head> and <body> are special elements with context constraints. They are already available in the document.
			if (
				(!config || config.contentModel !== HTMLElementConfigContentModelEnum.rawText) &&
				this.nextElement !== documentElement &&
				this.nextElement !== head &&
				this.nextElement !== body
			) {
				// When parser mode is "htmlDocument", any element added directly to the document or document element should be added to the body.
				if (
					documentElement &&
					(this.currentNode === this.rootNode ||
						this.currentNode === documentElement ||
						(this.currentNode === head && level >= HTMLDocumentStructureLevelEnum.body))
				) {
					if (level < HTMLDocumentStructureLevelEnum.body) {
						this.documentStructure.level = HTMLDocumentStructureLevelEnum.afterBody;
					}
					body[PropertySymbol.appendChild](this.nextElement, true);
				} else {
					this.currentNode[PropertySymbol.appendChild](this.nextElement, true);
				}
			}
		} else {
			this.currentNode[PropertySymbol.appendChild](this.nextElement, true);
		}

		// Sets the new element as the current node.

		if (
			!this.documentStructure ||
			this.nextElement !== this.documentStructure.nodes.body ||
			this.documentStructure.level <= HTMLDocumentStructureLevelEnum.body
		) {
			this.currentNode = this.nextElement;
			this.nodeStack.push(this.currentNode);
			this.tagNameStack.push(tagName);

			if (this.documentStructure && this.nextElement === this.documentStructure.nodes.body) {
				this.documentStructure.level = HTMLDocumentStructureLevelEnum.afterBody;
			}

			// Check if the tag is a void element and should be closed immediately.
			// Elements in the SVG namespace can be self-closed (e.g. "/>").
			// "/>" will be ignored in the HTML namespace.
			if (
				config?.contentModel === HTMLElementConfigContentModelEnum.noDescendants ||
				(isSelfClosed &&
					(<Element>this.currentNode)[PropertySymbol.namespaceURI] === NamespaceURI.svg)
			) {
				this.nodeStack.pop();
				this.tagNameStack.pop();
				this.currentNode = this.nodeStack[this.nodeStack.length - 1] || this.rootNode;
				this.readState = MarkupReadStateEnum.any;
			} else {
				// We will set the read state to "rawText" for raw text elements such as <script> and <style> elements.
				this.readState =
					config?.contentModel === HTMLElementConfigContentModelEnum.rawText
						? MarkupReadStateEnum.rawTextElement
						: MarkupReadStateEnum.any;
			}
		} else {
			this.readState = MarkupReadStateEnum.any;
		}

		this.startTagIndex = this.markupRegExp.lastIndex;
	}

	/**
	 * Parses end tag.
	 *
	 * @param tagName Tag name.
	 */
	private parseEndTag(tagName: string): void {
		// SVG elements are case-sensitive.
		const name =
			this.currentNode[PropertySymbol.namespaceURI] === NamespaceURI.html
				? StringUtility.asciiUpperCase(tagName)
				: SVGElementConfig[StringUtility.asciiLowerCase(tagName)]?.localName || tagName;
		const index = this.tagNameStack.lastIndexOf(name);

		// We close all tags up until the first tag that matches the end tag.
		if (index !== -1) {
			this.nodeStack.splice(index, this.nodeStack.length - index);
			this.tagNameStack.splice(index, this.tagNameStack.length - index);
			this.currentNode = this.nodeStack[this.nodeStack.length - 1] || this.rootNode;
		}
	}

	/**
	 * Parses comment.
	 *
	 * @param comment Comment.
	 */
	private parseComment(comment: string): void {
		const commentNode = this.rootDocument.createComment(
			XMLEncodeUtility.decodeHTMLEntities(comment)
		);

		if (this.documentStructure) {
			const level = this.documentStructure.level;
			const { documentElement, head, body } = this.documentStructure.nodes;

			// We need to add the comment node to the correct position in the document.
			let beforeNode: Node | null = null;
			if (this.currentNode === this.rootNode && level === HTMLDocumentStructureLevelEnum.root) {
				beforeNode = documentElement;
			} else if (
				this.currentNode === documentElement &&
				level === HTMLDocumentStructureLevelEnum.documentElement
			) {
				beforeNode = head;
			} else if (
				this.currentNode === documentElement &&
				level === HTMLDocumentStructureLevelEnum.head
			) {
				beforeNode = body;
			}

			this.currentNode[PropertySymbol.insertBefore](commentNode, beforeNode, true);
		} else {
			this.currentNode[PropertySymbol.appendChild](commentNode, true);
		}

		this.readState = MarkupReadStateEnum.any;
	}

	/**
	 * Parses document type.
	 *
	 * @param text Text.
	 */
	private parseDocumentType(text: string): void {
		const decodedText = XMLEncodeUtility.decodeHTMLEntities(text);

		if (this.documentStructure) {
			let { doctype } = this.documentStructure.nodes;
			const documentType = this.getDocumentType(decodedText);
			// Document type nodes are only allowed at the beginning of the document.
			if (documentType) {
				if (
					this.currentNode === this.rootNode &&
					this.documentStructure.level === HTMLDocumentStructureLevelEnum.root
				) {
					if (doctype) {
						doctype[PropertySymbol.name] = documentType.name;
						doctype[PropertySymbol.publicId] = documentType.publicId;
						doctype[PropertySymbol.systemId] = documentType.systemId;
					} else {
						doctype = (<Document>this.rootNode).implementation.createDocumentType(
							documentType.name,
							documentType.publicId,
							documentType.systemId
						);
						(<Document>this.rootNode).insertBefore(
							doctype,
							(<Document>this.rootNode).documentElement
						);
					}

					this.documentStructure.level = HTMLDocumentStructureLevelEnum.doctype;
				}
			} else {
				this.parseComment(decodedText);
			}
		} else {
			// Document type nodes are only allowed at the beginning of the document.
			if (!this.getDocumentType(decodedText)) {
				this.parseComment(decodedText);
			}
		}

		this.readState = MarkupReadStateEnum.any;
	}

	/**
	 * Parses raw text content for elements such as <script> and <style>.
	 *
	 * @param tagName End tag name.
	 * @param text Text.
	 */
	private parseRawTextElementContent(tagName: string, text: string): void {
		const upperTagName = StringUtility.asciiUpperCase(tagName);

		if (upperTagName !== this.currentNode[PropertySymbol.tagName]) {
			return;
		}

		// Scripts are not allowed to be executed when they are parsed using innerHTML, outerHTML, replaceWith() etc.
		// However, they are allowed to be executed when document.write() is used.
		// See: https://developer.mozilla.org/en-US/docs/Web/API/HTMLScriptElement
		if (upperTagName === 'SCRIPT') {
			(<HTMLScriptElement>this.currentNode)[PropertySymbol.evaluateScript] = this.evaluateScripts;
		} else if (upperTagName === 'LINK') {
			// An assumption that the same rule should be applied for the HTMLLinkElement is made here.
			(<HTMLLinkElement>this.currentNode)[PropertySymbol.evaluateCSS] = this.evaluateScripts;
		}

		// Plain text elements such as <script> and <style> should only contain text.
		// Plain text elements should not decode entities. See #1564.
		this.currentNode[PropertySymbol.appendChild](this.rootDocument.createTextNode(text), true);

		const rawTextElement = this.currentNode;

		this.nodeStack.pop();
		this.tagNameStack.pop();
		this.currentNode = this.nodeStack[this.nodeStack.length - 1] || this.rootNode;
		this.readState = MarkupReadStateEnum.any;

		// Appends the raw text element to its parent.

		if (this.documentStructure) {
			const { documentElement, body } = this.documentStructure.nodes;

			// When parser mode is "htmlDocument", any element added directly to the document or document element should be added to the body.
			if (
				documentElement &&
				(this.currentNode === this.rootNode || this.currentNode === documentElement)
			) {
				body[PropertySymbol.appendChild](rawTextElement, true);
			} else {
				this.currentNode[PropertySymbol.appendChild](rawTextElement, true);
			}
		} else {
			this.currentNode[PropertySymbol.appendChild](rawTextElement, true);
		}
	}

	/**
	 * Creates an element or returns a reference to it.
	 *
	 * @param tagName Tag name.
	 */
	private getStartTagElement(tagName: string): Element {
		const lowerTagName = StringUtility.asciiLowerCase(tagName);

		// NamespaceURI is inherited from the parent element.
		const namespaceURI = this.currentNode[PropertySymbol.namespaceURI];

		// NamespaceURI should be SVG when the tag name is "svg" (even in XML mode).
		if (lowerTagName === 'svg') {
			return this.rootDocument.createElementNS(NamespaceURI.svg, 'svg');
		}

		if (namespaceURI === NamespaceURI.svg) {
			return this.rootDocument.createElementNS(
				NamespaceURI.svg,
				SVGElementConfig[lowerTagName]?.localName || tagName
			);
		}

		// New element.
		switch (lowerTagName) {
			case 'html':
				if (!this.documentStructure) {
					return null;
				}
				if (this.documentStructure.level < HTMLDocumentStructureLevelEnum.documentElement) {
					this.documentStructure.level = HTMLDocumentStructureLevelEnum.documentElement;
				}
				return this.documentStructure.nodes.documentElement ?? null;
			case 'head':
				if (!this.documentStructure) {
					return null;
				}
				if (this.documentStructure.level < HTMLDocumentStructureLevelEnum.head) {
					this.documentStructure.level = HTMLDocumentStructureLevelEnum.head;
				} else if (this.documentStructure.level === HTMLDocumentStructureLevelEnum.head) {
					this.documentStructure.level = HTMLDocumentStructureLevelEnum.additionalHeadWithoutBody;
				}
				return this.documentStructure.nodes.head ?? null;
			case 'body':
				if (!this.documentStructure) {
					return null;
				}
				if (this.documentStructure.level < HTMLDocumentStructureLevelEnum.body) {
					this.documentStructure.level = HTMLDocumentStructureLevelEnum.body;
				}
				return this.documentStructure.nodes.body ?? null;
			default:
				return this.rootDocument.createElementNS(NamespaceURI.html, lowerTagName);
		}
	}

	/**
	 * Returns document type.
	 *
	 * @param value Value.
	 * @returns Document type.
	 */
	private getDocumentType(value: string): IDocumentType {
		if (!value.toUpperCase().startsWith('DOCTYPE')) {
			return null;
		}

		const docTypeSplit = value.split(SPACE_REGEXP);

		if (docTypeSplit.length <= 1) {
			return null;
		}

		const docTypeString = docTypeSplit.slice(1).join(' ');
		const attributes = [];
		const attributeRegExp = new RegExp(DOCUMENT_TYPE_ATTRIBUTE_REGEXP, 'gm');
		const isPublic = docTypeString.toUpperCase().includes('PUBLIC');
		let attributeMatch;

		while ((attributeMatch = attributeRegExp.exec(docTypeString))) {
			attributes.push(attributeMatch[1]);
		}

		const publicId = isPublic ? attributes[0] || '' : '';
		const systemId = isPublic ? attributes[1] || '' : attributes[0] || '';

		return {
			name: docTypeSplit[1].toLowerCase(),
			publicId,
			systemId
		};
	}
}
