import apiClient from './api';

export default {
    /**
     * Get a list of all jobs
     * @param {string} keyword - Optional search keyword
     * @returns {Promise} - Promise resolving to job list
     */
    listJobs(keyword = '') {
        return apiClient.get('/job/list', { params: { keyword } });
    },

    /**
     * Get a specific job by name
     * @param {string} jobName - Name of the job to retrieve
     * @returns {Promise} - Promise resolving to job details
     */
    getJob(jobName) {
        return apiClient.get(`/job/${jobName}`);
    },

    /**
     * Save a job (create or update)
     * @param {Object} jobData - Job data to save
     * @returns {Promise} - Promise resolving to saved job
     */
    saveJob(jobData) {
        return apiClient.post('/job/save', jobData);
    },

    /**
     * Delete a job
     * @param {string} jobName - Name of the job to delete
     * @returns {Promise} - Promise resolving when job is deleted
     */
    deleteJob(jobName) {
        return apiClient.delete(`/job/${jobName}`);
    },

    /**
     * Enable a job
     * @param {string} jobName - Name of the job to enable
     * @returns {Promise} - Promise resolving when job is enabled
     */
    enableJob(jobName) {
        return apiClient.post(`/job/enable/${jobName}`);
    },

    /**
     * Disable a job
     * @param {string} jobName - Name of the job to disable
     * @returns {Promise} - Promise resolving when job is disabled
     */
    disableJob(jobName) {
        return apiClient.post(`/job/disable/${jobName}`);
    },

    /**
     * Kill a running job
     * @param {string} jobName - Name of the job to kill
     * @returns {Promise} - Promise resolving when kill signal is sent
     */
    killJob(jobName) {
        return apiClient.post(`/job/kill/${jobName}`);
    },

    /**
     * Get a list of currently running jobs
     * @returns {Promise} - Promise resolving to list of running job names
     */
    getRunningJobs() {
        return apiClient.get('/job/running');
    }
};