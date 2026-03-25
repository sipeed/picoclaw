import DataTransfer from '../DataTransfer.js';
import IUIEventInit from '../IUIEventInit.js';
export default interface IInputEventInit extends IUIEventInit {
    inputType?: string;
    data?: string;
    dataTransfer?: DataTransfer;
    isComposing?: boolean;
}
//# sourceMappingURL=IInputEventInit.d.ts.map