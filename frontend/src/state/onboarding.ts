import { create } from "zustand";
import {
  acceptLegal,
  completeOnboarding,
  getOnboarding,
  setOnboardingStep,
  type Onboarding,
} from "../lib/accounts";

/** Onboarding steps (docs/ONBOARDING.md). */
export const onboardingSteps = ["welcome", "privacy", "connect", "link", "ready"] as const;
export type OnboardingStep = (typeof onboardingSteps)[number];

interface OnboardingState {
  loaded: boolean;
  state: Onboarding | null;
  /** Current step, 1-based. */
  step: number;
  /** Network chosen in step 3, or null when skipped. */
  engineId: string | null;
  /** Account linked in step 4, shown in step 5. */
  linkedAccountId: string | null;
  load: () => Promise<void>;
  goTo: (step: number) => void;
  setEngine: (engineId: string | null) => void;
  setLinkedAccount: (accountId: string | null) => void;
  accept: () => Promise<void>;
  finish: () => Promise<void>;
}

/** First-run flow state. The step is saved so closing the window resumes there. */
export const useOnboarding = create<OnboardingState>()((set, get) => ({
  loaded: false,
  state: null,
  step: 1,
  engineId: null,
  linkedAccountId: null,

  load: async () => {
    const state = await getOnboarding();
    // Linking can't resume after a restart (the half-created account is removed), so a
    // run that stopped on step 4 goes back to choosing the network.
    const step = state.step === 4 ? 3 : state.step;
    set({ state, step, loaded: true });
  },

  goTo: (step) => {
    set({ step });
    void setOnboardingStep(step);
  },

  setEngine: (engineId) => set({ engineId }),
  setLinkedAccount: (linkedAccountId) => set({ linkedAccountId }),

  accept: async () => {
    await acceptLegal();
    const state = get().state;
    if (state) set({ state: { ...state, legalAccepted: true } });
  },

  finish: async () => {
    await completeOnboarding();
    const state = get().state;
    if (state) set({ state: { ...state, completed: true } });
  },
}));

/** Whether the inbox is allowed: onboarding done and the current legal version accepted. */
export function needsOnboarding(state: Onboarding | null): boolean {
  return !state || !state.completed || !state.legalAccepted;
}
