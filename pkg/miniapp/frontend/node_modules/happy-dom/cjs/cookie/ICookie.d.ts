import CookieSameSiteEnum from './enums/CookieSameSiteEnum.cjs';
export default interface ICookie {
    key: string;
    originURL: URL;
    value: string | null;
    domain: string;
    path: string;
    expires: Date | null;
    httpOnly: boolean;
    secure: boolean;
    sameSite: CookieSameSiteEnum;
}
//# sourceMappingURL=ICookie.d.ts.map