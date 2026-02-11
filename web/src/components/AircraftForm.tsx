import { useState, useEffect, useRef } from 'react';
import type { Aircraft, CreateAircraftParams, AircraftType } from '../aircraftTypes';
import { AIRCRAFT_TYPES } from '../aircraftTypes';
import {
  moderateAircraftImageUpload,
  saveAircraftImageUpload,
  getAircraftImageUrl,
  type ModerationStatus,
} from '../aircraftApi';

interface AircraftFormProps {
  isOpen: boolean;
  aircraft?: Aircraft | null;
  onClose: () => void;
  onSubmit: (params: CreateAircraftParams) => Promise<Aircraft>;
}

type StatusTone = 'neutral' | 'success' | 'error';

interface PendingAircraftImage {
  previewUrl: string;
  uploadId?: string;
  moderationStatus?: ModerationStatus;
  moderationReason?: string;
}

function revokeBlobUrl(url?: string | null) {
  if (url && url.startsWith('blob:')) {
    URL.revokeObjectURL(url);
  }
}

export function AircraftForm({ isOpen, aircraft, onClose, onSubmit }: AircraftFormProps) {
  const [name, setName] = useState('');
  const [nickname, setNickname] = useState('');
  const [type, setType] = useState<AircraftType>('freestyle');
  const [description, setDescription] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const [existingImagePreview, setExistingImagePreview] = useState<string | null>(null);
  const [selectedImage, setSelectedImage] = useState<PendingAircraftImage | null>(null);
  const [showImageModal, setShowImageModal] = useState(false);
  const [modalImage, setModalImage] = useState<PendingAircraftImage | null>(null);
  const [imageStatusText, setImageStatusText] = useState<string | null>(null);
  const [imageStatusTone, setImageStatusTone] = useState<StatusTone>('neutral');
  const [isImageUploading, setIsImageUploading] = useState(false);
  const [isImageSaving, setIsImageSaving] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const selectedPreviewRef = useRef<string | null>(null);
  const modalPreviewRef = useRef<string | null>(null);

  const isEditing = !!aircraft;

  const replaceSelectedImage = (next: PendingAircraftImage | null) => {
    setSelectedImage((prev) => {
      if (prev?.previewUrl && prev.previewUrl !== next?.previewUrl) {
        revokeBlobUrl(prev.previewUrl);
      }
      return next;
    });
  };

  // Populate form when editing
  useEffect(() => {
    if (aircraft) {
      setName(aircraft.name);
      setNickname(aircraft.nickname || '');
      setType(aircraft.type);
      setDescription(aircraft.description || '');
      setExistingImagePreview(aircraft.hasImage ? getAircraftImageUrl(aircraft.id) : null);
    } else {
      setName('');
      setNickname('');
      setType('freestyle');
      setDescription('');
      setExistingImagePreview(null);
    }

    replaceSelectedImage(null);
    if (modalImage?.previewUrl) {
      revokeBlobUrl(modalImage.previewUrl);
    }
    setModalImage(null);
    setShowImageModal(false);
    setImageStatusText(null);
    setImageStatusTone('neutral');
    setIsImageUploading(false);
    setIsImageSaving(false);
    setError(null);

    if (fileInputRef.current) {
      fileInputRef.current.value = '';
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [aircraft, isOpen]);

  useEffect(() => {
    selectedPreviewRef.current = selectedImage?.previewUrl || null;
  }, [selectedImage?.previewUrl]);

  useEffect(() => {
    modalPreviewRef.current = modalImage?.previewUrl || null;
  }, [modalImage?.previewUrl]);

  // Cleanup object URLs when this component unmounts
  useEffect(() => {
    return () => {
      revokeBlobUrl(selectedPreviewRef.current);
      if (modalPreviewRef.current && modalPreviewRef.current !== selectedPreviewRef.current) {
        revokeBlobUrl(modalPreviewRef.current);
      }
    };
  }, []);

  const displayImagePreview = selectedImage?.previewUrl || existingImagePreview;

  const handleOpenImageModal = () => {
    setShowImageModal(true);
    setError(null);

    if (selectedImage) {
      setModalImage(selectedImage);
      setImageStatusTone('success');
      setImageStatusText('Approved');
    } else {
      setModalImage(null);
      setImageStatusTone('neutral');
      setImageStatusText(null);
    }
  };

  const handleCloseImageModal = () => {
    if (modalImage?.previewUrl && modalImage.previewUrl !== selectedImage?.previewUrl) {
      revokeBlobUrl(modalImage.previewUrl);
    }
    setModalImage(null);
    setShowImageModal(false);
    setImageStatusText(null);
    setImageStatusTone('neutral');
    setIsImageUploading(false);
    setIsImageSaving(false);
    if (fileInputRef.current) {
      fileInputRef.current.value = '';
    }
  };

  const handleImageFileChange = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;

    const allowedTypes = ['image/jpeg', 'image/jpg', 'image/png', 'image/webp'];
    if (!allowedTypes.includes(file.type)) {
      setError('Only JPEG, PNG, and WebP images are allowed');
      return;
    }

    if (file.size > 2 * 1024 * 1024) {
      setError('Image must be less than 2MB');
      return;
    }

    const previewUrl = URL.createObjectURL(file);
    if (modalImage?.previewUrl && modalImage.previewUrl !== selectedImage?.previewUrl) {
      revokeBlobUrl(modalImage.previewUrl);
    }
    setModalImage({ previewUrl });
    setError(null);

    if (fileInputRef.current) {
      fileInputRef.current.value = '';
    }

    try {
      setIsImageUploading(true);
      setImageStatusTone('neutral');
      setImageStatusText('Uploading image…');

      await new Promise((resolve) => setTimeout(resolve, 150));
      setImageStatusText('Checking image for safety…');

      const moderation = await moderateAircraftImageUpload(file);
      if (moderation.status === 'APPROVED' && moderation.uploadId) {
        setModalImage({
          previewUrl,
          uploadId: moderation.uploadId,
          moderationStatus: moderation.status,
          moderationReason: moderation.reason,
        });
        setImageStatusTone('success');
        setImageStatusText('Approved');
      } else if (moderation.status === 'REJECTED') {
        setModalImage({
          previewUrl,
          moderationStatus: moderation.status,
          moderationReason: moderation.reason,
        });
        setImageStatusTone('error');
        setImageStatusText('Not allowed');
      } else {
        setModalImage({
          previewUrl,
          moderationStatus: moderation.status,
          moderationReason: moderation.reason,
        });
        setImageStatusTone('error');
        setImageStatusText('Unable to verify right now');
      }
    } catch (err) {
      setModalImage({
        previewUrl,
        moderationStatus: 'PENDING_REVIEW',
      });
      setImageStatusTone('error');
      setImageStatusText('Unable to verify right now');
      setError(err instanceof Error ? err.message : 'Unable to verify image right now');
    } finally {
      setIsImageUploading(false);
    }
  };

  const handleSaveImageSelection = () => {
    if (!modalImage?.uploadId || modalImage.moderationStatus !== 'APPROVED') {
      return;
    }

    setIsImageSaving(true);
    replaceSelectedImage({
      previewUrl: modalImage.previewUrl,
      uploadId: modalImage.uploadId,
      moderationStatus: modalImage.moderationStatus,
      moderationReason: modalImage.moderationReason,
    });

    setShowImageModal(false);
    setModalImage(null);
    setImageStatusText(null);
    setImageStatusTone('neutral');
    setIsImageSaving(false);
  };

  const handleRemovePendingImage = () => {
    replaceSelectedImage(null);
    setError(null);
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!name.trim()) {
      setError('Name is required');
      return;
    }

    if (selectedImage && (!selectedImage.uploadId || selectedImage.moderationStatus !== 'APPROVED')) {
      setError('Selected image must be approved before saving');
      return;
    }

    setIsSubmitting(true);
    setError(null);

    try {
      const params: CreateAircraftParams = {
        name: name.trim(),
        nickname: nickname.trim() || undefined,
        type,
        description: description.trim() || undefined,
      };

      const savedAircraft = await onSubmit(params);

      if (selectedImage?.uploadId) {
        try {
          await saveAircraftImageUpload(savedAircraft.id, selectedImage.uploadId);
        } catch (imgErr) {
          console.error('Image upload failed:', imgErr);
          setError('Aircraft saved but image upload failed: ' + (imgErr instanceof Error ? imgErr.message : 'Unknown error'));
          setIsSubmitting(false);
          return;
        }
      }

      onClose();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save aircraft');
    } finally {
      setIsSubmitting(false);
    }
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 bg-black/50 backdrop-blur-sm z-50 flex items-center justify-center p-4">
      <div className="bg-slate-800 rounded-xl w-full max-w-lg max-h-[90vh] overflow-y-auto">
        {/* Header */}
        <div className="flex items-center justify-between p-4 border-b border-slate-700">
          <h2 className="text-lg font-semibold text-white">
            {isEditing ? 'Edit Aircraft' : 'Add New Aircraft'}
          </h2>
          <button
            onClick={onClose}
            className="p-1 text-slate-400 hover:text-white transition-colors"
          >
            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        {/* Form */}
        <form onSubmit={handleSubmit} className="p-4 space-y-4">
          {error && (
            <div className="p-3 bg-red-500/10 border border-red-500/20 rounded-lg text-red-400 text-sm">
              {error}
            </div>
          )}

          {/* Name */}
          <div>
            <label className="block text-sm font-medium text-slate-300 mb-1">
              Name <span className="text-red-400">*</span>
            </label>
            <input
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="e.g., Race Quad 5 inch"
              className="w-full px-3 py-2 bg-slate-700 border border-slate-600 rounded-lg text-white placeholder-slate-500 focus:outline-none focus:border-primary-500"
              autoFocus
            />
          </div>

          {/* Nickname */}
          <div>
            <label className="block text-sm font-medium text-slate-300 mb-1">
              Nickname
            </label>
            <input
              type="text"
              value={nickname}
              onChange={(e) => setNickname(e.target.value)}
              placeholder="e.g., Screamer"
              className="w-full px-3 py-2 bg-slate-700 border border-slate-600 rounded-lg text-white placeholder-slate-500 focus:outline-none focus:border-primary-500"
            />
          </div>

          {/* Type */}
          <div>
            <label className="block text-sm font-medium text-slate-300 mb-1">
              Type
            </label>
            <div className="grid grid-cols-3 sm:grid-cols-4 gap-2">
              {AIRCRAFT_TYPES.map((t) => (
                <button
                  key={t.value}
                  type="button"
                  onClick={() => setType(t.value)}
                  className={`flex flex-col items-center p-2 rounded-lg border transition-colors ${
                    type === t.value
                      ? 'bg-primary-600/20 border-primary-500 text-primary-400'
                      : 'bg-slate-700 border-slate-600 text-slate-400 hover:border-slate-500'
                  }`}
                >
                  <span className="text-xl mb-1">{t.icon}</span>
                  <span className="text-xs">{t.label}</span>
                </button>
              ))}
            </div>
          </div>

          {/* Image Upload */}
          <div>
            <label className="block text-sm font-medium text-slate-300 mb-1">
              Image
            </label>
            <div className="flex items-start gap-4">
              <button
                type="button"
                onClick={handleOpenImageModal}
                className="px-4 py-2 bg-slate-700 hover:bg-slate-600 border border-slate-600 rounded-lg text-slate-300 text-sm transition-colors"
              >
                {displayImagePreview ? 'Change Image' : 'Add Image'}
              </button>

              {displayImagePreview && (
                <div className="relative">
                  <div className="w-20 h-20 rounded-lg overflow-hidden bg-slate-700">
                    <img
                      src={displayImagePreview}
                      alt="Preview"
                      className="w-full h-full object-cover"
                    />
                  </div>
                  {selectedImage && (
                    <button
                      type="button"
                      onClick={handleRemovePendingImage}
                      className="absolute -top-2 -right-2 w-5 h-5 bg-red-500 hover:bg-red-600 rounded-full flex items-center justify-center text-white text-xs"
                    >
                      ×
                    </button>
                  )}
                </div>
              )}
            </div>
            <p className="mt-1 text-xs text-slate-500">
              JPEG, PNG, or WebP. Max 2MB. {selectedImage?.uploadId ? 'Approved image ready to save.' : 'Use image modal to run safety checks.'}
            </p>
          </div>

          {/* Description */}
          <div>
            <label className="block text-sm font-medium text-slate-300 mb-1">
              Description
            </label>
            <textarea
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="Notes about this build..."
              rows={3}
              className="w-full px-3 py-2 bg-slate-700 border border-slate-600 rounded-lg text-white placeholder-slate-500 focus:outline-none focus:border-primary-500 resize-none"
            />
          </div>

          {/* Actions */}
          <div className="flex pt-2">
            <button
              type="submit"
              disabled={isSubmitting || isImageUploading || isImageSaving}
              className="w-full px-4 py-2 bg-primary-600 hover:bg-primary-700 disabled:bg-primary-600/50 text-white font-medium rounded-lg transition-colors flex items-center justify-center gap-2"
            >
              {isSubmitting ? (
                <>
                  <div className="animate-spin rounded-full h-4 w-4 border-2 border-white/30 border-t-white" />
                  Saving...
                </>
              ) : isEditing ? (
                'Save Changes'
              ) : (
                'Add Aircraft'
              )}
            </button>
          </div>
        </form>
      </div>

      {/* Image Moderation Modal */}
      {showImageModal && (
        <div className="fixed inset-0 bg-black/60 flex items-center justify-center z-[60] p-4">
          <div className="bg-slate-800 rounded-xl p-6 max-w-md w-full shadow-2xl border border-slate-700">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-lg font-semibold text-white">Edit Aircraft Image</h3>
              <button
                type="button"
                onClick={handleCloseImageModal}
                disabled={isImageSaving}
                className="p-2 text-slate-400 hover:text-white hover:bg-slate-700 rounded-lg transition-colors disabled:opacity-50"
              >
                <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            </div>

            <div className="flex flex-col items-center gap-4 mb-4">
              <div className="w-36 h-36 rounded-lg overflow-hidden border-2 border-slate-600 bg-slate-700">
                {modalImage?.previewUrl ? (
                  <img src={modalImage.previewUrl} alt="Aircraft preview" className="w-full h-full object-cover" />
                ) : displayImagePreview ? (
                  <img src={displayImagePreview} alt="Current aircraft image" className="w-full h-full object-cover" />
                ) : (
                  <div className="w-full h-full flex items-center justify-center text-slate-500 text-4xl">🚁</div>
                )}
              </div>

              <input
                ref={fileInputRef}
                type="file"
                accept="image/jpeg,image/jpg,image/png,image/webp"
                onChange={handleImageFileChange}
                className="hidden"
              />
              <button
                type="button"
                onClick={() => fileInputRef.current?.click()}
                disabled={isImageUploading || isImageSaving}
                className="px-4 py-2 bg-slate-700 hover:bg-slate-600 text-white rounded-lg text-sm transition-colors disabled:opacity-50"
              >
                {modalImage?.previewUrl ? 'Choose Different' : 'Select Image'}
              </button>
              <p className="text-xs text-slate-500">JPEG, PNG, or WebP. Max 2MB.</p>
            </div>

            {imageStatusText && (
              <div
                className={`mb-4 p-3 rounded-lg text-sm border ${
                  imageStatusTone === 'success'
                    ? 'bg-green-500/10 border-green-500/30 text-green-400'
                    : imageStatusTone === 'error'
                      ? 'bg-red-500/10 border-red-500/30 text-red-400'
                      : 'bg-slate-700/50 border-slate-600 text-slate-300'
                }`}
              >
                <p>{imageStatusText}</p>
                {modalImage?.moderationReason && imageStatusTone !== 'success' && (
                  <p className="mt-1 text-xs text-slate-300">{modalImage.moderationReason}</p>
                )}
              </div>
            )}

            <div className="flex gap-2">
              <button
                type="button"
                onClick={handleCloseImageModal}
                disabled={isImageSaving}
                className="flex-1 px-4 py-2 bg-slate-700 hover:bg-slate-600 text-white rounded-lg text-sm transition-colors disabled:opacity-50"
              >
                Cancel
              </button>
              <button
                type="button"
                onClick={handleSaveImageSelection}
                disabled={
                  isImageSaving ||
                  isImageUploading ||
                  !modalImage?.uploadId ||
                  modalImage.moderationStatus !== 'APPROVED'
                }
                className="flex-1 px-4 py-2 bg-primary-600 hover:bg-primary-500 text-white rounded-lg text-sm transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
              >
                Save
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
