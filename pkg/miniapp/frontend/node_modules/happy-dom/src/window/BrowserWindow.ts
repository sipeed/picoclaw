import { Buffer } from 'buffer';
import { webcrypto } from 'crypto';
import { TextEncoder, TextDecoder } from 'util';
import Stream from 'stream';
import { ReadableStream } from 'stream/web';
import { URLSearchParams } from 'url';
import VM from 'vm';
import * as PropertySymbol from '../PropertySymbol.js';
import Base64 from '../base64/Base64.js';
import BrowserErrorCaptureEnum from '../browser/enums/BrowserErrorCaptureEnum.js';
import IBrowserFrame from '../browser/types/IBrowserFrame.js';
import Clipboard from '../clipboard/Clipboard.js';
import ClipboardItem from '../clipboard/ClipboardItem.js';
import CSS from '../css/CSS.js';
import CSSRule from '../css/CSSRule.js';
import CSSStyleSheet from '../css/CSSStyleSheet.js';
import CSSUnitValue from '../css/CSSUnitValue.js';
import CSSStyleDeclaration from '../css/declaration/CSSStyleDeclaration.js';
import CSSContainerRule from '../css/rules/CSSContainerRule.js';
import CSSFontFaceRule from '../css/rules/CSSFontFaceRule.js';
import CSSKeyframeRule from '../css/rules/CSSKeyframeRule.js';
import CSSKeyframesRule from '../css/rules/CSSKeyframesRule.js';
import CSSMediaRule from '../css/rules/CSSMediaRule.js';
import CSSStyleRule from '../css/rules/CSSStyleRule.js';
import CSSSupportsRule from '../css/rules/CSSSupportsRule.js';
import CustomElementRegistry from '../custom-element/CustomElementRegistry.js';
import DOMParser from '../dom-parser/DOMParser.js';
import DataTransfer from '../event/DataTransfer.js';
import DataTransferItem from '../event/DataTransferItem.js';
import DataTransferItemList from '../event/DataTransferItemList.js';
import Event from '../event/Event.js';
import EventTarget from '../event/EventTarget.js';
import MessagePort from '../event/MessagePort.js';
import Touch from '../event/Touch.js';
import UIEvent from '../event/UIEvent.js';
import AnimationEvent from '../event/events/AnimationEvent.js';
import ClipboardEvent from '../event/events/ClipboardEvent.js';
import CustomEvent from '../event/events/CustomEvent.js';
import ErrorEvent from '../event/events/ErrorEvent.js';
import FocusEvent from '../event/events/FocusEvent.js';
import HashChangeEvent from '../event/events/HashChangeEvent.js';
import InputEvent from '../event/events/InputEvent.js';
import KeyboardEvent from '../event/events/KeyboardEvent.js';
import MediaQueryListEvent from '../event/events/MediaQueryListEvent.js';
import MessageEvent from '../event/events/MessageEvent.js';
import MouseEvent from '../event/events/MouseEvent.js';
import PointerEvent from '../event/events/PointerEvent.js';
import ProgressEvent from '../event/events/ProgressEvent.js';
import StorageEvent from '../event/events/StorageEvent.js';
import SubmitEvent from '../event/events/SubmitEvent.js';
import TouchEvent from '../event/events/TouchEvent.js';
import WheelEvent from '../event/events/WheelEvent.js';
import DOMException from '../exception/DOMException.js';
import DOMExceptionNameEnum from '../exception/DOMExceptionNameEnum.js';
import AbortController from '../fetch/AbortController.js';
import AbortSignal from '../fetch/AbortSignal.js';
import Fetch from '../fetch/Fetch.js';
import Headers from '../fetch/Headers.js';
import Request from '../fetch/Request.js';
import Response from '../fetch/Response.js';
import IRequestInfo from '../fetch/types/IRequestInfo.js';
import IRequestInit from '../fetch/types/IRequestInit.js';
import Blob from '../file/Blob.js';
import File from '../file/File.js';
import FileReader from '../file/FileReader.js';
import FormData from '../form-data/FormData.js';
import History from '../history/History.js';
import IntersectionObserver from '../intersection-observer/IntersectionObserver.js';
import IntersectionObserverEntry from '../intersection-observer/IntersectionObserverEntry.js';
import Location from '../location/Location.js';
import MediaQueryList from '../match-media/MediaQueryList.js';
import MutationObserver from '../mutation-observer/MutationObserver.js';
import MutationRecord from '../mutation-observer/MutationRecord.js';
import MimeType from '../navigator/MimeType.js';
import MimeTypeArray from '../navigator/MimeTypeArray.js';
import Navigator from '../navigator/Navigator.js';
import Plugin from '../navigator/Plugin.js';
import PluginArray from '../navigator/PluginArray.js';
import Attr from '../nodes/attr/Attr.js';
import CharacterData from '../nodes/character-data/CharacterData.js';
import Comment from '../nodes/comment/Comment.js';
import DocumentFragment from '../nodes/document-fragment/DocumentFragment.js';
import DocumentType from '../nodes/document-type/DocumentType.js';
import Document from '../nodes/document/Document.js';
import DocumentReadyStateEnum from '../nodes/document/DocumentReadyStateEnum.js';
import DocumentReadyStateManager from '../nodes/document/DocumentReadyStateManager.js';
import DOMRect from '../dom/DOMRect.js';
import DOMRectReadOnly from '../dom/DOMRectReadOnly.js';
import Element from '../nodes/element/Element.js';
import HTMLCollection from '../nodes/element/HTMLCollection.js';
import NamedNodeMap from '../nodes/element/NamedNodeMap.js';
import HTMLAnchorElement from '../nodes/html-anchor-element/HTMLAnchorElement.js';
import HTMLAreaElement from '../nodes/html-area-element/HTMLAreaElement.js';
import Audio from '../nodes/html-audio-element/Audio.js';
import HTMLAudioElement from '../nodes/html-audio-element/HTMLAudioElement.js';
import HTMLBaseElement from '../nodes/html-base-element/HTMLBaseElement.js';
import HTMLBodyElement from '../nodes/html-body-element/HTMLBodyElement.js';
import HTMLBRElement from '../nodes/html-br-element/HTMLBRElement.js';
import HTMLButtonElement from '../nodes/html-button-element/HTMLButtonElement.js';
import HTMLCanvasElement from '../nodes/html-canvas-element/HTMLCanvasElement.js';
import HTMLDListElement from '../nodes/html-d-list-element/HTMLDListElement.js';
import HTMLDataElement from '../nodes/html-data-element/HTMLDataElement.js';
import HTMLDataListElement from '../nodes/html-data-list-element/HTMLDataListElement.js';
import HTMLDetailsElement from '../nodes/html-details-element/HTMLDetailsElement.js';
import HTMLDialogElement from '../nodes/html-dialog-element/HTMLDialogElement.js';
import HTMLDivElement from '../nodes/html-div-element/HTMLDivElement.js';
import HTMLDocument from '../nodes/html-document/HTMLDocument.js';
import HTMLElement from '../nodes/html-element/HTMLElement.js';
import HTMLEmbedElement from '../nodes/html-embed-element/HTMLEmbedElement.js';
import HTMLFieldSetElement from '../nodes/html-field-set-element/HTMLFieldSetElement.js';
import HTMLFormControlsCollection from '../nodes/html-form-element/HTMLFormControlsCollection.js';
import HTMLFormElement from '../nodes/html-form-element/HTMLFormElement.js';
import RadioNodeList from '../nodes/html-form-element/RadioNodeList.js';
import HTMLHeadElement from '../nodes/html-head-element/HTMLHeadElement.js';
import HTMLHeadingElement from '../nodes/html-heading-element/HTMLHeadingElement.js';
import HTMLHRElement from '../nodes/html-hr-element/HTMLHRElement.js';
import HTMLHtmlElement from '../nodes/html-html-element/HTMLHtmlElement.js';
import HTMLIFrameElement from '../nodes/html-iframe-element/HTMLIFrameElement.js';
import HTMLImageElement from '../nodes/html-image-element/HTMLImageElement.js';
import Image from '../nodes/html-image-element/Image.js';
import FileList from '../nodes/html-input-element/FileList.js';
import HTMLInputElement from '../nodes/html-input-element/HTMLInputElement.js';
import HTMLLabelElement from '../nodes/html-label-element/HTMLLabelElement.js';
import HTMLLegendElement from '../nodes/html-legend-element/HTMLLegendElement.js';
import HTMLLIElement from '../nodes/html-li-element/HTMLLIElement.js';
import HTMLLinkElement from '../nodes/html-link-element/HTMLLinkElement.js';
import HTMLMapElement from '../nodes/html-map-element/HTMLMapElement.js';
import HTMLMediaElement from '../nodes/html-media-element/HTMLMediaElement.js';
import MediaStream from '../nodes/html-media-element/MediaStream.js';
import MediaStreamTrack from '../nodes/html-media-element/MediaStreamTrack.js';
import RemotePlayback from '../nodes/html-media-element/RemotePlayback.js';
import TextTrack from '../nodes/html-media-element/TextTrack.js';
import TextTrackCue from '../nodes/html-media-element/TextTrackCue.js';
import TextTrackCueList from '../nodes/html-media-element/TextTrackCueList.js';
import TextTrackList from '../nodes/html-media-element/TextTrackList.js';
import TimeRanges from '../nodes/html-media-element/TimeRanges.js';
import VTTCue from '../nodes/html-media-element/VTTCue.js';
import HTMLMenuElement from '../nodes/html-menu-element/HTMLMenuElement.js';
import HTMLMetaElement from '../nodes/html-meta-element/HTMLMetaElement.js';
import HTMLMeterElement from '../nodes/html-meter-element/HTMLMeterElement.js';
import HTMLModElement from '../nodes/html-mod-element/HTMLModElement.js';
import HTMLOListElement from '../nodes/html-o-list-element/HTMLOListElement.js';
import HTMLObjectElement from '../nodes/html-object-element/HTMLObjectElement.js';
import HTMLOptGroupElement from '../nodes/html-opt-group-element/HTMLOptGroupElement.js';
import HTMLOptionElement from '../nodes/html-option-element/HTMLOptionElement.js';
import HTMLOutputElement from '../nodes/html-output-element/HTMLOutputElement.js';
import HTMLParagraphElement from '../nodes/html-paragraph-element/HTMLParagraphElement.js';
import HTMLParamElement from '../nodes/html-param-element/HTMLParamElement.js';
import HTMLPictureElement from '../nodes/html-picture-element/HTMLPictureElement.js';
import HTMLPreElement from '../nodes/html-pre-element/HTMLPreElement.js';
import HTMLProgressElement from '../nodes/html-progress-element/HTMLProgressElement.js';
import HTMLQuoteElement from '../nodes/html-quote-element/HTMLQuoteElement.js';
import HTMLScriptElement from '../nodes/html-script-element/HTMLScriptElement.js';
import HTMLSelectElement from '../nodes/html-select-element/HTMLSelectElement.js';
import HTMLSlotElement from '../nodes/html-slot-element/HTMLSlotElement.js';
import HTMLSourceElement from '../nodes/html-source-element/HTMLSourceElement.js';
import HTMLSpanElement from '../nodes/html-span-element/HTMLSpanElement.js';
import HTMLStyleElement from '../nodes/html-style-element/HTMLStyleElement.js';
import HTMLTableCaptionElement from '../nodes/html-table-caption-element/HTMLTableCaptionElement.js';
import HTMLTableCellElement from '../nodes/html-table-cell-element/HTMLTableCellElement.js';
import HTMLTableColElement from '../nodes/html-table-col-element/HTMLTableColElement.js';
import HTMLTableElement from '../nodes/html-table-element/HTMLTableElement.js';
import HTMLTableRowElement from '../nodes/html-table-row-element/HTMLTableRowElement.js';
import HTMLTableSectionElement from '../nodes/html-table-section-element/HTMLTableSectionElement.js';
import HTMLTemplateElement from '../nodes/html-template-element/HTMLTemplateElement.js';
import HTMLTextAreaElement from '../nodes/html-text-area-element/HTMLTextAreaElement.js';
import HTMLTimeElement from '../nodes/html-time-element/HTMLTimeElement.js';
import HTMLTitleElement from '../nodes/html-title-element/HTMLTitleElement.js';
import HTMLTrackElement from '../nodes/html-track-element/HTMLTrackElement.js';
import HTMLUListElement from '../nodes/html-u-list-element/HTMLUListElement.js';
import HTMLUnknownElement from '../nodes/html-unknown-element/HTMLUnknownElement.js';
import HTMLVideoElement from '../nodes/html-video-element/HTMLVideoElement.js';
import Node from '../nodes/node/Node.js';
import NodeList from '../nodes/node/NodeList.js';
import ProcessingInstruction from '../nodes/processing-instruction/ProcessingInstruction.js';
import ShadowRoot from '../nodes/shadow-root/ShadowRoot.js';
import SVGElement from '../nodes/svg-element/SVGElement.js';
import Text from '../nodes/text/Text.js';
import XMLDocument from '../nodes/xml-document/XMLDocument.js';
import PermissionStatus from '../permissions/PermissionStatus.js';
import Permissions from '../permissions/Permissions.js';
import Range from '../range/Range.js';
import ResizeObserver from '../resize-observer/ResizeObserver.js';
import Screen from '../screen/Screen.js';
import Selection from '../selection/Selection.js';
import Storage from '../storage/Storage.js';
import NodeFilter from '../tree-walker/NodeFilter.js';
import NodeIterator from '../tree-walker/NodeIterator.js';
import TreeWalker from '../tree-walker/TreeWalker.js';
import URL from '../url/URL.js';
import ValidityState from '../validity-state/ValidityState.js';
import XMLHttpRequest from '../xml-http-request/XMLHttpRequest.js';
import XMLHttpRequestEventTarget from '../xml-http-request/XMLHttpRequestEventTarget.js';
import XMLHttpRequestUpload from '../xml-http-request/XMLHttpRequestUpload.js';
import XMLSerializer from '../xml-serializer/XMLSerializer.js';
import CrossOriginBrowserWindow from './CrossOriginBrowserWindow.js';
import INodeJSGlobal from './INodeJSGlobal.js';
import VMGlobalPropertyScript from './VMGlobalPropertyScript.js';
import WindowErrorUtility from './WindowErrorUtility.js';
import WindowPageOpenUtility from './WindowPageOpenUtility.js';
import {
	PerformanceObserver,
	PerformanceEntry,
	PerformanceObserverEntryList as IPerformanceObserverEntryList
} from 'node:perf_hooks';
import EventPhaseEnum from '../event/EventPhaseEnum.js';
import HTMLOptionsCollection from '../nodes/html-select-element/HTMLOptionsCollection.js';
import WindowContextClassExtender from './WindowContextClassExtender.js';
import WindowBrowserContext from './WindowBrowserContext.js';
import CanvasCaptureMediaStreamTrack from '../nodes/html-canvas-element/CanvasCaptureMediaStreamTrack.js';
import SVGSVGElement from '../nodes/svg-svg-element/SVGSVGElement.js';
import SVGGraphicsElement from '../nodes/svg-graphics-element/SVGGraphicsElement.js';
import SVGAnimateElement from '../nodes/svg-animate-element/SVGAnimateElement.js';
import SVGAnimateMotionElement from '../nodes/svg-animate-motion-element/SVGAnimateMotionElement.js';
import SVGAnimateTransformElement from '../nodes/svg-animate-transform-element/SVGAnimateTransformElement.js';
import SVGCircleElement from '../nodes/svg-circle-element/SVGCircleElement.js';
import SVGClipPathElement from '../nodes/svg-clip-path-element/SVGClipPathElement.js';
import SVGDefsElement from '../nodes/svg-defs-element/SVGDefsElement.js';
import SVGDescElement from '../nodes/svg-desc-element/SVGDescElement.js';
import SVGEllipseElement from '../nodes/svg-ellipse-element/SVGEllipseElement.js';
import SVGFEBlendElement from '../nodes/svg-fe-blend-element/SVGFEBlendElement.js';
import SVGFEColorMatrixElement from '../nodes/svg-fe-color-matrix-element/SVGFEColorMatrixElement.js';
import SVGFEComponentTransferElement from '../nodes/svg-fe-component-transfer-element/SVGFEComponentTransferElement.js';
import SVGFECompositeElement from '../nodes/svg-fe-composite-element/SVGFECompositeElement.js';
import SVGFEConvolveMatrixElement from '../nodes/svg-fe-convolve-matrix-element/SVGFEConvolveMatrixElement.js';
import SVGFEDiffuseLightingElement from '../nodes/svg-fe-diffuse-lighting-element/SVGFEDiffuseLightingElement.js';
import SVGFEDisplacementMapElement from '../nodes/svg-fe-displacement-map-element/SVGFEDisplacementMapElement.js';
import SVGFEDistantLightElement from '../nodes/svg-fe-distant-light-element/SVGFEDistantLightElement.js';
import SVGFEDropShadowElement from '../nodes/svg-fe-drop-shadow-element/SVGFEDropShadowElement.js';
import SVGFEFloodElement from '../nodes/svg-fe-flood-element/SVGFEFloodElement.js';
import SVGFEFuncAElement from '../nodes/svg-fe-func-a-element/SVGFEFuncAElement.js';
import SVGFEFuncBElement from '../nodes/svg-fe-func-b-element/SVGFEFuncBElement.js';
import SVGFEFuncGElement from '../nodes/svg-fe-func-g-element/SVGFEFuncGElement.js';
import SVGFEFuncRElement from '../nodes/svg-fe-func-r-element/SVGFEFuncRElement.js';
import SVGFEGaussianBlurElement from '../nodes/svg-fe-gaussian-blur-element/SVGFEGaussianBlurElement.js';
import SVGFEImageElement from '../nodes/svg-fe-image-element/SVGFEImageElement.js';
import SVGFEMergeElement from '../nodes/svg-fe-merge-element/SVGFEMergeElement.js';
import SVGFEMergeNodeElement from '../nodes/svg-fe-merge-node-element/SVGFEMergeNodeElement.js';
import SVGFEMorphologyElement from '../nodes/svg-fe-morphology-element/SVGFEMorphologyElement.js';
import SVGFEOffsetElement from '../nodes/svg-fe-offset-element/SVGFEOffsetElement.js';
import SVGFEPointLightElement from '../nodes/svg-fe-point-light-element/SVGFEPointLightElement.js';
import SVGFESpecularLightingElement from '../nodes/svg-fe-specular-lighting-element/SVGFESpecularLightingElement.js';
import SVGFESpotLightElement from '../nodes/svg-fe-spot-light-element/SVGFESpotLightElement.js';
import SVGFETileElement from '../nodes/svg-fe-tile-element/SVGFETileElement.js';
import SVGFETurbulenceElement from '../nodes/svg-fe-turbulence-element/SVGFETurbulenceElement.js';
import SVGFilterElement from '../nodes/svg-filter-element/SVGFilterElement.js';
import SVGForeignObjectElement from '../nodes/svg-foreign-object-element/SVGForeignObjectElement.js';
import SVGGElement from '../nodes/svg-g-element/SVGGElement.js';
import SVGImageElement from '../nodes/svg-image-element/SVGImageElement.js';
import SVGLineElement from '../nodes/svg-line-element/SVGLineElement.js';
import SVGLinearGradientElement from '../nodes/svg-linear-gradient-element/SVGLinearGradientElement.js';
import SVGMarkerElement from '../nodes/svg-marker-element/SVGMarkerElement.js';
import SVGMaskElement from '../nodes/svg-mask-element/SVGMaskElement.js';
import SVGMetadataElement from '../nodes/svg-metadata-element/SVGMetadataElement.js';
import SVGMPathElement from '../nodes/svg-m-path-element/SVGMPathElement.js';
import SVGPathElement from '../nodes/svg-path-element/SVGPathElement.js';
import SVGPatternElement from '../nodes/svg-pattern-element/SVGPatternElement.js';
import SVGPolygonElement from '../nodes/svg-polygon-element/SVGPolygonElement.js';
import SVGPolylineElement from '../nodes/svg-polyline-element/SVGPolylineElement.js';
import SVGRadialGradientElement from '../nodes/svg-radial-gradient-element/SVGRadialGradientElement.js';
import SVGRectElement from '../nodes/svg-rect-element/SVGRectElement.js';
import SVGScriptElement from '../nodes/svg-script-element/SVGScriptElement.js';
import SVGSetElement from '../nodes/svg-set-element/SVGSetElement.js';
import SVGStopElement from '../nodes/svg-stop-element/SVGStopElement.js';
import SVGStyleElement from '../nodes/svg-style-element/SVGStyleElement.js';
import SVGSwitchElement from '../nodes/svg-switch-element/SVGSwitchElement.js';
import SVGSymbolElement from '../nodes/svg-symbol-element/SVGSymbolElement.js';
import SVGTextElement from '../nodes/svg-text-element/SVGTextElement.js';
import SVGTextPathElement from '../nodes/svg-text-path-element/SVGTextPathElement.js';
import SVGTitleElement from '../nodes/svg-title-element/SVGTitleElement.js';
import SVGTSpanElement from '../nodes/svg-t-span-element/SVGTSpanElement.js';
import SVGUseElement from '../nodes/svg-use-element/SVGUseElement.js';
import SVGViewElement from '../nodes/svg-view-element/SVGViewElement.js';
import SVGAnimationElement from '../nodes/svg-animation-element/SVGAnimationElement.js';
import SVGComponentTransferFunctionElement from '../nodes/svg-component-transfer-function-element/SVGComponentTransferFunctionElement.js';
import SVGGeometryElement from '../nodes/svg-geometry-element/SVGGeometryElement.js';
import SVGGradientElement from '../nodes/svg-gradient-element/SVGGradientElement.js';
import SVGTextPositioningElement from '../nodes/svg-text-positioning-element/SVGTextPositioningElement.js';
import DOMMatrixReadOnly from '../dom/dom-matrix/DOMMatrixReadOnly.js';
import DOMMatrix from '../dom/dom-matrix/DOMMatrix.js';
import SVGAngle from '../svg/SVGAngle.js';
import SVGAnimatedAngle from '../svg/SVGAnimatedAngle.js';
import SVGAnimatedBoolean from '../svg/SVGAnimatedBoolean.js';
import SVGAnimatedEnumeration from '../svg/SVGAnimatedEnumeration.js';
import SVGAnimatedInteger from '../svg/SVGAnimatedInteger.js';
import SVGAnimatedLength from '../svg/SVGAnimatedLength.js';
import SVGLength from '../svg/SVGLength.js';
import SVGAnimatedNumber from '../svg/SVGAnimatedNumber.js';
import SVGAnimatedNumberList from '../svg/SVGAnimatedNumberList.js';
import SVGAnimatedPreserveAspectRatio from '../svg/SVGAnimatedPreserveAspectRatio.js';
import SVGAnimatedRect from '../svg/SVGAnimatedRect.js';
import SVGAnimatedString from '../svg/SVGAnimatedString.js';
import SVGAnimatedTransformList from '../svg/SVGAnimatedTransformList.js';
import SVGLengthList from '../svg/SVGLengthList.js';
import SVGMatrix from '../svg/SVGMatrix.js';
import SVGNumber from '../svg/SVGNumber.js';
import SVGNumberList from '../svg/SVGNumberList.js';
import SVGPoint from '../svg/SVGPoint.js';
import SVGPointList from '../svg/SVGPointList.js';
import SVGPreserveAspectRatio from '../svg/SVGPreserveAspectRatio.js';
import SVGRect from '../svg/SVGRect.js';
import SVGStringList from '../svg/SVGStringList.js';
import SVGTransform from '../svg/SVGTransform.js';
import SVGTransformList from '../svg/SVGTransformList.js';
import SVGUnitTypes from '../svg/SVGUnitTypes.js';
import DOMPoint from '../dom/DOMPoint.js';
import SVGAnimatedLengthList from '../svg/SVGAnimatedLengthList.js';
import CustomElementReactionStack from '../custom-element/CustomElementReactionStack.js';
import IScrollToOptions from './IScrollToOptions.js';

const TIMER = {
	setTimeout: globalThis.setTimeout.bind(globalThis),
	clearTimeout: globalThis.clearTimeout.bind(globalThis),
	setInterval: globalThis.setInterval.bind(globalThis),
	clearInterval: globalThis.clearInterval.bind(globalThis),
	queueMicrotask: globalThis.queueMicrotask.bind(globalThis),
	setImmediate: globalThis.setImmediate.bind(globalThis),
	clearImmediate: globalThis.clearImmediate.bind(globalThis)
};

const IS_NODE_JS_TIMEOUT_ENVIRONMENT = setTimeout.toString().includes('new Timeout');

/**
 * Class for PerformanceObserverEntryList as it is only available as an interface from Node.js.
 */
class PerformanceObserverEntryList {
	/**
	 * Constructor.
	 */
	constructor() {
		throw new TypeError('Illegal constructor');
	}
}
/**
 * Zero Timeout.
 */
class Timeout {
	public callback: () => void;
	/**
	 * Constructor.
	 * @param callback Callback.
	 */
	constructor(callback: () => void) {
		this.callback = callback;
	}
}

/**
 * Browser window.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/Window.
 */
export default class BrowserWindow extends EventTarget implements INodeJSGlobal {
	// Nodes
	public readonly Node = Node;
	public readonly Attr = Attr;
	public readonly ShadowRoot = ShadowRoot;
	public readonly ProcessingInstruction = ProcessingInstruction;
	public readonly Element = Element;
	public readonly CharacterData = CharacterData;
	public readonly DocumentType = DocumentType;

	// Nodes that can be created using "new" keyword (populated by WindowContextClassExtender)
	public declare readonly Document: typeof Document;
	public declare readonly HTMLDocument: typeof HTMLDocument;
	public declare readonly XMLDocument: typeof XMLDocument;
	public declare readonly DocumentFragment: typeof DocumentFragment;
	public declare readonly Text: typeof Text;
	public declare readonly Comment: typeof Comment;
	public declare readonly Image: typeof Image;
	public declare readonly Audio: typeof Audio;

	// HTML Element classes
	public readonly HTMLAnchorElement = HTMLAnchorElement;
	public readonly HTMLButtonElement = HTMLButtonElement;
	public readonly HTMLOptGroupElement = HTMLOptGroupElement;
	public readonly HTMLOptionElement = HTMLOptionElement;
	public readonly HTMLElement = HTMLElement;
	public readonly HTMLUnknownElement = HTMLUnknownElement;
	public readonly HTMLTemplateElement = HTMLTemplateElement;
	public readonly HTMLInputElement = HTMLInputElement;
	public readonly HTMLSelectElement = HTMLSelectElement;
	public readonly HTMLTextAreaElement = HTMLTextAreaElement;
	public readonly HTMLImageElement = HTMLImageElement;
	public readonly HTMLStyleElement = HTMLStyleElement;
	public readonly HTMLLabelElement = HTMLLabelElement;
	public readonly HTMLSlotElement = HTMLSlotElement;
	public readonly HTMLMetaElement = HTMLMetaElement;
	public readonly HTMLMediaElement = HTMLMediaElement;
	public readonly HTMLAudioElement = HTMLAudioElement;
	public readonly HTMLVideoElement = HTMLVideoElement;
	public readonly HTMLBaseElement = HTMLBaseElement;
	public readonly HTMLDialogElement = HTMLDialogElement;
	public readonly HTMLScriptElement = HTMLScriptElement;
	public readonly HTMLLinkElement = HTMLLinkElement;
	public readonly HTMLIFrameElement = HTMLIFrameElement;
	public readonly HTMLFormElement = HTMLFormElement;
	public readonly HTMLUListElement = HTMLUListElement;
	public readonly HTMLTrackElement = HTMLTrackElement;
	public readonly HTMLTableRowElement = HTMLTableRowElement;
	public readonly HTMLTitleElement = HTMLTitleElement;
	public readonly HTMLTimeElement = HTMLTimeElement;
	public readonly HTMLTableSectionElement = HTMLTableSectionElement;
	public readonly HTMLTableCellElement = HTMLTableCellElement;
	public readonly HTMLTableElement = HTMLTableElement;
	public readonly HTMLSpanElement = HTMLSpanElement;
	public readonly HTMLSourceElement = HTMLSourceElement;
	public readonly HTMLQuoteElement = HTMLQuoteElement;
	public readonly HTMLProgressElement = HTMLProgressElement;
	public readonly HTMLPreElement = HTMLPreElement;
	public readonly HTMLPictureElement = HTMLPictureElement;
	public readonly HTMLParamElement = HTMLParamElement;
	public readonly HTMLParagraphElement = HTMLParagraphElement;
	public readonly HTMLOutputElement = HTMLOutputElement;
	public readonly HTMLOListElement = HTMLOListElement;
	public readonly HTMLObjectElement = HTMLObjectElement;
	public readonly HTMLMeterElement = HTMLMeterElement;
	public readonly HTMLMenuElement = HTMLMenuElement;
	public readonly HTMLMapElement = HTMLMapElement;
	public readonly HTMLLIElement = HTMLLIElement;
	public readonly HTMLLegendElement = HTMLLegendElement;
	public readonly HTMLModElement = HTMLModElement;
	public readonly HTMLHtmlElement = HTMLHtmlElement;
	public readonly HTMLHRElement = HTMLHRElement;
	public readonly HTMLHeadElement = HTMLHeadElement;
	public readonly HTMLHeadingElement = HTMLHeadingElement;
	public readonly HTMLFieldSetElement = HTMLFieldSetElement;
	public readonly HTMLEmbedElement = HTMLEmbedElement;
	public readonly HTMLDListElement = HTMLDListElement;
	public readonly HTMLDivElement = HTMLDivElement;
	public readonly HTMLDetailsElement = HTMLDetailsElement;
	public readonly HTMLDataListElement = HTMLDataListElement;
	public readonly HTMLDataElement = HTMLDataElement;
	public readonly HTMLTableColElement = HTMLTableColElement;
	public readonly HTMLTableCaptionElement = HTMLTableCaptionElement;
	public readonly HTMLCanvasElement = HTMLCanvasElement;
	public readonly HTMLBRElement = HTMLBRElement;
	public readonly HTMLBodyElement = HTMLBodyElement;
	public readonly HTMLAreaElement = HTMLAreaElement;

	// SVG Element classes
	public readonly SVGSVGElement = SVGSVGElement;
	public readonly SVGAnimateElement = SVGAnimateElement;
	public readonly SVGAnimateMotionElement = SVGAnimateMotionElement;
	public readonly SVGAnimateTransformElement = SVGAnimateTransformElement;
	public readonly SVGCircleElement = SVGCircleElement;
	public readonly SVGClipPathElement = SVGClipPathElement;
	public readonly SVGDefsElement = SVGDefsElement;
	public readonly SVGDescElement = SVGDescElement;
	public readonly SVGEllipseElement = SVGEllipseElement;
	public readonly SVGFEBlendElement = SVGFEBlendElement;
	public readonly SVGFEColorMatrixElement = SVGFEColorMatrixElement;
	public readonly SVGFEComponentTransferElement = SVGFEComponentTransferElement;
	public readonly SVGFECompositeElement = SVGFECompositeElement;
	public readonly SVGFEConvolveMatrixElement = SVGFEConvolveMatrixElement;
	public readonly SVGFEDiffuseLightingElement = SVGFEDiffuseLightingElement;
	public readonly SVGFEDisplacementMapElement = SVGFEDisplacementMapElement;
	public readonly SVGFEDistantLightElement = SVGFEDistantLightElement;
	public readonly SVGFEDropShadowElement = SVGFEDropShadowElement;
	public readonly SVGFEFloodElement = SVGFEFloodElement;
	public readonly SVGFEFuncAElement = SVGFEFuncAElement;
	public readonly SVGFEFuncBElement = SVGFEFuncBElement;
	public readonly SVGFEFuncGElement = SVGFEFuncGElement;
	public readonly SVGFEFuncRElement = SVGFEFuncRElement;
	public readonly SVGFEGaussianBlurElement = SVGFEGaussianBlurElement;
	public readonly SVGFEImageElement = SVGFEImageElement;
	public readonly SVGFEMergeElement = SVGFEMergeElement;
	public readonly SVGFEMergeNodeElement = SVGFEMergeNodeElement;
	public readonly SVGFEMorphologyElement = SVGFEMorphologyElement;
	public readonly SVGFEOffsetElement = SVGFEOffsetElement;
	public readonly SVGFEPointLightElement = SVGFEPointLightElement;
	public readonly SVGFESpecularLightingElement = SVGFESpecularLightingElement;
	public readonly SVGFESpotLightElement = SVGFESpotLightElement;
	public readonly SVGFETileElement = SVGFETileElement;
	public readonly SVGFETurbulenceElement = SVGFETurbulenceElement;
	public readonly SVGFilterElement = SVGFilterElement;
	public readonly SVGForeignObjectElement = SVGForeignObjectElement;
	public readonly SVGGElement = SVGGElement;
	public readonly SVGImageElement = SVGImageElement;
	public readonly SVGLineElement = SVGLineElement;
	public readonly SVGLinearGradientElement = SVGLinearGradientElement;
	public readonly SVGMarkerElement = SVGMarkerElement;
	public readonly SVGMaskElement = SVGMaskElement;
	public readonly SVGMetadataElement = SVGMetadataElement;
	public readonly SVGMPathElement = SVGMPathElement;
	public readonly SVGPathElement = SVGPathElement;
	public readonly SVGPatternElement = SVGPatternElement;
	public readonly SVGPolygonElement = SVGPolygonElement;
	public readonly SVGPolylineElement = SVGPolylineElement;
	public readonly SVGRadialGradientElement = SVGRadialGradientElement;
	public readonly SVGRectElement = SVGRectElement;
	public readonly SVGScriptElement = SVGScriptElement;
	public readonly SVGSetElement = SVGSetElement;
	public readonly SVGStopElement = SVGStopElement;
	public readonly SVGStyleElement = SVGStyleElement;
	public readonly SVGSwitchElement = SVGSwitchElement;
	public readonly SVGSymbolElement = SVGSymbolElement;
	public readonly SVGTextElement = SVGTextElement;
	public readonly SVGTextPathElement = SVGTextPathElement;
	public readonly SVGTitleElement = SVGTitleElement;
	public readonly SVGTSpanElement = SVGTSpanElement;
	public readonly SVGUseElement = SVGUseElement;
	public readonly SVGViewElement = SVGViewElement;

	// Abstract SVG Element classes
	public readonly SVGElement = SVGElement;
	public readonly SVGAnimationElement = SVGAnimationElement;
	public readonly SVGComponentTransferFunctionElement = SVGComponentTransferFunctionElement;
	public readonly SVGGeometryElement = SVGGeometryElement;
	public readonly SVGGradientElement = SVGGradientElement;
	public readonly SVGTextPositioningElement = SVGTextPositioningElement;
	public readonly SVGGraphicsElement = SVGGraphicsElement;

	// Event classes
	public readonly Event = Event;
	public readonly UIEvent = UIEvent;
	public readonly CustomEvent = CustomEvent;
	public readonly AnimationEvent = AnimationEvent;
	public readonly KeyboardEvent = KeyboardEvent;
	public readonly MessageEvent = MessageEvent;
	public readonly MouseEvent = MouseEvent;
	public readonly PointerEvent = PointerEvent;
	public readonly FocusEvent = FocusEvent;
	public readonly WheelEvent = WheelEvent;
	public readonly InputEvent = InputEvent;
	public readonly ErrorEvent = ErrorEvent;
	public readonly StorageEvent = StorageEvent;
	public readonly SubmitEvent = SubmitEvent;
	public readonly ProgressEvent = ProgressEvent;
	public readonly MediaQueryListEvent = MediaQueryListEvent;
	public readonly HashChangeEvent = HashChangeEvent;
	public readonly ClipboardEvent = ClipboardEvent;
	public readonly TouchEvent = TouchEvent;
	public readonly Touch = Touch;

	// Non-implemented event classes
	public readonly AudioProcessingEvent = Event;
	public readonly BeforeInputEvent = Event;
	public readonly BeforeUnloadEvent = Event;
	public readonly BlobEvent = Event;
	public readonly CloseEvent = Event;
	public readonly CompositionEvent = Event;
	public readonly CSSFontFaceLoadEvent = Event;
	public readonly DeviceLightEvent = Event;
	public readonly DeviceMotionEvent = Event;
	public readonly DeviceOrientationEvent = Event;
	public readonly DeviceProximityEvent = Event;
	public readonly DOMTransactionEvent = Event;
	public readonly DragEvent = Event;
	public readonly EditingBeforeInputEvent = Event;
	public readonly FetchEvent = Event;
	public readonly GamepadEvent = Event;
	public readonly IDBVersionChangeEvent = Event;
	public readonly MediaStreamEvent = Event;
	public readonly MutationEvent = Event;
	public readonly OfflineAudioCompletionEvent = Event;
	public readonly OverconstrainedError = Event;
	public readonly PageTransitionEvent = Event;
	public readonly PaymentRequestUpdateEvent = Event;
	public readonly PopStateEvent = Event;
	public readonly RelatedEvent = Event;
	public readonly RTCDataChannelEvent = Event;
	public readonly RTCIdentityErrorEvent = Event;
	public readonly RTCIdentityEvent = Event;
	public readonly RTCPeerConnectionIceEvent = Event;
	public readonly SensorEvent = Event;
	public readonly SVGEvent = Event;
	public readonly SVGZoomEvent = Event;
	public readonly TimeEvent = Event;
	public readonly TrackEvent = Event;
	public readonly TransitionEvent = Event;
	public readonly UserProximityEvent = Event;
	public readonly WebGLContextEvent = Event;
	public readonly TextEvent = Event;

	// Other classes that has to be bound to the Window context (populated by WindowContextClassExtender)
	public declare readonly NodeIterator: typeof NodeIterator;
	public declare readonly TreeWalker: typeof TreeWalker;
	public declare readonly MutationObserver: typeof MutationObserver;
	public declare readonly MessagePort: typeof MessagePort;
	public declare readonly DataTransfer: typeof DataTransfer;
	public declare readonly DataTransferItem: typeof DataTransferItem;
	public declare readonly DataTransferItemList: typeof DataTransferItemList;
	public declare readonly XMLSerializer: typeof XMLSerializer;
	public declare readonly CSSStyleSheet: typeof CSSStyleSheet;
	public declare readonly DOMException: typeof DOMException;
	public declare readonly CSSUnitValue: typeof CSSUnitValue;
	public declare readonly Selection: typeof Selection;
	public declare readonly Headers: typeof Headers;
	public declare readonly Request: typeof Request;
	public declare readonly Response: typeof Response;
	public declare readonly EventTarget: typeof EventTarget;
	public declare readonly XMLHttpRequestUpload: typeof XMLHttpRequestUpload;
	public declare readonly XMLHttpRequestEventTarget: typeof XMLHttpRequestEventTarget;
	public declare readonly AbortController: typeof AbortController;
	public declare readonly AbortSignal: typeof AbortSignal;
	public declare readonly FormData: typeof FormData;
	public declare readonly PermissionStatus: typeof PermissionStatus;
	public declare readonly ClipboardItem: typeof ClipboardItem;
	public declare readonly XMLHttpRequest: typeof XMLHttpRequest;
	public declare readonly DOMParser: typeof DOMParser;
	public declare readonly Range: typeof Range;
	public declare readonly VTTCue: typeof VTTCue;
	public declare readonly FileReader: typeof FileReader;
	public declare readonly MediaStream: typeof MediaStream;
	public declare readonly MediaStreamTrack: typeof MediaStreamTrack;
	public declare readonly CanvasCaptureMediaStreamTrack: typeof CanvasCaptureMediaStreamTrack;
	public declare readonly NamedNodeMap: typeof NamedNodeMap;
	public declare readonly TextTrack: typeof TextTrack;
	public declare readonly TextTrackList: typeof TextTrackList;
	public declare readonly TextTrackCue: typeof TextTrackCue;
	public declare readonly RemotePlayback: typeof RemotePlayback;

	// Other classes that don't have to be bound to the Window context
	public readonly Permissions = Permissions;
	public readonly History = History;
	public readonly Navigator = Navigator;
	public readonly Clipboard = Clipboard;
	public readonly TimeRanges = TimeRanges;
	public readonly TextTrackCueList = TextTrackCueList;
	public readonly ValidityState = ValidityState;
	public readonly MutationRecord = MutationRecord;
	public readonly IntersectionObserver = IntersectionObserver;
	public readonly IntersectionObserverEntry = IntersectionObserverEntry;
	public readonly CSSStyleDeclaration = CSSStyleDeclaration;
	public readonly CSSRule = CSSRule;
	public readonly CSSContainerRule = CSSContainerRule;
	public readonly CSSFontFaceRule = CSSFontFaceRule;
	public readonly CSSKeyframeRule = CSSKeyframeRule;
	public readonly CSSKeyframesRule = CSSKeyframesRule;
	public readonly CSSMediaRule = CSSMediaRule;
	public readonly CSSStyleRule = CSSStyleRule;
	public readonly CSSSupportsRule = CSSSupportsRule;
	public readonly DOMRect = DOMRect;
	public readonly DOMRectReadOnly = DOMRectReadOnly;
	public readonly Plugin = Plugin;
	public readonly PluginArray = PluginArray;
	public readonly Location = Location;
	public readonly CustomElementRegistry = CustomElementRegistry;
	public readonly ResizeObserver = ResizeObserver;
	public readonly URL = URL;
	public readonly Blob = Blob;
	public readonly File = File;
	public readonly Storage = Storage;
	public readonly MimeType = MimeType;
	public readonly MimeTypeArray = MimeTypeArray;
	public readonly NodeFilter = NodeFilter;
	public readonly HTMLCollection = HTMLCollection;
	public readonly HTMLFormControlCollection = HTMLFormControlsCollection;
	public readonly HTMLOptionsCollection = HTMLOptionsCollection;
	public readonly NodeList = NodeList;
	public readonly RadioNodeList = RadioNodeList;
	public readonly FileList = FileList;
	public readonly Screen = Screen;
	public readonly DOMMatrixReadOnly = DOMMatrixReadOnly;
	public readonly DOMMatrix = DOMMatrix;
	public readonly SVGAngle = SVGAngle;
	public readonly SVGAnimatedAngle = SVGAnimatedAngle;
	public readonly SVGAnimatedBoolean = SVGAnimatedBoolean;
	public readonly SVGAnimatedEnumeration = SVGAnimatedEnumeration;
	public readonly SVGAnimatedInteger = SVGAnimatedInteger;
	public readonly SVGAnimatedLength = SVGAnimatedLength;
	public readonly SVGAnimatedNumber = SVGAnimatedNumber;
	public readonly SVGAnimatedNumberList = SVGAnimatedNumberList;
	public readonly SVGAnimatedPreserveAspectRatio = SVGAnimatedPreserveAspectRatio;
	public readonly SVGAnimatedRect = SVGAnimatedRect;
	public readonly SVGAnimatedString = SVGAnimatedString;
	public readonly SVGAnimatedTransformList = SVGAnimatedTransformList;
	public readonly SVGLength = SVGLength;
	public readonly SVGLengthList = SVGLengthList;
	public readonly SVGMatrix = SVGMatrix;
	public readonly SVGNumber = SVGNumber;
	public readonly SVGNumberList = SVGNumberList;
	public readonly SVGPoint = SVGPoint;
	public readonly SVGPointList = SVGPointList;
	public readonly SVGPreserveAspectRatio = SVGPreserveAspectRatio;
	public readonly SVGRect = SVGRect;
	public readonly SVGStringList = SVGStringList;
	public readonly SVGTransform = SVGTransform;
	public readonly SVGTransformList = SVGTransformList;
	public readonly SVGAnimatedLengthList = SVGAnimatedLengthList;
	public readonly SVGUnitTypes = SVGUnitTypes;
	public readonly DOMPoint = DOMPoint;
	public readonly Window = <typeof BrowserWindow>this.constructor;

	// Node.js Classes
	public readonly URLSearchParams = URLSearchParams;
	public readonly WritableStream = Stream.Writable;
	public readonly ReadableStream = ReadableStream;
	public readonly TransformStream = Stream.Transform;
	public readonly PerformanceObserver = PerformanceObserver;
	public readonly PerformanceEntry = PerformanceEntry;
	public readonly PerformanceObserverEntryList: new () => IPerformanceObserverEntryList = <
		new () => IPerformanceObserverEntryList
	>PerformanceObserverEntryList;

	// Events
	public onload: ((event: Event) => void) | null = null;
	public onerror: ((event: ErrorEvent) => void) | null = null;

	// Public properties.
	public readonly document: Document;
	public readonly customElements: CustomElementRegistry = new CustomElementRegistry(this);
	public readonly window: BrowserWindow = this;
	public readonly globalThis: BrowserWindow = this;
	public readonly performance: typeof performance = performance;
	public readonly screenLeft: number = 0;
	public readonly screenTop: number = 0;
	public readonly screenX: number = 0;
	public readonly screenY: number = 0;
	public readonly crypto: typeof webcrypto = webcrypto;
	public readonly TextEncoder: typeof TextEncoder = TextEncoder;
	public readonly TextDecoder: typeof TextDecoder = TextDecoder;
	public readonly closed = false;
	public console: Console;
	public name = '';

	// Node.js Globals (populated by VMGlobalPropertyScript)
	public declare Array: typeof Array;
	public declare ArrayBuffer: typeof ArrayBuffer;
	public declare Boolean: typeof Boolean;
	public Buffer: typeof Buffer = Buffer;
	public declare DataView: typeof DataView;
	public declare Date: typeof Date;
	public declare Error: typeof Error;
	public declare EvalError: typeof EvalError;
	public declare Float32Array: typeof Float32Array;
	public declare Float64Array: typeof Float64Array;
	public declare Function: typeof Function;
	public declare Infinity: typeof Infinity;
	public declare Int16Array: typeof Int16Array;
	public declare Int32Array: typeof Int32Array;
	public declare Int8Array: typeof Int8Array;
	public declare Intl: typeof Intl;
	public declare JSON: typeof JSON;
	public declare Map: MapConstructor;
	public declare Math: typeof Math;
	public declare NaN: typeof NaN;
	public declare Number: typeof Number;
	public declare Object: typeof Object;
	public declare Promise: typeof Promise;
	public declare RangeError: typeof RangeError;
	public declare ReferenceError: typeof ReferenceError;
	public declare RegExp: typeof RegExp;
	public declare Set: SetConstructor;
	public declare String: typeof String;
	public declare Symbol: Function;
	public declare SyntaxError: typeof SyntaxError;
	public declare TypeError: typeof TypeError;
	public declare URIError: typeof URIError;
	public declare Uint16Array: typeof Uint16Array;
	public declare Uint32Array: typeof Uint32Array;
	public declare Uint8Array: typeof Uint8Array;
	public declare Uint8ClampedArray: typeof Uint8ClampedArray;
	public declare WeakMap: WeakMapConstructor;
	public declare WeakSet: WeakSetConstructor;
	public declare decodeURI: typeof decodeURI;
	public declare decodeURIComponent: typeof decodeURIComponent;
	public declare encodeURI: typeof encodeURI;
	public declare encodeURIComponent: typeof encodeURIComponent;
	public declare eval: typeof eval;
	/**
	 * @deprecated
	 */
	public declare escape: (str: string) => string;
	public declare global: typeof globalThis;
	public declare isFinite: typeof isFinite;
	public declare isNaN: typeof isNaN;
	public declare parseFloat: typeof parseFloat;
	public declare parseInt: typeof parseInt;
	public declare undefined: typeof undefined;
	/**
	 * @deprecated
	 */
	public declare unescape: (str: string) => string;
	public declare gc: () => void;
	public declare v8debug?: unknown;

	// Public internal properties

	// Used for tracking capture event listeners to improve performance when they are not used.
	// See EventTarget class.
	public [PropertySymbol.mutationObservers]: MutationObserver[] = [];
	public readonly [PropertySymbol.readyStateManager] = new DocumentReadyStateManager(this);
	public [PropertySymbol.location]: Location;
	public [PropertySymbol.history]: History;
	public [PropertySymbol.navigator]: Navigator;
	public [PropertySymbol.screen]: Screen;
	public [PropertySymbol.sessionStorage]: Storage;
	public [PropertySymbol.localStorage]: Storage;
	public [PropertySymbol.self]: BrowserWindow = this;
	public [PropertySymbol.top]: BrowserWindow = this;
	public [PropertySymbol.parent]: BrowserWindow = this;
	public [PropertySymbol.window]: BrowserWindow = this;
	public [PropertySymbol.internalId]: number = -1;
	public [PropertySymbol.customElementReactionStack] = new CustomElementReactionStack(this);

	// Private properties
	#browserFrame: IBrowserFrame;
	#innerWidth: number | null = null;
	#innerHeight: number | null = null;
	#outerWidth: number | null = null;
	#outerHeight: number | null = null;
	#devicePixelRatio: number | null = null;
	#zeroDelayTimeout: { timeouts: Array<Timeout> | null } = { timeouts: null };
	#timerLoopStacks: string[] = [];

	/**
	 * Constructor.
	 *
	 * @param browserFrame Browser frame.
	 * @param [options] Options.
	 * @param [options.url] URL.
	 */
	constructor(browserFrame: IBrowserFrame, options?: { url?: string }) {
		super();

		this.#browserFrame = browserFrame;

		this.console = browserFrame.page.console;

		this[PropertySymbol.navigator] = new Navigator(this);
		this[PropertySymbol.screen] = new Screen();
		this[PropertySymbol.sessionStorage] = new Storage();
		this[PropertySymbol.localStorage] = new Storage();
		this[PropertySymbol.location] = new Location(this.#browserFrame, options?.url ?? 'about:blank');
		this[PropertySymbol.history] = new History(this.#browserFrame, this);

		WindowBrowserContext.setWindowBrowserFrameRelation(this, this.#browserFrame);

		this[PropertySymbol.setupVMContext]();

		WindowContextClassExtender.extendClasses(this);

		// Document
		this.document = new this.HTMLDocument();
		this.document[PropertySymbol.defaultView] = this;

		// Ready state manager
		this[PropertySymbol.readyStateManager].waitUntilComplete().then(() => {
			this.document[PropertySymbol.readyState] = DocumentReadyStateEnum.complete;
			this.document.dispatchEvent(new Event('readystatechange'));

			// Not sure why target is set to document here, but this is how it works in the browser
			const loadEvent = new Event('load');

			loadEvent[PropertySymbol.currentTarget] = this;
			loadEvent[PropertySymbol.target] = this.document;
			loadEvent[PropertySymbol.eventPhase] = EventPhaseEnum.atTarget;

			this.dispatchEvent(loadEvent);

			loadEvent[PropertySymbol.currentTarget] = null;
			loadEvent[PropertySymbol.eventPhase] = EventPhaseEnum.none;
			loadEvent[PropertySymbol.dispatching] = false;
		});

		this[PropertySymbol.bindMethods]();
	}

	/**
	 * Returns self.
	 *
	 * @returns Self.
	 */
	public get self(): BrowserWindow {
		return this[PropertySymbol.self];
	}

	/**
	 * Returns self.
	 *
	 * @param self Self.
	 */
	public set self(self: BrowserWindow | null) {
		this[PropertySymbol.self] = self;
	}

	/**
	 * Returns top.
	 *
	 * @returns Top.
	 */
	public get top(): BrowserWindow {
		return this[PropertySymbol.top];
	}

	/**
	 * Returns parent.
	 *
	 * @returns Parent.
	 */
	public get parent(): BrowserWindow {
		return this[PropertySymbol.parent];
	}

	/**
	 * Returns parent.
	 *
	 * @param parent Parent.
	 */
	public set parent(parent: BrowserWindow | null) {
		this[PropertySymbol.parent] = parent;
	}

	/**
	 * Returns location.
	 */
	public get location(): Location {
		return this[PropertySymbol.location];
	}

	/**
	 * Returns location.
	 *
	 * @param href Href.
	 */
	public set location(href: string) {
		this[PropertySymbol.location].href = href;
	}

	/**
	 * Returns history.
	 */
	public get history(): History {
		return this[PropertySymbol.history];
	}

	/**
	 * Returns navigator.
	 */
	public get navigator(): Navigator {
		return this[PropertySymbol.navigator];
	}

	/**
	 * Returns screen.
	 */
	public get screen(): Screen {
		return this[PropertySymbol.screen];
	}

	/**
	 * Returns session storage.
	 */
	public get sessionStorage(): Storage {
		return this[PropertySymbol.sessionStorage];
	}

	/**
	 * Returns local storage.
	 */
	public get localStorage(): Storage {
		return this[PropertySymbol.localStorage];
	}

	/**
	 * Returns opener.
	 *
	 * @returns Opener.
	 */
	public get opener(): BrowserWindow | CrossOriginBrowserWindow | null {
		return this.#browserFrame[PropertySymbol.openerWindow];
	}

	/**
	 * The number of pixels that the document is currently scrolled horizontally.
	 *
	 * @returns Scroll X.
	 */
	public get scrollX(): number {
		return this.document?.documentElement?.scrollLeft ?? 0;
	}

	/**
	 * The read-only Window property pageXOffset is an alias for scrollX.
	 *
	 * @returns Scroll X.
	 */
	public get pageXOffset(): number {
		return this.scrollX;
	}

	/**
	 * The number of pixels that the document is currently scrolled vertically.
	 *
	 * @returns Scroll Y.
	 */
	public get scrollY(): number {
		return this.document?.documentElement?.scrollTop ?? 0;
	}

	/**
	 * The read-only Window property pageYOffset is an alias for scrollY.
	 *
	 * @returns Scroll Y.
	 */
	public get pageYOffset(): number {
		return this.scrollY;
	}

	/**
	 * The CSS interface holds useful CSS-related methods.
	 *
	 * @returns CSS interface.
	 */
	public get CSS(): CSS {
		return new CSS();
	}

	/**
	 * Returns inner width.
	 *
	 * @returns Inner width.
	 */
	public get innerWidth(): number {
		if (this.#innerWidth === null) {
			return this.#browserFrame.page.viewport.width;
		}
		return this.#innerWidth;
	}

	/**
	 * Sets inner width.
	 *
	 * @param value Inner width.
	 */
	public set innerWidth(value: number) {
		this.#innerWidth = value;
	}

	/**
	 * Returns inner height.
	 *
	 * @returns Inner height.
	 */
	public get innerHeight(): number {
		// It seems like this value can be defined according to spec, but changing it has no effect on the actual viewport.
		if (this.#innerHeight === null) {
			return this.#browserFrame.page.viewport.height;
		}
		return this.#innerHeight;
	}

	/**
	 * Sets inner height.
	 *
	 * @param value Inner height.
	 */
	public set innerHeight(value: number) {
		this.#innerHeight = value;
	}

	/**
	 * Returns outer width.
	 *
	 * @returns Outer width.
	 */
	public get outerWidth(): number {
		// It seems like this value can be defined according to spec, but changing it has no effect on the actual viewport.
		if (this.#outerWidth === null) {
			return this.#browserFrame.page.viewport.width;
		}
		return this.#outerWidth;
	}

	/**
	 * Sets outer width.
	 *
	 * @param value Outer width.
	 */
	public set outerWidth(value: number) {
		this.#outerWidth = value;
	}

	/**
	 * Returns outer height.
	 *
	 * @returns Outer height.
	 */
	public get outerHeight(): number {
		if (this.#outerHeight === null) {
			return this.#browserFrame.page.viewport.height;
		}
		return this.#outerHeight;
	}

	/**
	 * Sets outer height.
	 *
	 * @param value Outer height.
	 */
	public set outerHeight(value: number) {
		this.#outerHeight = value;
	}

	/**
	 * Returns device pixel ratio.
	 *
	 * @returns Device pixel ratio.
	 */
	public get devicePixelRatio(): number {
		// It seems like this value can be defined according to spec, but changing it has no effect on the actual viewport.
		if (this.#devicePixelRatio === null) {
			return this.#browserFrame.page.viewport.devicePixelRatio;
		}
		return this.#devicePixelRatio;
	}

	/**
	 * Sets device pixel ratio.
	 *
	 * @param value Device pixel ratio.
	 */
	public set devicePixelRatio(value: number) {
		this.#devicePixelRatio = value;
	}

	/**
	 * Returns an object containing the values of all CSS properties of an element.
	 *
	 * @param element Element.
	 * @returns CSS style declaration.
	 */
	public getComputedStyle(element: Element): CSSStyleDeclaration {
		element[PropertySymbol.computedStyle] =
			element[PropertySymbol.computedStyle] ||
			new CSSStyleDeclaration(PropertySymbol.illegalConstructor, this, { element, computed: true });
		return element[PropertySymbol.computedStyle];
	}

	/**
	 * Returns selection.
	 *
	 * @returns Selection.
	 */
	public getSelection(): Selection {
		return this.document.getSelection();
	}

	/**
	 * Scrolls to a particular set of coordinates.
	 *
	 * @param x X position or options object.
	 * @param y Y position.
	 */
	public scroll(x: IScrollToOptions | number, y?: number): void {
		if (typeof x !== 'object' && arguments.length === 1) {
			throw new this.TypeError(
				"Failed to execute 'scroll' on 'Window': The provided value is not of type 'ScrollToOptions'."
			);
		}

		const options = typeof x === 'object' ? x : { left: x, top: y };

		if (options.behavior === 'smooth') {
			this.setTimeout(() => {
				if (options.top !== undefined) {
					const top = Number(options.top);
					this.document.documentElement.scrollTop = isNaN(top) ? 0 : top;
				}
				if (options.left !== undefined) {
					const left = Number(options.left);
					this.document.documentElement.scrollLeft = isNaN(left) ? 0 : left;
				}
			});
		} else {
			if (options.top !== undefined) {
				const top = Number(options.top);
				this.document.documentElement.scrollTop = isNaN(top) ? 0 : top;
			}
			if (options.left !== undefined) {
				const left = Number(options.left);
				this.document.documentElement.scrollLeft = isNaN(left) ? 0 : left;
			}
		}
	}

	/**
	 * Scrolls to a particular set of coordinates.
	 *
	 * @param x X position or options object.
	 * @param y Y position.
	 */
	public scrollTo(x: IScrollToOptions | number, y?: number): void {
		if (typeof x !== 'object' && arguments.length === 1) {
			throw new this.TypeError(
				"Failed to execute 'scrollTo' on 'Window': The provided value is not of type 'ScrollToOptions'."
			);
		}
		this.scroll(x, y);
	}

	/**
	 * Scrolls by a relative amount from the current position.
	 *
	 * @param x Pixels to scroll by from top or scroll options object.
	 * @param y Pixels to scroll by from left.
	 */
	public scrollBy(x: IScrollToOptions | number, y?: number): void {
		if (typeof x !== 'object' && arguments.length === 1) {
			throw new this.TypeError(
				"Failed to execute 'scrollBy' on 'Window': The provided value is not of type 'ScrollToOptions'."
			);
		}
		const options = typeof x === 'object' ? x : { left: x, top: y };
		this.scroll({
			left: this.document.documentElement.scrollLeft + (options.left ?? 0),
			top: this.document.documentElement.scrollTop + (options.top ?? 0),
			behavior: options.behavior
		});
	}

	/**
	 * Shifts focus away from the window.
	 */
	public blur(): void {
		// TODO: Implement.
	}

	/**
	 * Gives focus to the window.
	 */
	public focus(): void {
		// TODO: Implement.
	}

	/**
	 * Loads a specified resource into a new or existing browsing context (that is, a tab, a window, or an iframe) under a specified name.
	 *
	 * @param [url] URL.
	 * @param [target] Target.
	 * @param [features] Window features.
	 * @returns Window.
	 */
	public open(
		url?: string,
		target?: string,
		features?: string
	): BrowserWindow | CrossOriginBrowserWindow | null {
		return WindowPageOpenUtility.openPage(this.#browserFrame, {
			url,
			target,
			features
		});
	}

	/**
	 * Closes the window.
	 */
	public close(): void {
		// When using a Window instance directly, the Window instance is the main frame and we will close the page and destroy the browser.
		// When using the Browser API we should only close the page when the Window instance is connected to the main frame (we should not close child frames such as iframes).
		if (this.#browserFrame.page?.mainFrame === this.#browserFrame) {
			this[PropertySymbol.destroy]();
			this.#browserFrame.page.close();
		}
	}

	/**
	 * Returns a new MediaQueryList object that can then be used to determine if the document matches the media query string.
	 *
	 * @param mediaQueryString A string specifying the media query to parse into a MediaQueryList.
	 * @returns A new MediaQueryList.
	 */
	public matchMedia(mediaQueryString: string): MediaQueryList {
		return new MediaQueryList({ window: this, media: mediaQueryString });
	}

	/**
	 * Sets a timer which executes a function once the timer expires.
	 *
	 * @param callback Function to be executed.
	 * @param [delay=0] Delay in ms.
	 * @param args Arguments passed to the callback function.
	 * @returns Timeout ID.
	 */
	public setTimeout(callback: Function, delay = 0, ...args: unknown[]): NodeJS.Timeout {
		if (this.closed) {
			return;
		}

		const settings = this.#browserFrame.page?.context?.browser?.settings;

		if (settings.timer.preventTimerLoops) {
			const stack = new Error().stack;
			const timerLoopStacks = this.#timerLoopStacks;
			if (timerLoopStacks.includes(stack)) {
				return;
			}
			timerLoopStacks.push(stack);
		}

		// We can group timeouts with a delay of 0 into one timeout to improve performance.
		// Grouping timeouts will also improve the performance of the async task manager.
		// It also makes the async task manager more stable as many timeouts may cause waitUntilComplete() to be resolved too early.
		if (!delay) {
			const zeroDelayTimeout = this.#zeroDelayTimeout;

			if (!zeroDelayTimeout.timeouts) {
				const useTryCatch =
					!settings ||
					(!settings.disableErrorCapturing &&
						settings.errorCapture === BrowserErrorCaptureEnum.tryAndCatch);

				const id = TIMER.setTimeout(() => {
					// We need to call endTimer() before the callback as the callback might throw an error.
					this.#browserFrame[PropertySymbol.asyncTaskManager].endTimer(id);
					const timeouts = zeroDelayTimeout.timeouts;
					zeroDelayTimeout.timeouts = null;
					for (const timeout of timeouts) {
						if (useTryCatch) {
							WindowErrorUtility.captureError(this, () => timeout.callback());
						} else {
							timeout.callback();
						}
					}
				});

				zeroDelayTimeout.timeouts = [];
				this.#browserFrame[PropertySymbol.asyncTaskManager].startTimer(id);
			}

			const timeout = new Timeout(() => callback(...args));

			zeroDelayTimeout.timeouts.push(timeout);

			return <NodeJS.Timeout>(<unknown>timeout);
		}

		const useTryCatch =
			!settings ||
			(!settings.disableErrorCapturing &&
				settings.errorCapture === BrowserErrorCaptureEnum.tryAndCatch);

		const id = TIMER.setTimeout(
			() => {
				// We need to call endTimer() before the callback as the callback might throw an error.
				this.#browserFrame[PropertySymbol.asyncTaskManager].endTimer(id);
				if (useTryCatch) {
					WindowErrorUtility.captureError(this, () => callback(...args));
				} else {
					callback(...args);
				}
			},
			settings?.timer.maxTimeout !== -1 && delay && delay > settings?.timer.maxTimeout
				? settings?.timer.maxTimeout
				: delay
		);
		this.#browserFrame[PropertySymbol.asyncTaskManager].startTimer(id);
		return id;
	}

	/**
	 * Cancels a timeout previously established by calling setTimeout().
	 *
	 * @param id ID of the timeout.
	 */
	public clearTimeout(id: NodeJS.Timeout): void {
		if (id && id instanceof Timeout) {
			const zeroDelayTimeout = this.#zeroDelayTimeout;
			if (!zeroDelayTimeout.timeouts) {
				return;
			}
			const index = zeroDelayTimeout.timeouts.indexOf(<Timeout>(<unknown>id));
			if (index !== -1) {
				zeroDelayTimeout.timeouts.splice(index, 1);
			}
			return;
		}
		// We need to make sure that the ID is a Timeout object, otherwise Node.js might throw an error.
		// This is only necessary if we are in a Node.js environment.
		if (IS_NODE_JS_TIMEOUT_ENVIRONMENT && (!id || id.constructor.name !== 'Timeout')) {
			return;
		}
		TIMER.clearTimeout(id);
		this.#browserFrame[PropertySymbol.asyncTaskManager].endTimer(id);
	}

	/**
	 * Calls a function with a fixed time delay between each call.
	 *
	 * @param callback Function to be executed.
	 * @param [delay=0] Delay in ms.
	 * @param args Arguments passed to the callback function.
	 * @returns Interval ID.
	 */
	public setInterval(callback: Function, delay = 0, ...args: unknown[]): NodeJS.Timeout {
		if (this.closed) {
			return;
		}
		const settings = this.#browserFrame.page?.context?.browser?.settings;
		const useTryCatch =
			!settings ||
			(!settings.disableErrorCapturing &&
				settings.errorCapture === BrowserErrorCaptureEnum.tryAndCatch);
		let iterations = 0;
		const id = TIMER.setInterval(
			() => {
				if (useTryCatch) {
					WindowErrorUtility.captureError(
						this,
						() => callback(...args),
						() => this.clearInterval(id)
					);
				} else {
					callback(...args);
				}
				if (settings?.timer.maxIntervalIterations !== -1) {
					if (iterations >= settings?.timer.maxIntervalIterations) {
						this.clearInterval(id);
					}
					iterations++;
				}
			},
			settings?.timer.maxIntervalTime !== -1 && delay && delay > settings?.timer.maxIntervalTime
				? settings?.timer.maxIntervalTime
				: delay
		);
		this.#browserFrame[PropertySymbol.asyncTaskManager].startTimer(id);
		return id;
	}

	/**
	 * Cancels a timed repeating action which was previously established by a call to setInterval().
	 *
	 * @param id ID of the interval.
	 */
	public clearInterval(id: NodeJS.Timeout): void {
		// We need to make sure that the ID is a Timeout object, otherwise Node.js might throw an error.
		// This is only necessary if we are in a Node.js environment.
		if (IS_NODE_JS_TIMEOUT_ENVIRONMENT && (!id || id.constructor.name !== 'Timeout')) {
			return;
		}
		TIMER.clearInterval(id);
		this.#browserFrame[PropertySymbol.asyncTaskManager].endTimer(id);
	}

	/**
	 * Mock animation frames with timeouts.
	 *
	 * @param callback Callback.
	 * @returns ID.
	 */
	public requestAnimationFrame(callback: (timestamp: number) => void): NodeJS.Immediate {
		if (this.closed) {
			return;
		}
		const settings = this.#browserFrame.page?.context?.browser?.settings;

		if (settings.timer.preventTimerLoops) {
			const stack = new Error().stack;
			const timerLoopStacks = this.#timerLoopStacks;
			if (timerLoopStacks.includes(stack)) {
				return;
			}
			timerLoopStacks.push(stack);
		}

		const useTryCatch =
			!settings ||
			(!settings.disableErrorCapturing &&
				settings.errorCapture === BrowserErrorCaptureEnum.tryAndCatch);
		const id = TIMER.setImmediate(() => {
			// We need to call endImmediate() before the callback as the callback might throw an error.
			this.#browserFrame[PropertySymbol.asyncTaskManager].endImmediate(id);
			if (useTryCatch) {
				WindowErrorUtility.captureError(this, () => callback(this.performance.now()));
			} else {
				callback(this.performance.now());
			}
		});
		this.#browserFrame[PropertySymbol.asyncTaskManager].startImmediate(id);
		return id;
	}

	/**
	 * Mock animation frames with timeouts.
	 *
	 * @param id ID.
	 */
	public cancelAnimationFrame(id: NodeJS.Immediate): void {
		// We need to make sure that the ID is an Immediate object, otherwise Node.js might throw an error.
		// This is only necessary if we are in a Node.js environment.
		if (IS_NODE_JS_TIMEOUT_ENVIRONMENT && (!id || id.constructor.name !== 'Immediate')) {
			return;
		}
		TIMER.clearImmediate(id);
		this.#browserFrame[PropertySymbol.asyncTaskManager].endImmediate(id);
	}

	/**
	 * Queues a microtask to be executed at a safe time prior to control returning to the browser's event loop.
	 *
	 * @param callback Function to be executed.
	 */
	public queueMicrotask(callback: Function): void {
		if (this.closed) {
			return;
		}
		let isAborted = false;
		const taskId = this.#browserFrame[PropertySymbol.asyncTaskManager].startTask(
			() => (isAborted = true)
		);
		const settings = this.#browserFrame.page?.context?.browser?.settings;
		const useTryCatch =
			!settings ||
			(!settings.disableErrorCapturing &&
				settings.errorCapture === BrowserErrorCaptureEnum.tryAndCatch);
		TIMER.queueMicrotask(() => {
			if (!isAborted) {
				// We need to call endTask() before the callback as the callback might throw an error.
				this.#browserFrame[PropertySymbol.asyncTaskManager].endTask(taskId);
				if (useTryCatch) {
					WindowErrorUtility.captureError(this, <() => unknown>callback);
				} else {
					callback();
				}
			}
		});
	}

	/**
	 * This method provides an easy, logical way to fetch resources asynchronously across the network.
	 *
	 * @param url URL.
	 * @param [init] Init.
	 * @returns Promise.
	 */
	public async fetch(url: IRequestInfo, init?: IRequestInit): Promise<Response> {
		if (this.closed) {
			return Promise.reject(
				new this.DOMException(
					"Failed to execute 'fetch' on 'Window': The window is closed.",
					DOMExceptionNameEnum.invalidStateError
				)
			);
		}
		return await new Fetch({
			browserFrame: this.#browserFrame,
			window: this,
			url,
			init
		}).send();
	}

	/**
	 * Creates a Base64-encoded ASCII string from a binary string (i.e., a string in which each character in the string is treated as a byte of binary data).
	 *
	 * @see https://developer.mozilla.org/en-US/docs/Web/API/btoa
	 * @param data Binay data.
	 * @returns Base64-encoded string.
	 */
	public btoa(data: unknown): string {
		return Base64.btoa(data);
	}

	/**
	 * Decodes a string of data which has been encoded using Base64 encoding.
	 *
	 * @see https://developer.mozilla.org/en-US/docs/Web/API/atob
	 * @see https://infra.spec.whatwg.org/#forgiving-base64-encode.
	 * @see Https://html.spec.whatwg.org/multipage/webappapis.html#btoa.
	 * @param data Binay string.
	 * @returns An ASCII string containing decoded data from encodedData.
	 */
	public atob(data: unknown): string {
		return Base64.atob(data);
	}

	/**
	 * Safely enables cross-origin communication between Window objects; e.g., between a page and a pop-up that it spawned, or between a page and an iframe embedded within it.
	 *
	 * @param message Message.
	 * @param [targetOrigin=*] Target origin.
	 * @param _transfer Transfer. Not implemented.
	 */
	public postMessage(message: unknown, targetOrigin = '*', _transfer?: unknown[]): void {
		// TODO: Implement transfer.

		if (this.closed) {
			return;
		}

		if (targetOrigin && targetOrigin !== '*' && this.location.origin !== targetOrigin) {
			throw new this.DOMException(
				`Failed to execute 'postMessage' on 'Window': The target origin provided ('${targetOrigin}') does not match the recipient window\'s origin ('${this.location.origin}').`,
				DOMExceptionNameEnum.securityError
			);
		}

		try {
			JSON.stringify(message);
		} catch (error) {
			throw new this.DOMException(
				`Failed to execute 'postMessage' on 'Window': The provided message cannot be serialized.`,
				DOMExceptionNameEnum.invalidStateError
			);
		}

		this.setTimeout(() =>
			this.dispatchEvent(
				new MessageEvent('message', {
					data: message,
					origin: this.#browserFrame.parentFrame
						? this.#browserFrame.parentFrame.window.location.origin
						: this.#browserFrame.window.location.origin,
					source: this.#browserFrame.parentFrame
						? this.#browserFrame.parentFrame.window
						: this.#browserFrame.window,
					lastEventId: ''
				})
			)
		);
	}

	/**
	 * Resizes the window.
	 *
	 * @param width Width.
	 * @param height Height.
	 */
	public resizeTo(width: number, height: number): void {
		if (this.closed) {
			return;
		}

		if (!width || !height) {
			throw new this.DOMException(
				`Failed to execute 'resizeTo' on 'Window': 2 arguments required, but only ${arguments.length} present.`
			);
		}

		// We can only resize the window if it is a popup.
		if (this.#browserFrame[PropertySymbol.popup]) {
			this.#browserFrame.page.setViewport({ width, height });
		}
	}

	/**
	 * Resizes the current window by a specified amount.
	 *
	 * @param width Width.
	 * @param height Height.
	 */
	public resizeBy(width: number, height: number): void {
		if (this.closed) {
			return;
		}

		if (!width || !height) {
			throw new this.DOMException(
				`Failed to execute 'resizeBy' on 'Window': 2 arguments required, but only ${arguments.length} present.`
			);
		}

		// We can only resize the window if it is a popup.
		if (this.#browserFrame[PropertySymbol.popup]) {
			const viewport = this.#browserFrame.page.viewport;
			this.#browserFrame.page.setViewport({
				width: viewport.width + width,
				height: viewport.height + height
			});
		}
	}

	/**
	 * Setup of VM context.
	 */
	protected [PropertySymbol.setupVMContext](): void {
		if (!VM.isContext(this)) {
			VM.createContext(this);

			// Sets global properties from the VM to the Window object.
			// Otherwise "this.Array" will be undefined for example.
			VMGlobalPropertyScript.runInContext(this);
		}
	}

	/**
	 * Destroys the window.
	 */
	public [PropertySymbol.destroy](): void {
		if (<boolean>this.closed) {
			return;
		}

		(<boolean>this.closed) = true;

		const mutationObservers = this[PropertySymbol.mutationObservers];

		for (const mutationObserver of mutationObservers) {
			if (mutationObserver[PropertySymbol.destroy]) {
				mutationObserver[PropertySymbol.destroy]();
			}
		}

		this[PropertySymbol.mutationObservers] = [];
		this[PropertySymbol.asyncTaskManager] = null;
		this[PropertySymbol.mutationObservers] = [];

		// Disconnects nodes from the document, so that they can be garbage collected.
		const childNodes = this.document[PropertySymbol.nodeArray];

		while (childNodes.length > 0) {
			// Makes sure that something won't be triggered by the disconnect.
			if ((<HTMLElement>childNodes[0]).disconnectedCallback) {
				delete (<HTMLElement>childNodes[0]).disconnectedCallback;
			}
			this.document[PropertySymbol.removeChild](childNodes[0]);
		}

		// Create some empty elements for scripts that are still running.
		const htmlElement = this.document.createElement('html');
		const headElement = this.document.createElement('head');
		const bodyElement = this.document.createElement('body');
		htmlElement.appendChild(headElement);
		htmlElement.appendChild(bodyElement);
		this.document[PropertySymbol.appendChild](htmlElement);

		if (this.location[PropertySymbol.destroy]) {
			this.location[PropertySymbol.destroy]();
		}

		if (this.customElements[PropertySymbol.destroy]) {
			this.customElements[PropertySymbol.destroy]();
		}

		if (this.history[PropertySymbol.destroy]) {
			this.history[PropertySymbol.destroy]();
		}

		this.document[PropertySymbol.activeElement] = null;
		this.document[PropertySymbol.nextActiveElement] = null;
		this.document[PropertySymbol.currentScript] = null;
		this.document[PropertySymbol.selection] = null;

		WindowBrowserContext.removeWindowBrowserFrameRelation(this);
	}

	/**
	 * Binds methods to a window as scope.
	 */
	public [PropertySymbol.bindMethods](): void {
		for (const _class of [BrowserWindow, EventTarget]) {
			const propertyDescriptors = Object.getOwnPropertyDescriptors(_class.prototype);
			const keys: Array<string | symbol> = Object.keys(propertyDescriptors);

			for (const key of keys) {
				const descriptor = propertyDescriptors[<string>key];
				if (descriptor.get || descriptor.set) {
					Object.defineProperty(this, key, {
						...descriptor,
						get: descriptor.get?.bind(this),
						set: descriptor.set?.bind(this)
					});
				} else if (
					key !== 'constructor' &&
					typeof descriptor.value === 'function' &&
					!descriptor.value.toString().startsWith('class ')
				) {
					Object.defineProperty(this, key, {
						...descriptor,
						value: descriptor.value.bind(this)
					});
				}
			}
		}
	}
}
