import SVGGeometryElement from '../svg-geometry-element/SVGGeometryElement.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import SVGAnimatedLength from '../../svg/SVGAnimatedLength.js';

/**
 * SVG Ellipse Element.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGEllipseElement
 */
export default class SVGEllipseElement extends SVGGeometryElement {
	// Internal properties
	public [PropertySymbol.cx]: SVGAnimatedLength | null = null;
	public [PropertySymbol.cy]: SVGAnimatedLength | null = null;
	public [PropertySymbol.rx]: SVGAnimatedLength | null = null;
	public [PropertySymbol.ry]: SVGAnimatedLength | null = null;

	/**
	 * Returns cx.
	 *
	 * @returns Cx.
	 */
	public get cx(): SVGAnimatedLength {
		if (!this[PropertySymbol.cx]) {
			this[PropertySymbol.cx] = new SVGAnimatedLength(
				PropertySymbol.illegalConstructor,
				this[PropertySymbol.window],
				{
					getAttribute: () => this.getAttribute('cx'),
					setAttribute: (value) => this.setAttribute('cx', value)
				}
			);
		}
		return this[PropertySymbol.cx];
	}

	/**
	 * Returns cy.
	 *
	 * @returns Cy.
	 */
	public get cy(): SVGAnimatedLength {
		if (!this[PropertySymbol.cy]) {
			this[PropertySymbol.cy] = new SVGAnimatedLength(
				PropertySymbol.illegalConstructor,
				this[PropertySymbol.window],
				{
					getAttribute: () => this.getAttribute('cy'),
					setAttribute: (value) => this.setAttribute('cy', value)
				}
			);
		}
		return this[PropertySymbol.cy];
	}

	/**
	 * Returns rx.
	 *
	 * @returns Rx.
	 */
	public get rx(): SVGAnimatedLength {
		if (!this[PropertySymbol.rx]) {
			this[PropertySymbol.rx] = new SVGAnimatedLength(
				PropertySymbol.illegalConstructor,
				this[PropertySymbol.window],
				{
					getAttribute: () => this.getAttribute('rx'),
					setAttribute: (value) => this.setAttribute('rx', value)
				}
			);
		}
		return this[PropertySymbol.rx];
	}

	/**
	 * Returns ry.
	 *
	 * @returns Ry.
	 */
	public get ry(): SVGAnimatedLength {
		if (!this[PropertySymbol.ry]) {
			this[PropertySymbol.ry] = new SVGAnimatedLength(
				PropertySymbol.illegalConstructor,
				this[PropertySymbol.window],
				{
					getAttribute: () => this.getAttribute('ry'),
					setAttribute: (value) => this.setAttribute('ry', value)
				}
			);
		}
		return this[PropertySymbol.ry];
	}
}
