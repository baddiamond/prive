import { useTranslation } from "react-i18next";
import type { Manifest } from "../../lib/accounts";
import { plannedEngines } from "./plannedEngines";

interface NetworkGridProps {
  engines: Manifest[];
  selected: string | null;
  onSelect: (engineId: string) => void;
}

/** Bundled networks first and selectable, then planned ones as "Coming soon" (disabled). */
export default function NetworkGrid({ engines, selected, onSelect }: NetworkGridProps) {
  const { t } = useTranslation();
  const bundled = new Set(engines.map((e) => e.id));
  const items = [
    ...engines.map((e) => ({ id: e.id, name: e.name, shortName: e.shortName, available: true })),
    ...plannedEngines.filter((p) => !bundled.has(p.id)).map((p) => ({ ...p, available: false })),
  ];

  return (
    <div className="grid grid-cols-[repeat(auto-fill,minmax(170px,1fr))] gap-2.5">
      {items.map((item) => {
        const isSelected = item.id === selected;
        return (
          <button
            key={item.id}
            type="button"
            disabled={!item.available}
            aria-pressed={item.available ? isSelected : undefined}
            onClick={() => onSelect(item.id)}
            className={`flex min-h-[76px] flex-col items-start justify-between gap-2 rounded-card border-[1.5px] p-3 text-left transition-colors duration-fast ease-prive ${
              isSelected
                ? "border-accent bg-accent-tint"
                : item.available
                  ? "border-border bg-surface hover:bg-raised"
                  : "cursor-default border-divider text-text-secondary"
            }`}
          >
            <span className="flex items-center gap-2">
              <span className="rounded-tag border border-border px-1.5 text-micro font-bold">{item.shortName}</span>
              <span className="font-semibold">{item.name}</span>
            </span>
            <span
              className={`rounded-tag px-2 py-0.5 text-micro font-bold ${
                item.available ? "bg-accent-tint text-accent" : "bg-raised text-text-secondary"
              }`}
            >
              {isSelected ? t("link.selected") : item.available ? t("link.available") : t("common.comingSoon")}
            </span>
          </button>
        );
      })}
    </div>
  );
}
