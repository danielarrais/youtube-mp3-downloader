import { DownloadItem as DownloadItemType } from '../types';
import { StatusBadge } from './StatusBadge';
import { ProgressBar } from './ProgressBar';
import { api } from '../services/api';
import { useTranslation } from '../hooks/useTranslation';

interface DownloadItemProps {
  item: DownloadItemType;
  onCancel: (id: string) => void;
  onRetry: (id: string) => void;
}

export function DownloadItem({ item, onCancel, onRetry }: DownloadItemProps) {
  const { t } = useTranslation();
  const showProgress = ['downloading', 'converting', 'fetching_info'].includes(item.status);
  const canCancel = ['pending', 'fetching_info', 'downloading', 'converting'].includes(item.status);
  const canRetry = ['failed', 'cancelled'].includes(item.status);
  const canDownload = ['completed', 'skipped'].includes(item.status);

  return (
    <div className="bg-gray-800 rounded-lg p-4 space-y-3">
      <div className="flex items-start justify-between gap-4">
        <div className="flex-1 min-w-0">
          <h3 className="font-medium text-white truncate">
            {item.title || item.url}
          </h3>
          {item.title && (
            <p className="text-sm text-gray-400 truncate">{item.url}</p>
          )}
        </div>
        <StatusBadge status={item.status} />
      </div>

      {showProgress && (
        <div className="space-y-1">
          <ProgressBar percent={item.progress.percent} status={item.status} />
          <div className="flex justify-between text-xs text-gray-400">
            <span>{item.progress.percent.toFixed(1)}%</span>
            {item.progress.speed && <span>{item.progress.speed}</span>}
          </div>
        </div>
      )}

      {item.error && (
        <p className="text-sm text-red-400">{item.error}</p>
      )}

      <div className="flex gap-2">
        {canCancel && (
          <button
            onClick={() => onCancel(item.id)}
            className="text-sm text-gray-400 hover:text-red-400 transition-colors"
          >
            {t.cancel}
          </button>
        )}
        {canRetry && (
          <button
            onClick={() => onRetry(item.id)}
            className="text-sm text-gray-400 hover:text-blue-400 transition-colors"
          >
            {t.retry}
          </button>
        )}
        {canDownload && (
          <button
            onClick={() => api.openFolder(item.file_path!)}
            className="text-sm text-green-400 hover:text-green-300 transition-colors"
          >
            {t.openFolder}
          </button>
        )}
      </div>
    </div>
  );
}
