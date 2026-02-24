import { useMemo, useState } from 'react';
import type { GearCatalogItem, GearType } from '../gearCatalogTypes';
import { getCatalogItemDisplayName } from '../gearCatalogTypes';
import type { BuildPart, BuildValidationError } from '../buildTypes';
import { useAuth } from '../hooks/useAuth';
import { CatalogSearchModal } from './CatalogSearchModal';

interface BuildBuilderProps {
  title: string;
  description: string;
  parts: BuildPart[];
  onTitleChange: (value: string) => void;
  onDescriptionChange: (value: string) => void;
  onPartsChange: (parts: BuildPart[]) => void;
  validationErrors?: BuildValidationError[];
  readOnly?: boolean;
  imagePreviewUrl?: string | null;
  onImageAction?: () => void;
  imageActionLabel?: string;
  imageHelperText?: string;
}

interface BuildRow {
  label: string;
  gearType: GearType;
  categoryKey: string;
  required?: boolean;
}

const REQUIRED_ROWS: BuildRow[] = [
  { label: 'Frame', gearType: 'frame', categoryKey: 'frame', required: true },
  { label: 'Motors', gearType: 'motor', categoryKey: 'motor', required: true },
  { label: 'Receiver', gearType: 'receiver', categoryKey: 'receiver', required: true },
  { label: 'VTX', gearType: 'vtx', categoryKey: 'vtx', required: true },
];

const POWER_ROWS: BuildRow[] = [
  { label: 'AIO', gearType: 'aio', categoryKey: 'aio' },
  { label: 'FC/ESC Stack', gearType: 'stack', categoryKey: 'stack' },
  { label: 'Flight Controller', gearType: 'fc', categoryKey: 'fc' },
  { label: 'ESC', gearType: 'esc', categoryKey: 'esc' },
];

const OPTIONAL_ROWS: BuildRow[] = [
  { label: 'Camera', gearType: 'camera', categoryKey: 'camera' },
  { label: 'Propellers', gearType: 'prop', categoryKey: 'prop' },
  { label: 'Antenna', gearType: 'antenna', categoryKey: 'antenna' },
  { label: 'GPS', gearType: 'gps', categoryKey: 'gps' },
  { label: 'Other', gearType: 'other', categoryKey: 'other' },
];

export function BuildBuilder({
  title,
  description,
  parts,
  onTitleChange,
  onDescriptionChange,
  onPartsChange,
  validationErrors,
  readOnly = false,
  imagePreviewUrl,
  onImageAction,
  imageActionLabel,
  imageHelperText,
}: BuildBuilderProps) {
  const { isAuthenticated } = useAuth();
  const [pickerGearType, setPickerGearType] = useState<GearType | null>(null);

  const partsByType = useMemo(() => {
    const map = new Map<GearType, BuildPart>();
    for (const part of parts) {
      map.set(part.gearType, part);
    }
    return map;
  }, [parts]);

  const errorMap = useMemo(() => {
    const map = new Map<string, string>();
    for (const err of validationErrors ?? []) {
      if (!map.has(err.category)) {
        map.set(err.category, err.message);
      }
    }
    return map;
  }, [validationErrors]);

  const hasAIO = Boolean(partsByType.get('aio')?.catalogItemId);
  const hasStack = Boolean(partsByType.get('stack')?.catalogItemId);
  const hasFC = Boolean(partsByType.get('fc')?.catalogItemId);
  const hasESC = Boolean(partsByType.get('esc')?.catalogItemId);
  const powerComplete = hasAIO || hasStack || (hasFC && hasESC);

  const upsertPart = (gearType: GearType, item: GearCatalogItem) => {
    const remaining = parts.filter((part) => part.gearType !== gearType);
    onPartsChange([
      ...remaining,
      {
        gearType,
        catalogItemId: item.id,
        catalogItem: {
          id: item.id,
          gearType: item.gearType,
          brand: item.brand,
          model: item.model,
          variant: item.variant,
          msrp: item.msrp,
          status: item.status,
          imageUrl: item.imageUrl,
        },
      },
    ]);
  };

  const removePart = (gearType: GearType) => {
    onPartsChange(parts.filter((part) => part.gearType !== gearType));
  };

  const renderRow = (row: BuildRow, options?: { showRequiredBadge?: boolean }) => {
    const part = partsByType.get(row.gearType);
    const selected = Boolean(part?.catalogItemId);
    const error = errorMap.get(row.categoryKey);

    return (
      <div key={row.gearType} className="rounded-lg border border-slate-700 bg-slate-800/60 p-3">
        <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
          <div className="min-w-0">
            <div className="flex flex-wrap items-center gap-2">
              <h4 className="text-sm font-medium text-white">{row.label}</h4>
              {options?.showRequiredBadge && row.required && (
                <span className="rounded bg-slate-700 px-2 py-0.5 text-[11px] uppercase tracking-wide text-slate-300">Required</span>
              )}
              <span className={`rounded px-2 py-0.5 text-[11px] uppercase tracking-wide ${selected ? 'bg-emerald-500/20 text-emerald-300' : 'bg-amber-500/20 text-amber-300'}`}>
                {selected ? 'Complete' : 'Missing'}
              </span>
            </div>
            <p className="mt-1 truncate text-sm text-slate-300">
              {part?.catalogItem ? getCatalogItemDisplayName(part.catalogItem as GearCatalogItem) : 'No part selected'}
            </p>
            {error && (
              <p className="mt-1 text-xs text-red-400">{error}</p>
            )}
          </div>
          {!readOnly && (
            <div className="flex w-full flex-wrap gap-2 sm:w-auto sm:justify-end">
              <button
                type="button"
                onClick={() => setPickerGearType(row.gearType)}
                className="rounded-md bg-primary-600 px-3 py-1.5 text-xs font-medium text-white transition hover:bg-primary-500 whitespace-nowrap"
              >
                Choose
              </button>
              {selected && (
                <button
                  type="button"
                  onClick={() => removePart(row.gearType)}
                  className="rounded-md border border-slate-600 px-3 py-1.5 text-xs font-medium text-slate-300 transition hover:border-slate-500 hover:text-white whitespace-nowrap"
                >
                  Clear
                </button>
              )}
            </div>
          )}
        </div>
      </div>
    );
  };

  return (
    <>
      <div className="space-y-4">
        <div className="space-y-3 rounded-xl border border-slate-700 bg-slate-800/60 p-4">
          <div className="grid gap-4 lg:grid-cols-[minmax(0,1fr),260px]">
            <div className="min-w-0 space-y-3">
              <div>
                <label className="mb-1 block text-sm font-medium text-slate-300">Build title</label>
                <input
                  value={title}
                  onChange={(event) => onTitleChange(event.target.value)}
                  disabled={readOnly}
                  className="w-full rounded-lg border border-slate-600 bg-slate-700 px-3 py-2 text-white focus:border-primary-500 focus:outline-none disabled:opacity-70"
                  placeholder={'My Freestyle 5"'}
                />
              </div>
              <div>
                <label className="mb-1 block text-sm font-medium text-slate-300">Description</label>
                <textarea
                  value={description}
                  onChange={(event) => onDescriptionChange(event.target.value)}
                  disabled={readOnly}
                  rows={3}
                  className="w-full rounded-lg border border-slate-600 bg-slate-700 px-3 py-2 text-white focus:border-primary-500 focus:outline-none disabled:opacity-70"
                  placeholder="Describe the goals, tune style, and intended use."
                />
              </div>
            </div>

            {(onImageAction || imagePreviewUrl) && (
              <div className="min-w-0 space-y-2 rounded-lg border border-slate-700 bg-slate-900/60 p-3">
                <p className="text-sm font-medium text-slate-300">Build image</p>
                <div className="aspect-[4/3] w-full overflow-hidden rounded-lg border border-slate-700 bg-slate-800">
                  {imagePreviewUrl ? (
                    <img src={imagePreviewUrl} alt={title || 'Build image'} className="h-full w-full object-cover" />
                  ) : (
                    <div className="flex h-full w-full items-center justify-center text-slate-500">
                      No image
                    </div>
                  )}
                </div>
                {!readOnly && onImageAction && (
                  <button
                    type="button"
                    onClick={onImageAction}
                    className="w-full rounded-md border border-slate-600 bg-slate-700 px-3 py-2 text-sm font-medium text-white transition hover:border-slate-500 hover:bg-slate-600"
                  >
                    {imageActionLabel ?? (imagePreviewUrl ? 'Change Image' : 'Upload Image')}
                  </button>
                )}
                {imageHelperText && (
                  <p className="text-xs text-slate-500">{imageHelperText}</p>
                )}
              </div>
            )}
          </div>
        </div>

        <section className="space-y-3">
          <h3 className="text-sm font-semibold uppercase tracking-wide text-slate-400">Core Required Parts</h3>
          {REQUIRED_ROWS.map((row) => renderRow(row, { showRequiredBadge: true }))}
        </section>

        <section className="space-y-3">
          <div className="flex items-center gap-2">
            <h3 className="text-sm font-semibold uppercase tracking-wide text-slate-400">Power Stack</h3>
            <span className="rounded bg-slate-700 px-2 py-0.5 text-[11px] uppercase tracking-wide text-slate-300">Required</span>
            <span className={`rounded px-2 py-0.5 text-[11px] uppercase tracking-wide ${powerComplete ? 'bg-emerald-500/20 text-emerald-300' : 'bg-amber-500/20 text-amber-300'}`}>
              {powerComplete ? 'Complete' : 'Missing'}
            </span>
          </div>
          <p className="text-xs text-slate-500">Pick an AIO, an FC/ESC stack, or select both FC and ESC.</p>
          {errorMap.get('power-stack') && (
            <p className="text-xs text-red-400">{errorMap.get('power-stack')}</p>
          )}
          {POWER_ROWS.map((row) => renderRow(row))}
        </section>

        <section className="space-y-3">
          <h3 className="text-sm font-semibold uppercase tracking-wide text-slate-400">Optional Parts</h3>
          {OPTIONAL_ROWS.map((row) => renderRow(row))}
        </section>
      </div>

      {pickerGearType && (
        <CatalogSearchModal
          isOpen
          onClose={() => setPickerGearType(null)}
          initialGearType={pickerGearType}
          showInventoryMatches={isAuthenticated}
          onSelectItem={(item) => {
            upsertPart(pickerGearType, item);
            setPickerGearType(null);
          }}
        />
      )}
    </>
  );
}
