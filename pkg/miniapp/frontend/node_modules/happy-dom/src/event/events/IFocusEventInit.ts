import EventTarget from '../EventTarget.js';
import IUIEventInit from '../IUIEventInit.js';

export default interface IFocusEventInit extends IUIEventInit {
	relatedTarget?: EventTarget;
}
