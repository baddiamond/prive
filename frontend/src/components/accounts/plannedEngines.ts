/**
 * Networks shown as "Coming soon" in the network grid (docs/ONBOARDING.md, step 3).
 * Brand names are not translated. Bundled engines are hidden from this list automatically.
 */
export const plannedEngines: readonly { id: string; name: string; shortName: string }[] = [
  { id: "telegram", name: "Telegram", shortName: "TG" },
  { id: "signal", name: "Signal", shortName: "SG" },
  { id: "slack", name: "Slack", shortName: "SL" },
  { id: "instagram", name: "Instagram", shortName: "IG" },
  { id: "messenger", name: "Messenger", shortName: "MS" },
  { id: "discord", name: "Discord", shortName: "DC" },
  { id: "gmessages", name: "Google Messages", shortName: "GM" },
  { id: "linkedin", name: "LinkedIn", shortName: "IN" },
  { id: "matrix", name: "Matrix", shortName: "MX" },
];
