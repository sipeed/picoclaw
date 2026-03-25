import CSSStyleDeclarationPropertyManager from '../../css/declaration/property-manager/CSSStyleDeclarationPropertyManager.js';
import ICachedResult from './ICachedResult.js';

export default interface ICachedComputedStyleResult extends ICachedResult {
	result: WeakRef<CSSStyleDeclarationPropertyManager | null> | null;
}
