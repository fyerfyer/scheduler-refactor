import apiClient from './api';

export default {
    /**
     * Get job execution logs
     * @param {string} jobName - Optional job name filter
     * @param {number} page - Page number (starting from 1)
     * @param {number} pageSize - Number of logs per page
     * @returns {Promise} - Promise resolving to log list and pagination info
     */
    listLogs(jobName = '', page = 1, pageSize = 10) {
        return apiClient.get('/log/list', {
            params: { jobName, page, pageSize }
        });
    },

    /**
     * Get the latest log for a specific job
     * @param {string} jobName - Name of the job
     * @returns {Promise} - Promise resolving to log details
     */
    getJobLog(jobName) {
        return apiClient.get(`/log/${jobName}`);
    },

    /**
     * Get a specific log by job name and timestamp
     * @param {string} jobName - Name of the job
     * @param {number} timestamp - Log timestamp
     * @returns {Promise} - Promise resolving to log details
     */
    getLogByTimestamp(jobName, timestamp) {
        return apiClient.get(`/log/${jobName}/${timestamp}`);
    },

    /**
     * Get statistics for a specific job
     * @param {string} jobName - Name of the job
     * @param {number} days - Number of days to include in statistics
     * @returns {Promise} - Promise resolving to job statistics
     */
    getJobStats(jobName, days = 7) {
        return apiClient.get(`/log/stats/${jobName}`, {
            params: { days }
        });
    },

    /**
     * Get overall log statistics
     * @param {number} days - Number of days to include in statistics
     * @returns {Promise} - Promise resolving to overall statistics
     */
    getAllStats(days = 1) {
        return apiClient.get('/log/stats/all', {
            params: { days }
        });
    }
};