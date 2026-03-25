import PermissionStatus from './PermissionStatus.cjs';
import BrowserWindow from '../window/BrowserWindow.cjs';
/**
 * Permissions API.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/Permissions.
 */
export default class Permissions {
    #private;
    /**
     * Constructor.
     *
     * @param window Window.
     */
    constructor(window: BrowserWindow);
    /**
     * Returns scroll restoration.
     *
     * @param permissionDescriptor Permission descriptor.
     * @param permissionDescriptor.name Permission name.
     * @param [permissionDescriptor.userVisibleOnly] User visible only.
     * @param [permissionDescriptor.sysex] Sysex.
     * @returns Permission status.
     */
    query(permissionDescriptor: {
        name: string;
        userVisibleOnly?: boolean;
        sysex?: boolean;
    }): Promise<PermissionStatus>;
}
//# sourceMappingURL=Permissions.d.ts.map