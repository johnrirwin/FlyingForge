import { useState, useEffect, useCallback, useRef } from 'react';
import type { Radio, RadioBackup, RadioModel, CreateRadioParams, CreateRadioBackupParams, BackupType, FirmwareFamily } from '../radioTypes';
import { BACKUP_TYPES, FIRMWARE_FAMILIES, formatFileSize, getBackupTypeLabel } from '../radioTypes';
import { getRadioModels, listRadios, createRadio, updateRadio, deleteRadio, listBackups, createBackup, downloadBackup, deleteBackup } from '../radioApi';

interface RadioSectionProps {
  onError?: (message: string) => void;
}

export function RadioSection({ onError }: RadioSectionProps) {
  // Radio state
  const [radios, setRadios] = useState<Radio[]>([]);
  const [selectedRadio, setSelectedRadio] = useState<Radio | null>(null);
  const [radioModels, setRadioModels] = useState<RadioModel[]>([]);
  const [isRadioLoading, setIsRadioLoading] = useState(true);
  const [, setRadioError] = useState<string | null>(null);

  // Backup state
  const [backups, setBackups] = useState<RadioBackup[]>([]);
  const [isBackupsLoading, setIsBackupsLoading] = useState(false);
  const [backupsError, setBackupsError] = useState<string | null>(null);

  // UI state
  const [showRadioSelector, setShowRadioSelector] = useState(false);
  const [showUploadModal, setShowUploadModal] = useState(false);
  const [isEditing, setIsEditing] = useState(false);
  const [editNotes, setEditNotes] = useState('');
  const [editFirmware, setEditFirmware] = useState<FirmwareFamily | ''>('');

  // Upload state
  const [uploadName, setUploadName] = useState('');
  const [uploadType, setUploadType] = useState<BackupType>('full-backup');
  const [uploadFile, setUploadFile] = useState<File | null>(null);
  const [isUploading, setIsUploading] = useState(false);
  const [uploadProgress, setUploadProgress] = useState(0);
  const fileInputRef = useRef<HTMLInputElement>(null);

  // Load radio models on mount
  useEffect(() => {
    getRadioModels()
      .then(response => setRadioModels(response.models))
      .catch(err => {
        const message =
          err instanceof Error ? err.message : 'Failed to load radio models';
        console.error('Failed to load radio models:', err);
        onError?.(message);
      });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // Load user's radios
  const loadRadios = useCallback(async () => {
    setIsRadioLoading(true);
    setRadioError(null);
    try {
      const response = await listRadios();
      setRadios(response.radios);
      if (response.radios.length > 0 && !selectedRadio) {
        setSelectedRadio(response.radios[0]);
      }
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to load radios';
      setRadioError(message);
      onError?.(message);
    } finally {
      setIsRadioLoading(false);
    }
  }, [selectedRadio, onError]);

  useEffect(() => {
    loadRadios();
  }, [loadRadios]);

  // Load backups when radio is selected
  const loadBackups = useCallback(async () => {
    if (!selectedRadio) {
      setBackups([]);
      return;
    }

    setIsBackupsLoading(true);
    setBackupsError(null);
    try {
      const response = await listBackups(selectedRadio.id);
      setBackups(response.backups);
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to load backups';
      setBackupsError(message);
    } finally {
      setIsBackupsLoading(false);
    }
  }, [selectedRadio]);

  useEffect(() => {
    loadBackups();
  }, [loadBackups]);

  // Handle radio creation
  const handleCreateRadio = async (model: RadioModel) => {
    try {
      const params: CreateRadioParams = {
        manufacturer: model.manufacturer,
        model: model.model,
      };
      const newRadio = await createRadio(params);
      setRadios(prev => [newRadio, ...prev]);
      setSelectedRadio(newRadio);
      setShowRadioSelector(false);
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to create radio';
      onError?.(message);
    }
  };

  // Handle radio update
  const handleUpdateRadio = async () => {
    if (!selectedRadio) return;

    try {
      const updated = await updateRadio(selectedRadio.id, {
        firmwareFamily: editFirmware === '' ? undefined : editFirmware,
        notes: editNotes === '' ? undefined : editNotes,
      });
      setSelectedRadio(updated);
      setRadios(prev => prev.map(r => r.id === updated.id ? updated : r));
      setIsEditing(false);
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to update radio';
      onError?.(message);
    }
  };

  // Handle radio deletion
  const handleDeleteRadio = async () => {
    if (!selectedRadio) return;
    if (!confirm('Are you sure you want to delete this radio and all its backups?')) return;

    try {
      await deleteRadio(selectedRadio.id);
      setRadios(prev => prev.filter(r => r.id !== selectedRadio.id));
      setSelectedRadio(radios.length > 1 ? radios.find(r => r.id !== selectedRadio.id) || null : null);
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to delete radio';
      onError?.(message);
    }
  };

  // Handle backup upload
  const handleUploadBackup = async () => {
    if (!selectedRadio || !uploadFile || !uploadName) return;

    setIsUploading(true);
    setUploadProgress(0);

    try {
      const params: CreateRadioBackupParams = {
        backupName: uploadName,
        backupType: uploadType,
        file: uploadFile,
      };

      // Simulate progress since fetch doesn't support progress natively
      const progressInterval = setInterval(() => {
        setUploadProgress(prev => Math.min(prev + 10, 90));
      }, 200);

      try {
        const newBackup = await createBackup(selectedRadio.id, params);

        setUploadProgress(100);

        setBackups(prev => [newBackup, ...prev]);
        setShowUploadModal(false);
        resetUploadForm();
      } finally {
        clearInterval(progressInterval);
      }
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to upload backup';
      onError?.(message);
    } finally {
      setIsUploading(false);
      setUploadProgress(0);
    }
  };

  // Handle backup download
  const handleDownloadBackup = async (backup: RadioBackup) => {
    if (!selectedRadio) return;

    try {
      await downloadBackup(selectedRadio.id, backup.id, backup.fileName);
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to download backup';
      onError?.(message);
    }
  };

  // Handle backup deletion
  const handleDeleteBackup = async (backup: RadioBackup) => {
    if (!selectedRadio) return;
    if (!confirm(`Delete backup "${backup.backupName}"?`)) return;

    try {
      await deleteBackup(selectedRadio.id, backup.id);
      setBackups(prev => prev.filter(b => b.id !== backup.id));
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to delete backup';
      onError?.(message);
    }
  };

  const resetUploadForm = () => {
    setUploadName('');
    setUploadType('full-backup');
    setUploadFile(null);
    if (fileInputRef.current) {
      fileInputRef.current.value = '';
    }
  };

  const startEditing = () => {
    if (selectedRadio) {
      setEditNotes(selectedRadio.notes || '');
      setEditFirmware(selectedRadio.firmwareFamily || '');
      setIsEditing(true);
    }
  };

  // Loading state
  if (isRadioLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-500"></div>
      </div>
    );
  }

  // No radio configured - show selector
  if (radios.length === 0 || showRadioSelector) {
    return (
      <div className="p-6">
        <div className="max-w-2xl mx-auto">
          <div className="text-center mb-8">
            <div className="inline-flex items-center justify-center w-16 h-16 bg-slate-800 rounded-full mb-4">
              <svg className="w-8 h-8 text-primary-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
              </svg>
            </div>
            <h2 className="text-2xl font-bold text-white mb-2">
              {radios.length === 0 ? 'Add Your Radio' : 'Select Radio Model'}
            </h2>
            <p className="text-slate-400">
              Select your radio transmitter to start managing backups
            </p>
          </div>

          {/* Radio model grid */}
          <div className="grid grid-cols-2 md:grid-cols-3 gap-4">
            {radioModels.map(model => (
              <button
                key={model.id}
                onClick={() => handleCreateRadio(model)}
                className="p-4 bg-slate-800 hover:bg-slate-700 border border-slate-700 hover:border-primary-500 rounded-lg text-left transition-all"
              >
                <div className="text-xs text-primary-400 font-medium mb-1">{model.manufacturer}</div>
                <div className="text-white font-semibold">{model.model}</div>
              </button>
            ))}
          </div>

          {radios.length > 0 && (
            <button
              onClick={() => setShowRadioSelector(false)}
              className="mt-6 w-full py-2 text-slate-400 hover:text-white transition-colors"
            >
              Cancel
            </button>
          )}
        </div>
      </div>
    );
  }

  return (
    <div className="flex-1 overflow-y-auto flex flex-col">
      {/* Radio header */}
      <div className="p-6 border-b border-slate-800">
        <div className="flex items-start justify-between">
          <div className="flex items-center gap-4">
            <div className="w-12 h-12 bg-slate-800 rounded-lg flex items-center justify-center">
              <svg className="w-6 h-6 text-primary-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
              </svg>
            </div>
            <div>
              <h2 className="text-xl font-bold text-white">
                {selectedRadio?.manufacturer} {selectedRadio?.model}
              </h2>
              {selectedRadio?.firmwareFamily && (
                <span className="inline-flex items-center px-2 py-0.5 mt-1 rounded text-xs font-medium bg-primary-500/20 text-primary-400">
                  {selectedRadio.firmwareFamily}
                </span>
              )}
            </div>
          </div>

          <div className="flex items-center gap-2">
            <button
              onClick={() => setShowRadioSelector(true)}
              className="px-3 py-1.5 text-sm text-slate-400 hover:text-white border border-slate-700 hover:border-slate-600 rounded-lg transition-colors"
            >
              Change Radio
            </button>
            {!isEditing ? (
              <button
                onClick={startEditing}
                className="px-3 py-1.5 text-sm text-slate-400 hover:text-white border border-slate-700 hover:border-slate-600 rounded-lg transition-colors"
              >
                Edit
              </button>
            ) : (
              <>
                <button
                  onClick={handleUpdateRadio}
                  className="px-3 py-1.5 text-sm text-white bg-primary-600 hover:bg-primary-500 rounded-lg transition-colors"
                >
                  Save
                </button>
                <button
                  onClick={() => setIsEditing(false)}
                  className="px-3 py-1.5 text-sm text-slate-400 hover:text-white transition-colors"
                >
                  Cancel
                </button>
              </>
            )}
            <button
              onClick={handleDeleteRadio}
              className="p-1.5 text-red-400 hover:text-red-300 transition-colors"
              title="Delete radio"
            >
              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
              </svg>
            </button>
          </div>
        </div>

        {/* Edit form */}
        {isEditing && (
          <div className="mt-4 p-4 bg-slate-800/50 rounded-lg space-y-4">
            <div>
              <label className="block text-sm font-medium text-slate-400 mb-1">Firmware</label>
              <select
                value={editFirmware}
                onChange={(e) => setEditFirmware(e.target.value as FirmwareFamily | '')}
                className="w-full px-3 py-2 bg-slate-800 border border-slate-700 rounded-lg text-white focus:outline-none focus:border-primary-500"
              >
                <option value="">Select firmware...</option>
                {FIRMWARE_FAMILIES.map(fw => (
                  <option key={fw.value} value={fw.value}>{fw.label}</option>
                ))}
              </select>
            </div>
            <div>
              <label className="block text-sm font-medium text-slate-400 mb-1">Notes</label>
              <textarea
                value={editNotes}
                onChange={(e) => setEditNotes(e.target.value)}
                placeholder="Add notes about your radio..."
                rows={3}
                className="w-full px-3 py-2 bg-slate-800 border border-slate-700 rounded-lg text-white placeholder-slate-500 focus:outline-none focus:border-primary-500 resize-none"
              />
            </div>
          </div>
        )}

        {/* Notes display (when not editing) */}
        {!isEditing && selectedRadio?.notes && (
          <p className="mt-3 text-sm text-slate-400">{selectedRadio.notes}</p>
        )}
      </div>

      {/* Backups section */}
      <div className="flex-1 overflow-hidden flex flex-col">
        <div className="p-4 border-b border-slate-800 flex items-center justify-between">
          <h3 className="text-lg font-semibold text-white">Configuration Backups</h3>
          <button
            onClick={() => setShowUploadModal(true)}
            className="px-4 py-2 bg-primary-600 hover:bg-primary-500 text-white font-medium rounded-lg transition-colors flex items-center gap-2"
          >
            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12" />
            </svg>
            Upload Backup
          </button>
        </div>

        <div className="flex-1 overflow-auto p-4">
          {isBackupsLoading ? (
            <div className="flex items-center justify-center h-32">
              <div className="animate-spin rounded-full h-6 w-6 border-b-2 border-primary-500"></div>
            </div>
          ) : backupsError ? (
            <div className="text-center py-8 text-red-400">{backupsError}</div>
          ) : backups.length === 0 ? (
            <div className="text-center py-12">
              <div className="inline-flex items-center justify-center w-12 h-12 bg-slate-800 rounded-full mb-3">
                <svg className="w-6 h-6 text-slate-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12" />
                </svg>
              </div>
              <p className="text-slate-400">No backups yet</p>
              <p className="text-sm text-slate-500 mt-1">Upload your first backup to get started</p>
            </div>
          ) : (
            <div className="space-y-3">
              {backups.map(backup => (
                <div
                  key={backup.id}
                  className="p-4 bg-slate-800/50 hover:bg-slate-800 border border-slate-700 rounded-lg transition-colors"
                >
                  <div className="flex items-start justify-between">
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2">
                        <h4 className="font-medium text-white truncate">{backup.backupName}</h4>
                        <span className="px-2 py-0.5 text-xs font-medium bg-slate-700 text-slate-300 rounded">
                          {getBackupTypeLabel(backup.backupType)}
                        </span>
                      </div>
                      <div className="flex items-center gap-4 mt-1 text-sm text-slate-400">
                        <span className="truncate">{backup.fileName}</span>
                        <span>{formatFileSize(backup.fileSize)}</span>
                        <span>{new Date(backup.createdAt).toLocaleDateString()}</span>
                      </div>
                    </div>
                    <div className="flex items-center gap-2 ml-4">
                      <button
                        onClick={() => handleDownloadBackup(backup)}
                        className="p-2 text-slate-400 hover:text-primary-400 transition-colors"
                        title="Download"
                      >
                        <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
                        </svg>
                      </button>
                      <button
                        onClick={() => handleDeleteBackup(backup)}
                        className="p-2 text-slate-400 hover:text-red-400 transition-colors"
                        title="Delete"
                      >
                        <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                        </svg>
                      </button>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>

      {/* Upload Modal */}
      {showUploadModal && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
          <div className="bg-slate-900 border border-slate-700 rounded-xl w-full max-w-md">
            <div className="p-4 border-b border-slate-800 flex items-center justify-between">
              <h3 className="text-lg font-semibold text-white">Upload Backup</h3>
              <button
                onClick={() => { setShowUploadModal(false); resetUploadForm(); }}
                className="text-slate-400 hover:text-white transition-colors"
              >
                <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            </div>

            <div className="p-4 space-y-4">
              {/* Backup name */}
              <div>
                <label className="block text-sm font-medium text-slate-400 mb-1">
                  Backup Name <span className="text-red-400">*</span>
                </label>
                <input
                  type="text"
                  value={uploadName}
                  onChange={(e) => setUploadName(e.target.value)}
                  placeholder="e.g., Pre-update backup"
                  className="w-full px-3 py-2 bg-slate-800 border border-slate-700 rounded-lg text-white placeholder-slate-500 focus:outline-none focus:border-primary-500"
                />
              </div>

              {/* Backup type */}
              <div>
                <label className="block text-sm font-medium text-slate-400 mb-1">Backup Type</label>
                <select
                  value={uploadType}
                  onChange={(e) => setUploadType(e.target.value as BackupType)}
                  className="w-full px-3 py-2 bg-slate-800 border border-slate-700 rounded-lg text-white focus:outline-none focus:border-primary-500"
                >
                  {BACKUP_TYPES.map(type => (
                    <option key={type.value} value={type.value}>{type.label}</option>
                  ))}
                </select>
              </div>

              {/* File input */}
              <div>
                <label className="block text-sm font-medium text-slate-400 mb-1">
                  File <span className="text-red-400">*</span>
                </label>
                <div
                  onClick={() => fileInputRef.current?.click()}
                  className="border-2 border-dashed border-slate-700 hover:border-slate-600 rounded-lg p-6 text-center cursor-pointer transition-colors"
                >
                  {uploadFile ? (
                    <div>
                      <svg className="w-8 h-8 text-primary-400 mx-auto mb-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
                      </svg>
                      <p className="text-white font-medium">{uploadFile.name}</p>
                      <p className="text-sm text-slate-400">{formatFileSize(uploadFile.size)}</p>
                    </div>
                  ) : (
                    <div>
                      <svg className="w-8 h-8 text-slate-500 mx-auto mb-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12" />
                      </svg>
                      <p className="text-slate-400">Click to select file</p>
                      <p className="text-xs text-slate-500 mt-1">ZIP, BIN, YAML, JSON (max 100MB)</p>
                    </div>
                  )}
                </div>
                <input
                  ref={fileInputRef}
                  type="file"
                  onChange={(e) => setUploadFile(e.target.files?.[0] || null)}
                  className="hidden"
                  accept=".zip,.bin,.yaml,.yml,.json,.etx"
                />
              </div>

              {/* Upload progress */}
              {isUploading && (
                <div className="mt-4">
                  <div className="flex items-center justify-between text-sm text-slate-400 mb-1">
                    <span>Uploading...</span>
                    <span>{uploadProgress}%</span>
                  </div>
                  <div className="w-full h-2 bg-slate-700 rounded-full overflow-hidden">
                    <div
                      className="h-full bg-primary-500 transition-all duration-200"
                      style={{ width: `${uploadProgress}%` }}
                    />
                  </div>
                </div>
              )}
            </div>

            <div className="p-4 border-t border-slate-800 flex justify-end gap-3">
              <button
                onClick={handleUploadBackup}
                disabled={!uploadFile || !uploadName || isUploading}
                className="px-4 py-2 bg-primary-600 hover:bg-primary-500 disabled:bg-slate-700 disabled:text-slate-500 text-white font-medium rounded-lg transition-colors"
              >
                {isUploading ? 'Uploading...' : 'Upload'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
