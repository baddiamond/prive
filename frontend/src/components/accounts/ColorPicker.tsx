import { useTranslation } from "react-i18next";
import { accountColors, type AccountColor } from "../../lib/accounts";
import { accountBg } from "./colors";

interface ColorPickerProps {
  value: AccountColor;
  onChange: (color: AccountColor) => void;
}

/** Account color swatches. Each has a name, so color is never the only cue. */
export default function ColorPicker({ value, onChange }: ColorPickerProps) {
  const { t } = useTranslation();
  return (
    <fieldset className="flex flex-col gap-2">
      <legend className="mb-2 text-caption font-semibold text-text-secondary">{t("link.accountColor")}</legend>
      <div className="flex flex-wrap gap-2.5">
        {accountColors.map((c) => {
          const selected = c === value;
          const name = t(`link.colors.${c}`);
          return (
            <button
              key={c}
              type="button"
              aria-label={name}
              title={name}
              aria-pressed={selected}
              onClick={() => onChange(c)}
              className={`h-8 w-8 rounded-full ${accountBg[c]} ${
                selected ? "shadow-[0_0_0_2px_var(--prive-bg),0_0_0_4px_var(--prive-text)]" : ""
              }`}
            />
          );
        })}
      </div>
    </fieldset>
  );
}
