import IVirtualConsoleLogGroup from './IVirtualConsoleLogGroup.cjs';
import VirtualConsoleLogLevelEnum from './enums/VirtualConsoleLogLevelEnum.cjs';
import VirtualConsoleLogTypeEnum from './enums/VirtualConsoleLogTypeEnum.cjs';
export default interface IVirtualConsoleLogEntry {
    type: VirtualConsoleLogTypeEnum;
    level: VirtualConsoleLogLevelEnum;
    message: Array<string | object>;
    group: IVirtualConsoleLogGroup | null;
}
//# sourceMappingURL=IVirtualConsoleLogEntry.d.ts.map