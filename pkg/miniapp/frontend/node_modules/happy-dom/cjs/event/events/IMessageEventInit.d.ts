import IEventInit from '../IEventInit.cjs';
import BrowserWindow from '../../window/BrowserWindow.cjs';
import MessagePort from '../MessagePort.cjs';
export default interface IMessageEventInit extends IEventInit {
    data?: unknown | null;
    origin?: string;
    lastEventId?: string;
    source?: BrowserWindow | null;
    ports?: MessagePort[];
}
//# sourceMappingURL=IMessageEventInit.d.ts.map