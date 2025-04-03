/**
 * Utilities for formatting dates and timestamps consistently throughout the application
 */

/**
 * Format a Unix timestamp to a readable date/time string
 * @param {number} timestamp - Unix timestamp in seconds
 * @param {boolean} includeSeconds - Whether to include seconds in the output
 * @returns {string} - Formatted date string
 */
export function formatTimestamp(timestamp, includeSeconds = true) {
    if (!timestamp) return 'N/A';

    // Convert to milliseconds if needed (handle both second and millisecond timestamps)
    const ts = timestamp > 9999999999 ? timestamp : timestamp * 1000;

    try {
        const date = new Date(ts);
        if (isNaN(date.getTime())) return 'Invalid date';

        const options = {
            year: 'numeric',
            month: 'short',
            day: 'numeric',
            hour: '2-digit',
            minute: '2-digit'
        };

        if (includeSeconds) {
            options.second = '2-digit';
        }

        return date.toLocaleString(undefined, options);
    } catch (error) {
        console.error('Error formatting timestamp:', error);
        return 'Error';
    }
}

/**
 * Format a duration in seconds to a human-readable string
 * @param {number} seconds - Duration in seconds
 * @returns {string} - Formatted duration string
 */
export function formatDuration(seconds) {
    if (typeof seconds !== 'number' || isNaN(seconds) || seconds < 0) {
        return 'N/A';
    }

    if (seconds < 1) {
        return 'Less than a second';
    }

    if (seconds < 60) {
        return `${Math.floor(seconds)} second${seconds !== 1 ? 's' : ''}`;
    }

    if (seconds < 3600) {
        const minutes = Math.floor(seconds / 60);
        const remainingSeconds = Math.floor(seconds % 60);

        let result = `${minutes} minute${minutes !== 1 ? 's' : ''}`;
        if (remainingSeconds > 0) {
            result += ` ${remainingSeconds} second${remainingSeconds !== 1 ? 's' : ''}`;
        }

        return result;
    }

    // More than an hour
    const hours = Math.floor(seconds / 3600);
    const remainingMinutes = Math.floor((seconds % 3600) / 60);

    let result = `${hours} hour${hours !== 1 ? 's' : ''}`;
    if (remainingMinutes > 0) {
        result += ` ${remainingMinutes} minute${remainingMinutes !== 1 ? 's' : ''}`;
    }

    return result;
}

/**
 * Format duration between two timestamps
 * @param {number} startTime - Start timestamp in seconds
 * @param {number} endTime - End timestamp in seconds
 * @returns {string} - Formatted duration
 */
export function formatTimeRange(startTime, endTime) {
    if (!startTime || !endTime) return 'N/A';

    const durationSec = endTime - startTime;
    return formatDuration(durationSec);
}

/**
 * Get a relative time string (e.g., "2 hours ago")
 * @param {number} timestamp - Unix timestamp in seconds
 * @returns {string} - Relative time string
 */
export function getRelativeTime(timestamp) {
    if (!timestamp) return 'N/A';

    // Convert to milliseconds if needed
    const ts = timestamp > 9999999999 ? timestamp : timestamp * 1000;

    try {
        const now = Date.now();
        const date = new Date(ts);
        const diffMs = now - date.getTime();

        // Convert to seconds
        const diffSec = Math.floor(diffMs / 1000);

        if (diffSec < 60) {
            return diffSec <= 0 ? 'Just now' : `${diffSec} second${diffSec !== 1 ? 's' : ''} ago`;
        }

        if (diffSec < 3600) {
            const minutes = Math.floor(diffSec / 60);
            return `${minutes} minute${minutes !== 1 ? 's' : ''} ago`;
        }

        if (diffSec < 86400) {
            const hours = Math.floor(diffSec / 3600);
            return `${hours} hour${hours !== 1 ? 's' : ''} ago`;
        }

        const days = Math.floor(diffSec / 86400);
        if (days < 30) {
            return `${days} day${days !== 1 ? 's' : ''} ago`;
        }

        // For older dates, just return the formatted date
        return formatTimestamp(timestamp, false);

    } catch (error) {
        console.error('Error calculating relative time:', error);
        return 'Error';
    }
}

export default {
    formatTimestamp,
    formatDuration,
    formatTimeRange,
    getRelativeTime
};