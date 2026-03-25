import IEventInit from '../IEventInit.cjs';
export default interface IErrorEventInit extends IEventInit {
    message?: string;
    filename?: string;
    lineno?: number;
    colno?: number;
    error?: Error;
}
//# sourceMappingURL=IErrorEventInit.d.ts.map