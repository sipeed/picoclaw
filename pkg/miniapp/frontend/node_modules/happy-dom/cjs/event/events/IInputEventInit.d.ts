import DataTransfer from '../DataTransfer.cjs';
import IUIEventInit from '../IUIEventInit.cjs';
export default interface IInputEventInit extends IUIEventInit {
    inputType?: string;
    data?: string;
    dataTransfer?: DataTransfer;
    isComposing?: boolean;
}
//# sourceMappingURL=IInputEventInit.d.ts.map