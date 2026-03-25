export default interface IVirtualConsoleLogGroup {
	id: number;
	label: string;
	collapsed: boolean;
	parent: IVirtualConsoleLogGroup;
}
