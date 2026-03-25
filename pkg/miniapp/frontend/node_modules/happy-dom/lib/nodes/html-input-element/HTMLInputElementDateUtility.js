/**
 * Date utility for HTML input elements.
 */
export default class HTMLInputElementDateUtility {
    /**
     * Returns iso week number from given date
     *
     * @see https://stackoverflow.com/a/6117889
     * @param date Date or number.
     * @returns Iso-week string.
     */
    static dateIsoWeek(date) {
        date = typeof date === 'number' ? new Date(date) : date;
        // Copy date so don't modify original
        date = new Date(Date.UTC(date.getFullYear(), date.getMonth(), date.getDate()));
        // Set to nearest Thursday: current date + 4 - current day number
        // Make Sunday's day number 7
        date.setUTCDate(date.getUTCDate() + 4 - (date.getUTCDay() || 7));
        // Get first day of year
        const yearStart = new Date(Date.UTC(date.getUTCFullYear(), 0, 1));
        // Calculate full weeks to nearest Thursday
        const weekNo = Math.ceil(((date - yearStart) / 86400000 + 1) / 7);
        return `${date.getUTCFullYear()}-W${weekNo < 10 ? '0' : ''}${weekNo}`;
    }
    /**
     * Returns a date object for monday of given iso week string (\d\d\d\d-W\d\d)
     *
     * @param isoWeek Iso-week string.
     * @returns Date.
     */
    static isoWeekDate(isoWeek) {
        // Algorythm adapted from https://en.wikipedia.org/wiki/ISO_week_date
        const [, Y, W] = isoWeek.match(/^(\d{4})-W(\d{2})$/) || [];
        if (!Y || !W || Number(W) > 53 || Number(W) < 1) {
            return new Date('x'); // Return an invalid date
        }
        const date = new Date(`${Y}-01-01T00:00Z`);
        const jan4th = new Date(`${Y}-01-04T00:00Z`);
        const jan4thDay = (jan4th.getUTCDay() + 6) % 7;
        const ordinalDate = 1 + (Number(W) - 1) * 7 - jan4thDay + 3;
        date.setUTCDate(ordinalDate);
        if (date.getUTCFullYear() > Number(Y)) {
            return new Date('x'); // Return an invalid date
        }
        return date;
    }
}
//# sourceMappingURL=HTMLInputElementDateUtility.js.map