import IVirtualConsoleLogGroup from './IVirtualConsoleLogGroup.js';
import VirtualConsoleLogLevelEnum from './enums/VirtualConsoleLogLevelEnum.js';
import VirtualConsoleLogTypeEnum from './enums/VirtualConsoleLogTypeEnum.js';

export default interface IVirtualConsoleLogEntry {
	type: VirtualConsoleLogTypeEnum;
	level: VirtualConsoleLogLevelEnum;
	message: Array<string | object>;
	group: IVirtualConsoleLogGroup | null;
}
