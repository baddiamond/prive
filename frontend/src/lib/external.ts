import { BrowserOpenURL } from "../../wailsjs/runtime/runtime";

/** Project documents linked from the UI (opened in the system browser). */
export const projectLinks = {
  disclaimer: "https://github.com/baddiamond/prive/blob/main/DISCLAIMER.md",
  acceptableUse: "https://github.com/baddiamond/prive/blob/main/ACCEPTABLE_USE.md",
  privacy: "https://github.com/baddiamond/prive/blob/main/PRIVACY.md",
  addingAnEngine: "https://github.com/baddiamond/prive/blob/main/docs/ADDING_AN_ENGINE.md",
  engineProposal: "https://github.com/baddiamond/prive/issues/new?template=engine_proposal.md",
} as const;

/** Opens a link in the system browser, never inside the app window. */
export function openExternal(url: string): void {
  BrowserOpenURL(url);
}
