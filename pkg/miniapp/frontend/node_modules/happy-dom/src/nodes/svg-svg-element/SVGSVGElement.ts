import SVGGraphicsElement from '../svg-graphics-element/SVGGraphicsElement.js';
import SVGRect from '../../svg/SVGRect.js';
import SVGPoint from '../../svg/SVGPoint.js';
import SVGLength from '../../svg/SVGLength.js';
import SVGAngle from '../../svg/SVGAngle.js';
import SVGNumber from '../../svg/SVGNumber.js';
import SVGTransform from '../../svg/SVGTransform.js';
import SVGAnimatedRect from '../../svg/SVGAnimatedRect.js';
import Event from '../../event/Event.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import SVGAnimatedPreserveAspectRatio from '../../svg/SVGAnimatedPreserveAspectRatio.js';
import SVGAnimatedLength from '../../svg/SVGAnimatedLength.js';
import Element from '../element/Element.js';
import NodeList from '../node/NodeList.js';
import SVGElement from '../svg-element/SVGElement.js';
import SVGMatrix from '../../svg/SVGMatrix.js';
import HTMLCollection from '../element/HTMLCollection.js';
import ParentNodeUtility from '../parent-node/ParentNodeUtility.js';
import IHTMLElementTagNameMap from '../../config/IHTMLElementTagNameMap.js';
import ISVGElementTagNameMap from '../../config/ISVGElementTagNameMap.js';

/**
 * SVGSVGElement.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGSVGElement
 */
export default class SVGSVGElement extends SVGGraphicsElement {
	// Internal properties
	public [PropertySymbol.preserveAspectRatio]: SVGAnimatedPreserveAspectRatio | null = null;
	public [PropertySymbol.x]: SVGAnimatedLength | null = null;
	public [PropertySymbol.y]: SVGAnimatedLength | null = null;
	public [PropertySymbol.width]: SVGAnimatedLength | null = null;
	public [PropertySymbol.height]: SVGAnimatedLength | null = null;
	public [PropertySymbol.currentScale]: number = 1;
	public [PropertySymbol.viewBox]: SVGAnimatedRect | null = null;

	// Public properties
	public declare cloneNode: (deep?: boolean) => SVGSVGElement;

	// Events
	public onafterprint: ((event: Event) => void) | null = null;
	public onbeforeprint: ((event: Event) => void) | null = null;
	public onbeforeunload: ((event: Event) => void) | null = null;
	public ongamepadconnected: ((event: Event) => void) | null = null;
	public ongamepaddisconnected: ((event: Event) => void) | null = null;
	public onhashchange: ((event: Event) => void) | null = null;
	public onlanguagechange: ((event: Event) => void) | null = null;
	public onmessage: ((event: Event) => void) | null = null;
	public onmessageerror: ((event: Event) => void) | null = null;
	public onoffline: ((event: Event) => void) | null = null;
	public ononline: ((event: Event) => void) | null = null;
	public onpagehide: ((event: Event) => void) | null = null;
	public onpageshow: ((event: Event) => void) | null = null;
	public onpopstate: ((event: Event) => void) | null = null;
	public onrejectionhandled: ((event: Event) => void) | null = null;
	public onstorage: ((event: Event) => void) | null = null;
	public onunhandledrejection: ((event: Event) => void) | null = null;
	public onunload: ((event: Event) => void) | null = null;

	/**
	 * Returns preserve aspect ratio.
	 *
	 * @returns Preserve aspect ratio.
	 */
	public get preserveAspectRatio(): SVGAnimatedPreserveAspectRatio {
		if (!this[PropertySymbol.preserveAspectRatio]) {
			this[PropertySymbol.preserveAspectRatio] = new SVGAnimatedPreserveAspectRatio(
				PropertySymbol.illegalConstructor,
				this[PropertySymbol.window],
				{
					getAttribute: () => this.getAttribute('preserveAspectRatio'),
					setAttribute: (value) => this.setAttribute('preserveAspectRatio', value)
				}
			);
		}
		return this[PropertySymbol.preserveAspectRatio];
	}

	/**
	 * Returns height.
	 *
	 * @returns Height.
	 */
	public get height(): SVGAnimatedLength {
		if (!this[PropertySymbol.height]) {
			this[PropertySymbol.height] = new SVGAnimatedLength(
				PropertySymbol.illegalConstructor,
				this[PropertySymbol.window],
				{
					getAttribute: () => this.getAttribute('height'),
					setAttribute: (value) => this.setAttribute('height', value)
				}
			);
		}
		return this[PropertySymbol.height];
	}

	/**
	 * Returns width.
	 *
	 * @returns Width.
	 */
	public get width(): SVGAnimatedLength {
		if (!this[PropertySymbol.width]) {
			this[PropertySymbol.width] = new SVGAnimatedLength(
				PropertySymbol.illegalConstructor,
				this[PropertySymbol.window],
				{
					getAttribute: () => this.getAttribute('width'),
					setAttribute: (value) => this.setAttribute('width', value)
				}
			);
		}
		return this[PropertySymbol.width];
	}

	/**
	 * Returns x position.
	 *
	 * @returns X position.
	 */
	public get x(): SVGAnimatedLength {
		if (!this[PropertySymbol.x]) {
			this[PropertySymbol.x] = new SVGAnimatedLength(
				PropertySymbol.illegalConstructor,
				this[PropertySymbol.window],
				{
					getAttribute: () => this.getAttribute('x'),
					setAttribute: (value) => this.setAttribute('x', value)
				}
			);
		}
		return this[PropertySymbol.x];
	}

	/**
	 * Returns y position.
	 *
	 * @returns Y position.
	 */
	public get y(): SVGAnimatedLength {
		if (!this[PropertySymbol.y]) {
			this[PropertySymbol.y] = new SVGAnimatedLength(
				PropertySymbol.illegalConstructor,
				this[PropertySymbol.window],
				{
					getAttribute: () => this.getAttribute('y'),
					setAttribute: (value) => this.setAttribute('y', value)
				}
			);
		}
		return this[PropertySymbol.y];
	}

	/**
	 * Returns currentScale.
	 *
	 * @returns CurrentScale.
	 */
	public get currentScale(): number {
		return this[PropertySymbol.currentScale];
	}

	/**
	 * Sets currentScale.
	 *
	 * @param currentScale CurrentScale.
	 */
	public set currentScale(currentScale: number) {
		const parsed =
			typeof currentScale !== 'number' ? parseFloat(String(currentScale)) : currentScale;
		if (isNaN(parsed)) {
			throw this[PropertySymbol.window].TypeError(
				`Failed to set the 'currentScale' property on 'SVGSVGElement': The provided float value is non-finite.`
			);
		}
		if (parsed < 1) {
			return;
		}
		this[PropertySymbol.currentScale] = parsed;
	}

	/**
	 * Returns current translate.
	 *
	 * @returns SVG point.
	 */
	public get currentTranslate(): SVGPoint {
		return new SVGPoint(PropertySymbol.illegalConstructor, this[PropertySymbol.window]);
	}

	/**
	 * Returns view box.
	 *
	 * @returns View box.
	 */
	public get viewBox(): SVGAnimatedRect {
		if (!this[PropertySymbol.viewBox]) {
			this[PropertySymbol.viewBox] = new SVGAnimatedRect(
				PropertySymbol.illegalConstructor,
				this[PropertySymbol.window],
				{
					getAttribute: () => this.getAttribute('viewBox'),
					setAttribute: (value) => this.setAttribute('viewBox', value)
				}
			);
		}
		return this[PropertySymbol.viewBox];
	}

	/**
	 * Pauses animation.
	 */
	public pauseAnimations(): void {}

	/**
	 * Unpauses animation.
	 */
	public unpauseAnimations(): void {}

	/**
	 * Returns "true" if animation is paused.
	 *
	 * @returns "true" if animation is paused.
	 */
	public animationsPaused(): boolean {
		return false;
	}

	/**
	 * Returns the current time in seconds relative to the start time for the current SVG document fragment.
	 *
	 * @returns Current time in seconds.
	 */
	public getCurrentTime(): number {
		return 0;
	}

	/**
	 * Sets current time.
	 *
	 * @param _seconds Seconds.
	 */
	public setCurrentTime(_seconds: number): void {}

	/**
	 * Returns intersection list.
	 *
	 * @param _rect SVG Rect.
	 * @param _element SVG Element.
	 * @returns Intersection list.
	 */
	public getIntersectionList(_rect: SVGRect, _element: SVGElement): NodeList<SVGElement> {
		return new NodeList<SVGElement>(PropertySymbol.illegalConstructor, []);
	}

	/**
	 * Returns enclousure list.
	 *
	 * @param _rect SVG Rect.
	 * @param _element SVG Element.
	 * @returns Enclousure list.
	 */
	public getEnclosureList(_rect: SVGRect, _element: SVGElement): NodeList<SVGElement> {
		return new NodeList<SVGElement>(PropertySymbol.illegalConstructor, []);
	}

	/**
	 * Returns true if the rendered content of the given element intersects the supplied rectangle.
	 *
	 * @param _element SVG Element.
	 * @param _rect SVG Rect.
	 * @returns Intersection state.
	 */
	public checkIntersection(_element: SVGElement, _rect: SVGRect): boolean {
		return false;
	}

	/**
	 * Returns true if the rendered content of the given element is entirely contained within the supplied rectangle.
	 *
	 * @param _element SVG Element.
	 * @param _rect SVG Rect.
	 * @returns Enclousure state.
	 */
	public checkEnclosure(_element: SVGElement, _rect: SVGRect): boolean {
		return false;
	}

	/**
	 * Unselects any selected objects, including any selections of text strings and type-in bars.
	 */
	public deselectAll(): void {}

	/**
	 * Returns a number.
	 *
	 * @returns Number.
	 */
	public createSVGNumber(): SVGNumber {
		return new SVGNumber(PropertySymbol.illegalConstructor, this[PropertySymbol.window]);
	}

	/**
	 * Returns a length.
	 *
	 * @returns Length.
	 */
	public createSVGLength(): SVGLength {
		return new SVGLength(PropertySymbol.illegalConstructor, this[PropertySymbol.window]);
	}

	/**
	 * Returns a angle.
	 *
	 * @returns Angle.
	 */
	public createSVGAngle(): SVGAngle {
		return new SVGAngle(PropertySymbol.illegalConstructor, this[PropertySymbol.window]);
	}

	/**
	 * Returns a point.
	 *
	 * @returns Point.
	 */
	public createSVGPoint(): SVGPoint {
		return new SVGPoint(PropertySymbol.illegalConstructor, this[PropertySymbol.window]);
	}

	/**
	 * Returns a matrix.
	 *
	 * @returns Matrix.
	 */
	public createSVGMatrix(): SVGMatrix {
		return new SVGMatrix(PropertySymbol.illegalConstructor, this[PropertySymbol.window]);
	}

	/**
	 * Returns a rect.
	 *
	 * @returns Rect.
	 */
	public createSVGRect(): SVGRect {
		return new SVGRect(PropertySymbol.illegalConstructor, this[PropertySymbol.window]);
	}

	/**
	 * Returns a transform.
	 *
	 * @returns Transform.
	 */
	public createSVGTransform(): SVGTransform {
		return new SVGTransform(PropertySymbol.illegalConstructor, this[PropertySymbol.window]);
	}

	/**
	 * Returns a transform from a matrix.
	 *
	 * @param matrix Matrix.
	 */
	public createSVGTransformFromMatrix(matrix: SVGMatrix): SVGTransform {
		const transform = new SVGTransform(
			PropertySymbol.illegalConstructor,
			this[PropertySymbol.window]
		);
		transform.setMatrix(matrix);
		return transform;
	}

	/**
	 * Returns an elements by class name.
	 *
	 * @param className Tag name.
	 * @returns Matching element.
	 */
	public getElementsByClassName(className: string): HTMLCollection<Element> {
		return ParentNodeUtility.getElementsByClassName(this, className);
	}

	/**
	 * Returns an elements by tag name.
	 *
	 * @param tagName Tag name.
	 * @returns Matching element.
	 */
	public getElementsByTagName<K extends keyof ISVGElementTagNameMap>(
		tagName: K
	): HTMLCollection<ISVGElementTagNameMap[K]>;

	/**
	 * Returns an elements by tag name.
	 *
	 * @param tagName Tag name.
	 * @returns Matching element.
	 */
	public getElementsByTagName(tagName: string): HTMLCollection<Element>;

	/**
	 * Returns an elements by tag name.
	 *
	 * @param tagName Tag name.
	 * @returns Matching element.
	 */
	public getElementsByTagName(tagName: string): HTMLCollection<Element> {
		return ParentNodeUtility.getElementsByTagName(this, tagName);
	}

	/**
	 * Returns an elements by tag name and namespace.
	 *
	 * @param namespaceURI Namespace URI.
	 * @param tagName Tag name.
	 * @returns Matching element.
	 */
	public getElementsByTagNameNS<K extends keyof IHTMLElementTagNameMap>(
		namespaceURI: 'http://www.w3.org/1999/xhtml',
		tagName: K
	): HTMLCollection<IHTMLElementTagNameMap[K]>;

	/**
	 * Returns an elements by tag name and namespace.
	 *
	 * @param namespaceURI Namespace URI.
	 * @param tagName Tag name.
	 * @returns Matching element.
	 */
	public getElementsByTagNameNS<K extends keyof ISVGElementTagNameMap>(
		namespaceURI: 'http://www.w3.org/2000/svg',
		tagName: K
	): HTMLCollection<ISVGElementTagNameMap[K]>;

	/**
	 * Returns an elements by tag name and namespace.
	 *
	 * @param namespaceURI Namespace URI.
	 * @param tagName Tag name.
	 * @returns Matching element.
	 */
	public getElementsByTagNameNS(namespaceURI: string, tagName: string): HTMLCollection<Element>;

	/**
	 * Returns an elements by tag name and namespace.
	 *
	 * @param namespaceURI Namespace URI.
	 * @param tagName Tag name.
	 * @returns Matching element.
	 */
	public getElementsByTagNameNS(namespaceURI: string, tagName: string): HTMLCollection<Element> {
		return ParentNodeUtility.getElementsByTagNameNS(this, namespaceURI, tagName);
	}

	/**
	 * Returns an element by ID.
	 *
	 * @param id ID.
	 * @returns Matching element.
	 */
	public getElementById(id: string): Element | null {
		return <Element>ParentNodeUtility.getElementById(this, id);
	}

	/**
	 * @override
	 */
	public override [PropertySymbol.cloneNode](deep = false): SVGSVGElement {
		return <SVGSVGElement>super[PropertySymbol.cloneNode](deep);
	}
}
