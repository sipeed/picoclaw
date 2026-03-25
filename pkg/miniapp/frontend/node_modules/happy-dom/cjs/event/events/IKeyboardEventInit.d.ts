import IUIEventInit from '../IUIEventInit.cjs';
export default interface IKeyboardEventInit extends IUIEventInit {
    key?: string;
    code?: string;
    location?: number;
    ctrlKey?: boolean;
    shiftKey?: boolean;
    altKey?: boolean;
    metaKey?: boolean;
    repeat?: boolean;
    isComposing?: boolean;
    /**
     * @deprecated
     */
    keyCode?: number;
}
//# sourceMappingURL=IKeyboardEventInit.d.ts.map