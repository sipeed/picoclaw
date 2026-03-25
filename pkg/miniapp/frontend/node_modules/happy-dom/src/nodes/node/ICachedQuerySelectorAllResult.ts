import Element from '../element/Element.js';
import ICachedResult from './ICachedResult.js';
import NodeList from './NodeList.js';

export default interface ICachedQuerySelectorAllResult extends ICachedResult {
	result: WeakRef<NodeList<Element>> | null;
}
