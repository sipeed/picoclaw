/**
 * Virtual console log type.
 */
enum VirtualConsoleLogTypeEnum {
	// Log
	log = 'log',
	table = 'table',
	trace = 'trace',
	dir = 'dir',
	dirxml = 'dirxml',
	group = 'group',
	groupCollapsed = 'groupCollapsed',
	debug = 'debug',
	timeLog = 'timeLog',

	// Info
	info = 'info',
	count = 'count',
	timeEnd = 'timeEnd',

	// Warning
	warn = 'warn',
	countReset = 'countReset',

	// Error
	error = 'error',
	assert = 'assert'
}
export default VirtualConsoleLogTypeEnum;
