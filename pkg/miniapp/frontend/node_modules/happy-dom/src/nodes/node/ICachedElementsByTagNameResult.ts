import Element from '../element/Element.js';
import ICachedResult from './ICachedResult.js';

export default interface ICachedElementsByTagNameResult extends ICachedResult {
	result: WeakRef<Element[]> | null;
}
