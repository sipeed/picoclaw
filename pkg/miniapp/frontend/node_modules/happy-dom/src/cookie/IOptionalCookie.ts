import CookieSameSiteEnum from './enums/CookieSameSiteEnum.js';

export default interface IOptionalCookie {
	// Required
	key: string;
	originURL: URL;

	// Optional
	value?: string | null;
	domain?: string;
	path?: string;
	expires?: Date | null;
	httpOnly?: boolean;
	secure?: boolean;
	sameSite?: CookieSameSiteEnum;
}
