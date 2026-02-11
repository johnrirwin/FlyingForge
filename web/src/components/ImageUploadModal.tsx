import { useRef, type ReactNode } from 'react';

export type UploadStatusTone = 'neutral' | 'success' | 'error';

interface ImageUploadModalProps {
  isOpen: boolean;
  title: string;
  previewUrl?: string | null;
  previewAlt: string;
  placeholder?: ReactNode;
  accept: string;
  helperText: string;
  selectButtonLabel: string;
  onSelectFile: (file: File) => void | Promise<void>;
  onClose: () => void;
  onSave: () => void;
  disableSelect?: boolean;
  disableClose?: boolean;
  disableSave?: boolean;
  saveLabel?: string;
  cancelLabel?: string;
  statusText?: string | null;
  statusTone?: UploadStatusTone;
  statusReason?: string;
  errorMessage?: string | null;
  zIndexClassName?: string;
}

export function ImageUploadModal({
  isOpen,
  title,
  previewUrl,
  previewAlt,
  placeholder,
  accept,
  helperText,
  selectButtonLabel,
  onSelectFile,
  onClose,
  onSave,
  disableSelect = false,
  disableClose = false,
  disableSave = false,
  saveLabel = 'Save',
  cancelLabel = 'Cancel',
  statusText = null,
  statusTone = 'neutral',
  statusReason,
  errorMessage = null,
  zIndexClassName = 'z-[70]',
}: ImageUploadModalProps) {
  const fileInputRef = useRef<HTMLInputElement>(null);

  if (!isOpen) return null;

  return (
    <div className={`fixed inset-0 bg-black/60 flex items-center justify-center p-4 ${zIndexClassName}`}>
      <div className="bg-slate-800 rounded-xl p-6 max-w-md w-full shadow-2xl border border-slate-700">
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-lg font-semibold text-white">{title}</h3>
          <button
            type="button"
            onClick={onClose}
            disabled={disableClose}
            className="p-2 text-slate-400 hover:text-white hover:bg-slate-700 rounded-lg transition-colors disabled:opacity-50"
          >
            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        <div className="flex flex-col items-center gap-4 mb-4">
          <div className="w-36 h-36 rounded-lg overflow-hidden border-2 border-slate-600 bg-slate-700">
            {previewUrl ? (
              <img src={previewUrl} alt={previewAlt} className="w-full h-full object-cover" />
            ) : (
              <div className="w-full h-full flex items-center justify-center text-slate-500 text-4xl">
                {placeholder ?? '📦'}
              </div>
            )}
          </div>

          <input
            ref={fileInputRef}
            type="file"
            accept={accept}
            onChange={(event) => {
              const file = event.target.files?.[0];
              event.target.value = '';
              if (!file) return;
              void onSelectFile(file);
            }}
            className="hidden"
          />
          <button
            type="button"
            onClick={() => fileInputRef.current?.click()}
            disabled={disableSelect}
            className="px-4 py-2 bg-slate-700 hover:bg-slate-600 text-white rounded-lg text-sm transition-colors disabled:opacity-50"
          >
            {selectButtonLabel}
          </button>
          <p className="text-xs text-slate-500 text-center">{helperText}</p>
        </div>

        {errorMessage && (
          <div className="mb-4 p-3 rounded-lg text-sm border bg-red-500/10 border-red-500/30 text-red-400">
            <p>{errorMessage}</p>
          </div>
        )}

        {statusText && (
          <div
            className={`mb-4 p-3 rounded-lg text-sm border ${
              statusTone === 'success'
                ? 'bg-green-500/10 border-green-500/30 text-green-400'
                : statusTone === 'error'
                  ? 'bg-red-500/10 border-red-500/30 text-red-400'
                  : 'bg-slate-700/50 border-slate-600 text-slate-300'
            }`}
          >
            <p>{statusText}</p>
            {statusReason && statusTone !== 'success' && (
              <p className="mt-1 text-xs text-slate-300">{statusReason}</p>
            )}
          </div>
        )}

        <div className="flex gap-2">
          <button
            type="button"
            onClick={onClose}
            disabled={disableClose}
            className="flex-1 px-4 py-2 bg-slate-700 hover:bg-slate-600 text-white rounded-lg text-sm transition-colors disabled:opacity-50"
          >
            {cancelLabel}
          </button>
          <button
            type="button"
            onClick={onSave}
            disabled={disableSave}
            className="flex-1 px-4 py-2 bg-primary-600 hover:bg-primary-500 text-white rounded-lg text-sm transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {saveLabel}
          </button>
        </div>
      </div>
    </div>
  );
}
