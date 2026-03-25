import HTMLButtonElement from '../html-button-element/HTMLButtonElement.js';
import HTMLFieldSetElement from '../html-field-set-element/HTMLFieldSetElement.js';
import HTMLInputElement from '../html-input-element/HTMLInputElement.js';
import HTMLObjectElement from '../html-object-element/HTMLObjectElement.js';
import HTMLOutputElement from '../html-output-element/HTMLOutputElement.js';
import HTMLSelectElement from '../html-select-element/HTMLSelectElement.js';
import HTMLTextAreaElement from '../html-text-area-element/HTMLTextAreaElement.js';

type THTMLFormControlElement =
	| HTMLInputElement
	| HTMLSelectElement
	| HTMLTextAreaElement
	| HTMLButtonElement
	| HTMLFieldSetElement
	| HTMLObjectElement
	| HTMLOutputElement;

export default THTMLFormControlElement;
