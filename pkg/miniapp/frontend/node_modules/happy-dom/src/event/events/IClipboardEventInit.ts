import DataTransfer from '../DataTransfer.js';
import IEventInit from '../IEventInit.js';

export default interface IClipboardEventInit extends IEventInit {
	clipboardData?: DataTransfer | null;
}
