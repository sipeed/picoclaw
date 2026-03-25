import CookieSameSiteEnum from './enums/CookieSameSiteEnum.js';
import ICookie from './ICookie.js';

export default <ICookie>{
	// Required
	key: null,
	originURL: null,

	// Optional
	value: null,
	domain: '',
	path: '',
	expires: null,
	httpOnly: false,
	secure: false,
	sameSite: CookieSameSiteEnum.lax
};
