/**
 * Formats a chat list time like messaging apps do: the time today, "yesterday", the
 * weekday within the last week, then a short date. Uses the UI language's conventions.
 */
export function formatChatTime(date: Date, locale: string, yesterday: string, now = new Date()): string {
  const startOfDay = (d: Date) => new Date(d.getFullYear(), d.getMonth(), d.getDate()).getTime();
  const days = Math.round((startOfDay(now) - startOfDay(date)) / 86_400_000);
  if (days <= 0) {
    return new Intl.DateTimeFormat(locale, { hour: "2-digit", minute: "2-digit" }).format(date);
  }
  if (days === 1) return yesterday;
  if (days < 7) return new Intl.DateTimeFormat(locale, { weekday: "short" }).format(date);
  return new Intl.DateTimeFormat(locale, { day: "numeric", month: "short", year: days > 300 ? "2-digit" : undefined }).format(date);
}
