import type { ApiError } from '@/types';
import { getUsageServiceErrorCode } from '@/services/api/usageService';

/**
 * 将 API 错误转换为本地化的用户友好消息
 *
 * @param error - 捕获的错误对象
 * @param t - 翻译函数
 * @returns 本地化的错误消息
 */
export function getLocalizedErrorMessage(
  error: unknown,
  t: (key: string, options?: Record<string, unknown>) => string
): string {
  const usageServiceCode = getUsageServiceErrorCode(error);
  if (usageServiceCode) {
    return t(`usage_service_errors.${usageServiceCode}`, {
      defaultValue: t('usage_service_errors.request_failed'),
    });
  }

  const apiError = error as Partial<ApiError>;
  const status = typeof apiError.status === 'number' ? apiError.status : undefined;
  const code = typeof apiError.code === 'string' ? apiError.code : undefined;
  const message =
    error instanceof Error
      ? error.message
      : typeof apiError.message === 'string'
        ? apiError.message
        : typeof error === 'string'
          ? error
          : '';

  const withHttpStatus = (summary: string) => {
    if (!status) return summary;

    const genericAxiosMessage = `Request failed with status code ${status}`;
    const detail = message.trim();
    const backendDetail =
      detail && detail !== genericAxiosMessage
        ? ` (${t('login.error_backend_detail')}: ${detail})`
        : '';

    return `HTTP ${status}: ${summary}${backendDetail}`;
  };

  if (status === 401) return withHttpStatus(t('login.error_unauthorized'));
  if (status === 403) return withHttpStatus(t('login.error_forbidden'));
  if (status === 404) return withHttpStatus(t('login.error_not_found'));
  if (status && status >= 500) return withHttpStatus(t('login.error_server'));
  if (code === 'ECONNABORTED' || message.toLowerCase().includes('timeout')) {
    return t('login.error_timeout');
  }
  if (code === 'ERR_NETWORK' || message.toLowerCase().includes('network error')) {
    return t('login.error_network');
  }
  if (code === 'ERR_CERT_AUTHORITY_INVALID' || message.toLowerCase().includes('certificate')) {
    return t('login.error_ssl');
  }
  if (message.toLowerCase().includes('cors') || message.toLowerCase().includes('cross-origin')) {
    return t('login.error_cors');
  }

  return withHttpStatus(t('login.error_invalid'));
}
