/**
 * Validates cron expressions
 */

/**
 * Checks if a cron expression is valid
 * @param {string} cronExpr - The cron expression to validate
 * @returns {boolean} - Whether the expression is valid
 */
export function isValidCronExpression(cronExpr) {
    if (!cronExpr || typeof cronExpr !== 'string') {
        return false;
    }

    // Trim whitespace
    cronExpr = cronExpr.trim();

    // Basic format validation (5-6 space-separated fields)
    const parts = cronExpr.split(/\s+/);
    if (parts.length < 5 || parts.length > 6) {
        return false;
    }

    // Validate each part
    try {
        // Minutes: 0-59
        if (!validateCronField(parts[0], 0, 59)) return false;

        // Hours: 0-23
        if (!validateCronField(parts[1], 0, 23)) return false;

        // Day of month: 1-31
        if (!validateCronField(parts[2], 1, 31)) return false;

        // Month: 1-12 or names
        if (!validateCronField(parts[3], 1, 12)) return false;

        // Day of week: 0-7 (0 and 7 both represent Sunday)
        if (!validateCronField(parts[4], 0, 7)) return false;

        // Year: optional
        if (parts.length === 6 && !validateCronField(parts[5], 1970, 2099)) return false;

        return true;
    } catch (e) {
        return false;
    }
}

/**
 * Validates a single field in a cron expression
 * @param {string} field - The cron field to validate
 * @param {number} min - Minimum allowed value
 * @param {number} max - Maximum allowed value
 * @returns {boolean} - Whether the field is valid
 */
function validateCronField(field, min, max) {
    // Handle special characters
    if (field === '*') return true;

    // Handle step values: */2, */5 etc.
    if (field.includes('*/')) {
        const step = parseInt(field.split('/')[1]);
        return !isNaN(step) && step > 0;
    }

    // Handle ranges: 1-5, 10-30 etc.
    if (field.includes('-')) {
        const [start, end] = field.split('-').map(Number);
        return !isNaN(start) && !isNaN(end) &&
            start >= min && end <= max &&
            start <= end;
    }

    // Handle lists: 1,2,3,4
    if (field.includes(',')) {
        return field.split(',').every(value => {
            const num = parseInt(value);
            return !isNaN(num) && num >= min && num <= max;
        });
    }

    // Handle simple numbers
    const num = parseInt(field);
    return !isNaN(num) && num >= min && num <= max;
}

/**
 * Get a human-readable description of a cron expression
 * @param {string} cronExpr - Cron expression
 * @returns {string} - Human readable description or error message
 */
export function describeCronExpression(cronExpr) {
    if (!isValidCronExpression(cronExpr)) {
        return 'Invalid cron expression';
    }

    const expressions = {
        '* * * * *': 'Every minute',
        '0 * * * *': 'Every hour at minute 0',
        '0 0 * * *': 'Every day at 12:00 AM',
        '0 12 * * *': 'Every day at 12:00 PM',
        '0 0 * * 0': 'Every Sunday at 12:00 AM',
        '0 0 * * 1': 'Every Monday at 12:00 AM',
        '0 0 1 * *': 'First day of every month at 12:00 AM',
        '0 0 1 1 *': 'January 1st at 12:00 AM'
    };

    return expressions[cronExpr] || 'Custom schedule';
}

/**
 * Common cron expression presets
 */
export const CRON_PRESETS = [
    { label: 'Every minute', value: '* * * * *' },
    { label: 'Every hour', value: '0 * * * *' },
    { label: 'Every day at midnight', value: '0 0 * * *' },
    { label: 'Every day at noon', value: '0 12 * * *' },
    { label: 'Every Sunday', value: '0 0 * * 0' },
    { label: 'Every Monday', value: '0 0 * * 1' },
    { label: 'First day of month', value: '0 0 1 * *' },
    { label: 'First day of year', value: '0 0 1 1 *' }
];

export default {
    isValidCronExpression,
    describeCronExpression,
    CRON_PRESETS
};