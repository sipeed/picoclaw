import HTMLElement from '../../nodes/html-element/HTMLElement.js';
import IEventInit from '../IEventInit.js';

export default interface ISubmitEventInit extends IEventInit {
	submitter?: HTMLElement;
}
