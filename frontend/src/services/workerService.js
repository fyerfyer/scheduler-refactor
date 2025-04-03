import apiClient from './api';

export default {
    /**
     * Get all worker nodes
     * @returns {Promise} - Promise resolving to worker list
     */
    listWorkers() {
        return apiClient.get('/worker/list');
    },

    /**
     * Get worker statistics
     * @returns {Promise} - Promise resolving to worker statistics
     */
    getWorkerStats() {
        return apiClient.get('/worker/stats');
    }
};