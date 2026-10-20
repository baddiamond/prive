import { useEffect, useId, useRef, useState, type ReactNode } from "react";
import { useTranslation } from "react-i18next";
import { languageCodes, resolveLanguage } from "../../i18n";
import type { ThemePreference } from "../../lib/api";
import type { AccountColor } from "../../lib/accounts";
import { errorCode, translateCode } from "../../lib/errors";
import { projectLinks } from "../../lib/external";
import { shortcutLabel } from "../../lib/platform";
import { useAccounts } from "../../state/accounts";
import { onboardingSteps, useOnboarding } from "../../state/onboarding";
import { useSettings } from "../../state/settings";
import AccountTile from "../accounts/AccountTile";
import { nextColor, saveIdentity } from "../accounts/identity";
import LinkPanel from "../accounts/LinkPanel";
import NetworkGrid from "../accounts/NetworkGrid";
import { useLinkFlow, type LinkFlow } from "../accounts/useLinkFlow";
import ExternalLink from "../common/ExternalLink";
import { CheckIcon, LinkIcon, LockIcon, PersonIcon } from "../icons";
import LogoMark from "../LogoMark";

/** First run (docs/ONBOARDING.md): welcome, before you start, connect, link, ready. */
export default function Onboarding() {
  const state = useOnboarding((s) => s.state);
  // Onboarding already done, but the legal documents changed: only ask to agree again.
  if (state?.completed && !state.legalAccepted) return <ReAccept />;
  return <Flow />;
}

function Flow() {
  const { t } = useTranslation();
  const step = useOnboarding((s) => s.step);
  const goTo = useOnboarding((s) => s.goTo);
  const engineId = useOnboarding((s) => s.engineId);
  const setEngine = useOnboarding((s) => s.setEngine);
  const accept = useOnboarding((s) => s.accept);
  const setLinked = useOnboarding((s) => s.setLinkedAccount);
  const legalAccepted = useOnboarding((s) => s.state?.legalAccepted ?? false);
  const engines = useAccounts((s) => s.engines);
  const accounts = useAccounts((s) => s.accounts);

  const [agreed, setAgreed] = useState(legalAccepted);
  const [name, setName] = useState("");
  const [color, setColor] = useState<AccountColor>(() => nextColor(accounts.map((s) => s.account.color)));
  const [error, setError] = useState<string | null>(null);
  const [linkFlow, setLinkFlow] = useState<LinkFlow | null>(null);
  const headingRef = useRef<HTMLHeadingElement>(null);

  // Each step has one h1; focus moves to it when the step changes.
  useEffect(() => headingRef.current?.focus(), [step]);

  const next = async () => {
    setError(null);
    try {
      if (step === 2) await accept();
      if (step === 4 && linkFlow?.accountId) {
        await saveIdentity(linkFlow.accountId, name, color);
        setLinked(linkFlow.accountId);
      }
      goTo(step + 1);
    } catch (err) {
      setError(errorCode(err));
    }
  };

  const canContinue = (step === 2 && agreed) || (step === 3 && engineId !== null) || (step === 4 && linkFlow?.phase === "linked");

  return (
    <div className="flex h-full flex-col items-center overflow-y-auto bg-bg px-4 py-8">
      <Stepper current={step} />
      <main className="flex w-full max-w-[760px] flex-1 flex-col">
        {step === 1 && <WelcomeStep headingRef={headingRef} onStart={() => goTo(2)} />}
        {step === 2 && <PrivacyStep headingRef={headingRef} agreed={agreed} onAgree={setAgreed} />}
        {step === 3 && (
          <StepLayout headingRef={headingRef} title={t("onboarding.connect.title")} body={t("onboarding.connect.body")}>
            <NetworkGrid engines={engines} selected={engineId} onSelect={setEngine} />
            <p className="text-caption text-text-secondary">
              {t("onboarding.connect.suggest")}{" "}
              <ExternalLink href={projectLinks.engineProposal}>{t("onboarding.connect.suggestLink")}</ExternalLink>
            </p>
          </StepLayout>
        )}
        {step === 4 && engineId && (
          <LinkStep
            headingRef={headingRef}
            engineId={engineId}
            name={name}
            color={color}
            onName={setName}
            onColor={setColor}
            onFlow={setLinkFlow}
          />
        )}
        {step === 5 && <ReadyStep headingRef={headingRef} onAddAnother={() => {
          setEngine(null);
          setName("");
          setColor(nextColor(useAccounts.getState().accounts.map((s) => s.account.color)));
          goTo(3);
        }} />}

        {error && (
          <p role="alert" className="mt-4 text-status-danger">
            {translateCode(t, error)}
          </p>
        )}

        {step >= 2 && step <= 4 && (
          <div className="mt-8 flex items-center gap-3 border-t border-divider pt-5">
            <button
              type="button"
              onClick={() => goTo(step - 1)}
              className="h-10 rounded-control border-[1.5px] border-border px-5 font-semibold hover:bg-raised"
            >
              {t("common.back")}
            </button>
            <span className="flex-1" />
            {step === 3 && (
              <button
                type="button"
                onClick={() => {
                  setEngine(null);
                  setLinked(null);
                  goTo(5);
                }}
                className="h-10 px-4 font-semibold text-text-secondary hover:text-text"
              >
                {t("onboarding.connect.skip")}
              </button>
            )}
            <button
              type="button"
              disabled={!canContinue}
              onClick={() => void next()}
              className="h-10 rounded-control bg-accent px-6 font-bold text-on-accent disabled:cursor-not-allowed disabled:opacity-50"
            >
              {t("common.continue")}
            </button>
          </div>
        )}
      </main>
    </div>
  );
}

function Stepper({ current }: { current: number }) {
  const { t } = useTranslation();
  return (
    <nav aria-label={t("onboarding.progress")} className="mb-7 w-full max-w-[760px]">
      <ol className="flex flex-wrap items-center justify-center gap-2">
        {onboardingSteps.map((s, i) => {
          const n = i + 1;
          const done = n < current;
          const active = n === current;
          return (
            <li key={s} aria-current={active ? "step" : undefined} className="flex items-center gap-2">
              <span
                className={`flex h-7 w-7 items-center justify-center rounded-full border-[1.5px] text-caption font-bold ${
                  done
                    ? "border-accent bg-accent text-on-accent"
                    : active
                      ? "border-accent text-accent"
                      : "border-border text-text-faint"
                }`}
              >
                {done ? <CheckIcon size={14} /> : n}
              </span>
              <span className={`hidden text-caption sm:inline ${active ? "font-bold text-text" : "font-medium text-text-secondary"}`}>
                {t(`onboarding.steps.${s}`)}
              </span>
              {n < onboardingSteps.length && <span aria-hidden className="mx-1 h-px w-6 bg-divider" />}
            </li>
          );
        })}
      </ol>
    </nav>
  );
}

interface HeadingProps {
  headingRef: React.RefObject<HTMLHeadingElement | null>;
}

function StepLayout({ headingRef, title, body, children }: HeadingProps & { title: string; body: string; children: ReactNode }) {
  return (
    <section className="flex flex-col gap-6">
      <div className="flex flex-col gap-2">
        <h1 ref={headingRef} tabIndex={-1} className="text-display font-bold tracking-[-0.02em] outline-none">
          {title}
        </h1>
        <p className="text-body-strong text-text-secondary">{body}</p>
      </div>
      {children}
    </section>
  );
}

function WelcomeStep({ headingRef, onStart }: HeadingProps & { onStart: () => void }) {
  const { t } = useTranslation();
  const language = useSettings((s) => s.language);
  const systemLocale = useSettings((s) => s.systemLocale);
  const setLanguage = useSettings((s) => s.setLanguage);
  const theme = useSettings((s) => s.theme);
  const setTheme = useSettings((s) => s.setTheme);
  const languageId = useId();
  const themeId = useId();
  const current = language || resolveLanguage("", systemLocale);

  return (
    <section className="flex flex-1 flex-col items-center justify-center gap-7 text-center">
      <LogoMark width={96} height={96} />
      <div className="flex flex-col gap-3">
        <h1 ref={headingRef} tabIndex={-1} className="text-display font-bold tracking-[-0.02em] outline-none">
          {t("onboarding.welcome.title")}
        </h1>
        <p className="mx-auto max-w-[440px] text-body-strong text-text-secondary">{t("onboarding.welcome.body")}</p>
      </div>
      <div className="flex w-full max-w-[420px] flex-col gap-5 text-left">
        <fieldset className="flex flex-col gap-2">
          <legend id={languageId} className="mb-2 text-caption font-semibold text-text-secondary">
            {t("onboarding.welcome.language")}
            {!language && <span className="font-normal"> · {t("onboarding.welcome.detected")}</span>}
          </legend>
          <div className="grid grid-cols-2 gap-2">
            {languageCodes.map((code) => (
              <ChoiceButton key={code} selected={current === code} onClick={() => void setLanguage(code)}>
                {t(`settings.language.options.${code}`)}
              </ChoiceButton>
            ))}
          </div>
        </fieldset>
        <fieldset className="flex flex-col gap-2">
          <legend id={themeId} className="mb-2 text-caption font-semibold text-text-secondary">
            {t("onboarding.welcome.appearance")}
          </legend>
          <div className="grid grid-cols-3 gap-2">
            {(["system", "light", "dark"] as ThemePreference[]).map((option) => (
              <ChoiceButton key={option} selected={theme === option} onClick={() => void setTheme(option)}>
                {t(`settings.theme.options.${option}`)}
              </ChoiceButton>
            ))}
          </div>
        </fieldset>
      </div>
      <button type="button" onClick={onStart} className="h-12 rounded-control bg-accent px-8 text-body-strong font-bold text-on-accent">
        {t("onboarding.welcome.getStarted")}
      </button>
      <span className="text-caption text-text-faint">{t("onboarding.welcome.footer")}</span>
    </section>
  );
}

function ChoiceButton({ selected, onClick, children }: { selected: boolean; onClick: () => void; children: ReactNode }) {
  return (
    <button
      type="button"
      aria-pressed={selected}
      onClick={onClick}
      className={`h-10 rounded-control border-[1.5px] font-medium transition-colors duration-fast ease-prive ${
        selected ? "border-accent bg-accent-tint" : "border-border hover:bg-raised"
      }`}
    >
      {children}
    </button>
  );
}

function PrivacyStep({ headingRef, agreed, onAgree }: HeadingProps & { agreed: boolean; onAgree: (v: boolean) => void }) {
  const { t } = useTranslation();
  const facts = [
    { key: "local", icon: <LockIcon /> },
    { key: "unofficial", icon: <LinkIcon /> },
    { key: "personal", icon: <PersonIcon /> },
  ] as const;

  return (
    <StepLayout headingRef={headingRef} title={t("onboarding.privacy.title")} body={t("onboarding.privacy.body")}>
      <div className="flex flex-col gap-3">
        {facts.map((f) => (
          <div key={f.key} className="flex items-start gap-3 rounded-card border border-divider bg-surface p-4">
            <span className="flex h-10 w-10 shrink-0 items-center justify-center rounded-control bg-accent-tint text-accent">
              {f.icon}
            </span>
            <span className="flex flex-col gap-1">
              <strong className="font-semibold">{t(`onboarding.privacy.facts.${f.key}.title`)}</strong>
              <span className="leading-relaxed text-text-secondary">{t(`onboarding.privacy.facts.${f.key}.body`)}</span>
            </span>
          </div>
        ))}
      </div>
      <label className="flex cursor-pointer items-start gap-3 rounded-card border-[1.5px] border-border p-4">
        <input
          type="checkbox"
          checked={agreed}
          onChange={(e) => onAgree(e.target.checked)}
          className="mt-0.5 h-5 w-5 shrink-0 accent-[var(--prive-accent)]"
        />
        <span>{t("onboarding.privacy.agree")}</span>
      </label>
      <div className="flex flex-wrap gap-4 text-caption">
        <ExternalLink href={projectLinks.disclaimer}>{t("legal.disclaimer")}</ExternalLink>
        <ExternalLink href={projectLinks.acceptableUse}>{t("legal.acceptableUse")}</ExternalLink>
        <ExternalLink href={projectLinks.privacy}>{t("legal.privacy")}</ExternalLink>
      </div>
    </StepLayout>
  );
}

interface LinkStepProps extends HeadingProps {
  engineId: string;
  name: string;
  color: AccountColor;
  onName: (name: string) => void;
  onColor: (color: AccountColor) => void;
  onFlow: (flow: LinkFlow) => void;
}

function LinkStep({ headingRef, engineId, name, color, onName, onColor, onFlow }: LinkStepProps) {
  const { t } = useTranslation();
  const manifest = useAccounts((s) => s.engines.find((e) => e.id === engineId));
  const flow = useLinkFlow(engineId, color);
  useEffect(() => onFlow(flow), [flow, onFlow]);
  if (!manifest) return null;
  return (
    <StepLayout headingRef={headingRef} title={t("link.title", { network: manifest.name })} body={t("link.body")}>
      <LinkPanel manifest={manifest} flow={flow} name={name} color={color} onName={onName} onColor={onColor} />
    </StepLayout>
  );
}

function ReadyStep({ headingRef, onAddAnother }: HeadingProps & { onAddAnother: () => void }) {
  const { t } = useTranslation();
  const finish = useOnboarding((s) => s.finish);
  const linkedId = useOnboarding((s) => s.linkedAccountId);
  const linked = useAccounts((s) => s.accounts.find((a) => a.account.id === linkedId));
  const [error, setError] = useState<string | null>(null);

  const tips = [
    { key: shortcutLabel("K"), label: t("onboarding.ready.tips.search") },
    { key: shortcutLabel("1–9"), label: t("onboarding.ready.tips.jump") },
    { key: shortcutLabel("N"), label: t("onboarding.ready.tips.add") },
  ];

  return (
    <section className="flex flex-1 flex-col items-center justify-center gap-6 text-center">
      <span className="flex h-16 w-16 items-center justify-center rounded-full bg-accent text-on-accent">
        <CheckIcon size={34} />
      </span>
      <div className="flex flex-col gap-2">
        <h1 ref={headingRef} tabIndex={-1} className="text-display font-bold tracking-[-0.02em] outline-none">
          {linked ? t("onboarding.ready.title") : t("onboarding.ready.skippedTitle")}
        </h1>
        <p className="mx-auto max-w-[460px] text-body-strong text-text-secondary">
          {linked ? t("onboarding.ready.body") : t("onboarding.ready.skippedBody")}
        </p>
      </div>
      {linked && (
        <div className="flex items-center gap-3 rounded-card border border-divider bg-surface px-4 py-3 text-left">
          <AccountTile label={linked.account.displayName || linked.manifest.name} color={linked.account.color} size={40} />
          <span className="flex flex-col">
            <span className="font-semibold">
              {linked.account.displayName || linked.manifest.name} · {linked.manifest.name}
            </span>
            <span className="text-caption text-text-secondary">{t(`account.state.${linked.state}`)}</span>
          </span>
        </div>
      )}
      <ul className="flex flex-wrap justify-center gap-2">
        {tips.map((tip) => (
          <li key={tip.key} className="flex items-center gap-2 rounded-control border border-divider px-3 py-2 text-caption text-text-secondary">
            <kbd className="rounded-tag border border-border px-1.5 py-0.5 font-mono text-micro font-semibold text-text">{tip.key}</kbd>
            {tip.label}
          </li>
        ))}
      </ul>
      {error && (
        <p role="alert" className="text-status-danger">
          {translateCode(t, error)}
        </p>
      )}
      <div className="flex flex-wrap justify-center gap-3">
        <button
          type="button"
          onClick={() => void finish().catch((err: unknown) => setError(errorCode(err)))}
          className="h-11 rounded-control bg-accent px-6 font-bold text-on-accent"
        >
          {t("onboarding.ready.openInbox")}
        </button>
        <button type="button" onClick={onAddAnother} className="h-11 rounded-control border-[1.5px] border-border px-6 font-semibold hover:bg-raised">
          {t("onboarding.ready.addAnother")}
        </button>
      </div>
    </section>
  );
}

/** Shown when the legal documents changed after onboarding: agree again to continue. */
function ReAccept() {
  const { t } = useTranslation();
  const accept = useOnboarding((s) => s.accept);
  const [agreed, setAgreed] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const headingRef = useRef<HTMLHeadingElement>(null);
  useEffect(() => headingRef.current?.focus(), []);

  return (
    <div className="flex h-full flex-col items-center overflow-y-auto bg-bg px-4 py-12">
      <main className="flex w-full max-w-[760px] flex-col">
        <PrivacyStep headingRef={headingRef} agreed={agreed} onAgree={setAgreed} />
        {error && (
          <p role="alert" className="mt-4 text-status-danger">
            {translateCode(t, error)}
          </p>
        )}
        <div className="mt-8 flex justify-end border-t border-divider pt-5">
          <button
            type="button"
            disabled={!agreed}
            onClick={() => void accept().catch((err: unknown) => setError(errorCode(err)))}
            className="h-10 rounded-control bg-accent px-6 font-bold text-on-accent disabled:cursor-not-allowed disabled:opacity-50"
          >
            {t("common.continue")}
          </button>
        </div>
      </main>
    </div>
  );
}
